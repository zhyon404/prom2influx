package main

import (
	"context"
	"fmt"
	"github.com/influxdata/influxdb1-client"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/zhyon404/prom2influx"
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type config struct {
	influxdbURL      string
	prometheusURL    string
	influxdbDatabase string
	start            string
	end              string
	step             time.Duration
	c                int
	retry            int
}

func parseFlags() *config {
	a := kingpin.New(filepath.Base(os.Args[0]), "Remote storage adapter")
	a.HelpFlag.Short('h')

	cfg := &config{}

	a.Flag("influxdb-url", "The URL of the remote InfluxDB server to send samples to. None, if empty.").
		Default("").StringVar(&cfg.influxdbURL)
	a.Flag("prometheus-url", "The URL of the remote prometheus server to read samples to. None, if empty.").
		Default("").StringVar(&cfg.prometheusURL)
	a.Flag("influxdb.database", "The name of the database to use for storing samples in InfluxDB.").
		Default("prometheus").StringVar(&cfg.influxdbDatabase)
	a.Flag("start", "The time start.").
		Default("").StringVar(&cfg.start)
	a.Flag("end", "The time end").
		Default("").StringVar(&cfg.end)
	a.Flag("step", "The step").
		Default("1m").DurationVar(&cfg.step)
	a.Flag("c", "The connections").
		Default("1").IntVar(&cfg.c)
	a.Flag("retry", "The retry").
		Default("3").IntVar(&cfg.retry)

	_, err := a.Parse(os.Args[1:])
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		a.Usage(os.Args[1:])
		os.Exit(2)
	}

	return cfg
}

func main() {
	cfg := parseFlags()
	host, err := url.Parse(cfg.influxdbURL)
	if err != nil {
		log.Fatal(err)
	}
	start, err := time.Parse(cfg.start, time.RFC3339)
	if err != nil {
		log.Println(err)
	}
	end, err := time.Parse(cfg.end, time.RFC3339)
	if err != nil {
		log.Println(err)
	}
	// NOTE: this assumes you've setup a user and have setup shell env variables,
	// namely INFLUX_USER/INFLUX_PWD. If not just omit Username/Password below.
	conf := client.Config{
		URL:      *host,
		Username: os.Getenv("INFLUX_USER"),
		Password: os.Getenv("INFLUX_PWD"),
	}
	con, err := client.NewClient(conf)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connection", con)
	c, _ := api.NewClient(api.Config{
		Address: cfg.prometheusURL,
	})
	api := v1.NewAPI(c)
	t := prom2influx.NewTrans(cfg.influxdbDatabase, start, end, cfg.step, api, con, cfg.c, cfg.retry)
	log.Fatalln(t.Run(context.Background()))
}
