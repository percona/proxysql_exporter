# Percona ProxySQL Exporter

[![Release](https://github-release-version.herokuapp.com/github/percona/proxysql_exporter/release.svg?style=flat)](https://github.com/percona/proxysql_exporter/releases/latest)
[![Build Status](https://travis-ci.org/percona/proxysql_exporter.svg?branch=master)](https://travis-ci.org/percona/proxysql_exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/percona/proxysql_exporter)](https://goreportcard.com/report/github.com/percona/proxysql_exporter)
[![CLA assistant](https://cla-assistant.io/readme/badge/percona/proxysql_exporter)](https://cla-assistant.io/percona/proxysql_exporter)

Prometheus exporter for [ProxySQL](https://github.com/sysown/proxysql) performance data.

## Collectors

 * Global status metrics from `SHOW MYSQL STATUS`.
 * Connection pool usage statistics from `stats_mysql_connection_pool`.

## Setting the MySQL server's data source name

The MySQL server's data source name must be set via the `DATA_SOURCE_NAME` environment variable. The format of this variable is described at https://github.com/go-sql-driver/mysql#dsn-data-source-name.

## Building

1. Setup [`GOPATH`](https://golang.org/doc/code.html#GOPATH).
2. Clone repository to `GOPATH`: `go get -v -d github.com/percona/proxysql_exporter`.
3. Install released version of Glide: `curl https://glide.sh/get | sh` or `brew install glide`.
4. Fetch dependencies: `glide install`.
5. Install exporter: `go install -v`. Binary will be created in `$GOPATH/bin`.


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

## Submit Bug Report
If you find a bug in Percona ProxySQL Exporter or one of the related projects, you should submit a report to that project's [JIRA](https://jira.percona.com) issue tracker.

Your first step should be to search the existing set of open tickets for a similar report. If you find that someone else has already reported your problem, then you can upvote that report to increase its visibility.

If there is no existing report, submit a report following these steps:

1.  [Sign in to Percona JIRA.](https://jira.percona.com/login.jsp) You will need to create an account if you do not have one.
2.  [Go to the Create Issue screen and select the relevant project.](https://jira.percona.com/secure/CreateIssueDetails!init.jspa?pid=11600&issuetype=1&priority=3&components=11601)
3.  Fill in the fields of Summary, Description, Steps To Reproduce, and Affects Version to the best you can. If the bug corresponds to a crash, attach the stack trace from the logs.

An excellent resource is [Elika Etemad's article on filing good bug reports.](http://fantasai.inkedblade.net/style/talks/filing-good-bugs/).

As a general rule of thumb, please try to create bug reports that are:

-   *Reproducible.* Include steps to reproduce the problem.
-   *Specific.* Include as much detail as possible: which version, what environment, etc.
-   *Unique.* Do not duplicate existing tickets.
-   *Scoped to a Single Bug.* One bug per report.
