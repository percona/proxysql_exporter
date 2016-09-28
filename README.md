# proxysql_exporter

Prometheus exporter for [ProxySQL](https://github.com/sysown/proxysql) performance data.

## Collectors

 * Global status metrics from `SHOW MYSQL STATUS`
 * Connection pool usage statistics from `stats_mysql_connection_pool`

## Build

```
GOOS=linux go build proxysql_exporter.go
```

## Run

```
export DATA_SOURCE_NAME="admin:admin@tcp(localhost:6032)/"
./proxysql_exporter
```

Note, for some reason it does not work with "stats:stats" credentials despite you can run the queries executed by the
exporter successfully using mysql cli (`Error 1045: no such table: global_variables`).

## Visualize

There is a Grafana dashboard for ProxySQL available as a part of [PMM](https://www.percona.com/doc/percona-monitoring-and-management/index.html) project, you can see the demo [here](https://pmmdemo.percona.com/graph/dashboard/db/proxysql-overview).
