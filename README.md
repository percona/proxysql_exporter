# proxysql_exporter

Prometheus exporter for [ProxySQL](https://github.com/sysown/proxysql) performance data.

## Features

 * Global status statistics from `stats.stats_mysql_global` (`SHOW MYSQL STATUS`)
 * Connection pool usage metrics from `stats.stats_mysql_connection_pool`

Note, if you name your MySQL endpoint via `comment` field in `mysql_servers` table, it will be set to the label
`endpoint` for all connection pool metrics. Otherwise, this label will be set to `hostname:port`.

## Build

```
GOOS=linux go build proxysql_exporter.go
```

## Run

```
export DATA_SOURCE_NAME="admin:admin@tcp(localhost:6032)/"
./proxysql_exporter
```
