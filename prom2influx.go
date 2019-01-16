package prom2influx

import (
	"context"
	"encoding/json"
	"github.com/influxdata/influxdb1-client"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"log"
	"strconv"
	"sync"
	"time"
	"unsafe"
)

func NewTrans(database string, start, end time.Time, step time.Duration, p v1.API, i *client.Client, c, retry int) *Trans {
	return &Trans{
		Database: database,
		Start:    start,
		End:      end,
		Step:     step,
		p:        p,
		i:        i,
		C:        c,
		Retry:    retry,
	}
}

type Trans struct {
	Database string
	Start    time.Time
	End      time.Time
	Step     time.Duration
	p        v1.API
	i        *client.Client
	C        int
	Retry    int
}

func (t *Trans) Run(ctx context.Context) error {
	names, err := t.p.LabelValues(ctx, "__name__")
	if err != nil {
		return err
	}
	if t.End.IsZero() {
		t.End = time.Now()
	}
	if t.Step == 0 {
		t.Step = time.Minute
	}
	if t.Start.IsZero() {
		flags, err := t.p.Flags(ctx)
		if err != nil {
			return err
		}
		v, ok := flags["storage.tsdb.retention"]
		if !ok {
			panic("storage.tsdb.retention not found")
		}
		if v[len(v)-1] == 'd' {
			a, err := strconv.Atoi(v[0 : len(v)-1])
			if err != nil {
				return err
			}
			v = strconv.Itoa(a*24) + "h"
		}
		d, err := time.ParseDuration(v)
		if err != nil {
			return err
		}
		t.Start = time.Now().Add(-d)
	}
	if err != nil {
		return err
	}
	if t.C == 0 {
		t.C = 1
	}
	v, _ := json.Marshal(t)
	log.Println(string(v))
	c := make(chan struct{}, t.C)
	wg := sync.WaitGroup{}
	wg.Add(len(names))
	for _, i := range names {
		c <- struct{}{}
		go func(i string) {
			log.Println("start ", i)
			err := t.runOne(string(i))
			if err != nil {
				log.Fatal(err)
			}
			log.Println("done ", i)
			wg.Done()
			<-c
		}(string(i))
	}
	wg.Wait()
	return nil
}

func (t *Trans) runOne(name string) error {
	start := t.Start
	finish := t.End
	for start.Before(finish) {
		end := start.Add(t.Step * 60 * 1)
		log.Println("one...", start.Format(time.RFC3339), end.Format(time.RFC3339))
		ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
		v, err := t.p.QueryRange(ctx, name, v1.Range{
			Start: start,
			End:   end,
			Step:  t.Step,
		})
		if err != nil {
			panic("")
			return err
		}
		bps := t.valueToInfluxdb(name, v)
		for _, i := range bps {
			var err error
			for try := 0; try <= t.Retry; try++ {
				_, err = t.i.Write(i)
				if err == nil {
					break
				}
			}
			if err != nil {
				panic(err)
				return err
			}
		}
		start = end
	}
	return nil
}

var externalLabels = map[string]string{"monitor": "codelab-monitor"}

func (t *Trans) valueToInfluxdb(name string, v model.Value) (bps []client.BatchPoints) {
	switch v.(type) {
	case model.Matrix:
		v := v.(model.Matrix)
		for _, i := range v {
			bp := client.BatchPoints{
				Database:  t.Database,
				Tags:      metricToTag(i.Metric),
				Precision: "n",
			}
			for _, j := range i.Values {
				t := j.Timestamp.Time()
				bp.Points = append(bp.Points, client.Point{
					Tags:        externalLabels,
					Measurement: name,
					Time:        t,
					Fields:      map[string]interface{}{"value": float64(j.Value)},
				})
			}
			bps = append(bps, bp)
		}
	case *model.Scalar:
		v := v.(*model.Scalar)
		bps = append(bps, client.BatchPoints{
			Points: []client.Point{{
				Measurement: name,
				Tags:        externalLabels,
				Fields:      map[string]interface{}{"value": float64(v.Value)},
				Precision:   "n",
			}},
			Database: t.Database,
			Time:     v.Timestamp.Time().Add(-(time.Hour * 24 * 15)),
		})
	case model.Vector:
		v := v.(model.Vector)
		bp := client.BatchPoints{
			Database:  t.Database,
			Precision: "n",
		}
		for _, i := range v {
			tags := metricToTag(i.Metric)
			for k, v := range externalLabels {
				tags[k] = v
			}
			bp.Points = append(bp.Points, client.Point{
				Measurement: name,
				Tags:        tags,
				Time:        i.Timestamp.Time(),
				Fields:      map[string]interface{}{"value": i.Value},
			})
		}
		bps = append(bps, bp)
	case *model.String:
		v := v.(*model.String)
		bps = append(bps, client.BatchPoints{
			Points: []client.Point{{
				Measurement: name,
				Tags:        externalLabels,

				Fields:    map[string]interface{}{"value": string(v.Value)},
				Precision: "ms",
			}},
			Database: t.Database,
			Time:     v.Timestamp.Time(),
		})
	default:
		panic("unknown type")
	}
	return
}

func metricToTag(metric model.Metric) map[string]string {
	return *(*map[string]string)(unsafe.Pointer(&metric))
}
