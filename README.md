# proxysql_exporter

[![Build Status](https://travis-ci.org/percona/proxysql_exporter.svg?branch=master)](https://travis-ci.org/percona/proxysql_exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/percona/proxysql_exporter)](https://goreportcard.com/report/github.com/percona/proxysql_exporter)

Prometheus exporter for [ProxySQL](https://github.com/sysown/proxysql) performance data.

## Collectors

 * Global status metrics from `SHOW MYSQL STATUS`
 * Connection pool usage statistics from `stats_mysql_connection_pool`

## Build

```
GOOS=linux go build proxysql_exporter.go
```

## Usage

```
export DATA_SOURCE_NAME="stats:stats@tcp(localhost:6032)/"
./proxysql_exporter
```

To enable HTTP basic authentication, set environment variable `HTTP_AUTH` to user:password pair.
For example: `export HTTP_AUTH="user:password"`

Note, using `stats` user requires ProxySQL 1.2.4 or higher. Otherwise, use `admin` user.

## Visualize

There is a Grafana dashboard for ProxySQL available as a part of [PMM](https://www.percona.com/doc/percona-monitoring-and-management/index.html) project, you can see the demo [here](https://pmmdemo.percona.com/graph/dashboard/db/proxysql-overview).
