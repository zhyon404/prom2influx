# prom2influx
migrate historical data from Prometheus to InfluxDB

## Building prom2influx

You can use go to retrieve all the dependencies and then build an executable
```
cd cmd/prom2influx
go get -u github.com/influxdata/influxdb1-client
go get -u github.com/pkg/errors
go get -u github.com/prometheus/client_golang/api
go get -u github.com/prometheus/client_golang/api/prometheus/v1
go get -u github.com/zhyon404/prom2influx
go get -u gopkg.in/alecthomas/kingpin.v2
go build
```

## Usage
```
usage: prom2influx [<flags>]

Remote storage adapter

Flags:
  -h, --help               Show context-sensitive help (also try --help-long and
                           --help-man).
      --influxdb-url=""    The URL of the remote InfluxDB server to send samples
                           to. None, if empty.
      --prometheus-url=""  The URL of the remote prometheus server to read
                           samples to. None, if empty.
      --influxdb.database="prometheus"
                           The name of the database to use for storing samples
                           in InfluxDB.
      --start=""           The time start.
      --end=""             The time end
      --step=1m            The step
      --c=1                The connections
      --retry=3            The retry
```

an example would be:

```
prom2influx --influxdb-url="http://{IP}:8086" --prometheus-url="http://{IP}:9090" --influxdb.database="Demo"
```