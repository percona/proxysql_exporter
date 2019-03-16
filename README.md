# Percona ProxySQL Exporter

[![Release](https://github-release-version.herokuapp.com/github/percona/proxysql_exporter/release.svg?style=flat)](https://github.com/percona/proxysql_exporter/releases/latest)
[![Build Status](https://travis-ci.org/percona/proxysql_exporter.svg?branch=master)](https://travis-ci.org/percona/proxysql_exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/percona/proxysql_exporter)](https://goreportcard.com/report/github.com/percona/proxysql_exporter)
[![CLA assistant](https://cla-assistant.percona.com/readme/badge/percona/proxysql_exporter)](https://cla-assistant.percona.com/percona/proxysql_exporter)
Prometheus exporter for [ProxySQL](https://github.com/sysown/proxysql) performance data.
Supported versions: 1.2 and 1.3.


## Building and running

### Building

```bash
make
```


### Running

The MySQL server's data source name must be set via the `DATA_SOURCE_NAME` environment variable. The format of this
variable is described at https://github.com/go-sql-driver/mysql#dsn-data-source-name.

To enable HTTP basic authentication, set environment variable `HTTP_AUTH` to user:password pair. Alternatively, you can
use YAML file with `server_user` and `server_password` fields.

```bash
export DATA_SOURCE_NAME='stats:stats@tcp(127.0.0.1:42004)/'
export HTTP_AUTH='user:password'
./proxysql_exporter <flags>
```

Note, using `stats` user requires ProxySQL 1.2.4 or higher. Otherwise, use `admin` user.


### Collector Flags

Name                          | Description
------------------------------|------------
collect.mysql_connection_pool | Collect from stats_mysql_connection_pool.
collect.mysql_connection_list | Collect connection list from stats_mysql_processlist.
collect.mysql_status          | Collect from stats_mysql_global (SHOW MYSQL STATUS).


### General Flags

Name                                       | Description
-------------------------------------------|--------------------------------------------------------------------------------------------------
log.format                                 | Set the log target and format. Example: "logger:syslog?appname=bob&local=7" or "logger:stdout?json=true" (default "logger:stderr")
log.level                                  | Only log messages with the given severity or above. Valid levels: [debug, info, warn, error, fatal]
version                                    | Print version information and exit.
web.auth-file                              | Path to YAML file with server_user, server_password options for http basic auth (overrides HTTP_AUTH env var).
web.listen-address                         | Address to listen on for web interface and telemetry. (default ":42004")
web.ssl-cert-file                          | Path to SSL certificate file.
web.ssl-key-file                           | Path to SSL key file.
web.telemetry-path                         | Path under which to expose metrics. (default "/metrics")


## Visualizing

There is a Grafana dashboard for ProxySQL available as a part of [PMM](https://www.percona.com/doc/percona-monitoring-and-management/index.html) project, you can see the demo [here](https://pmmdemo.percona.com/graph/dashboard/db/proxysql-overview).


## Submitting Bug Reports

If you find a bug in Percona ProxySQL Exporter or one of the related projects, you should submit a report to that project's [JIRA](https://jira.percona.com) issue tracker.

Your first step should be [to search](https://jira.percona.com/issues/?jql=project=PMM%20AND%20component=ProxySQL_Exporter) the existing set of open tickets for a similar report. If you find that someone else has already reported your problem, then you can upvote that report to increase its visibility.

If there is no existing report, submit a report following these steps:

1. [Sign in to Percona JIRA.](https://jira.percona.com/login.jsp) You will need to create an account if you do not have one.
2. [Go to the Create Issue screen and select the relevant project.](https://jira.percona.com/secure/CreateIssueDetails!init.jspa?pid=11600&issuetype=1&priority=3&components=11601)
3. Fill in the fields of Summary, Description, Steps To Reproduce, and Affects Version to the best you can. If the bug corresponds to a crash, attach the stack trace from the logs.

An excellent resource is [Elika Etemad's article on filing good bug reports.](http://fantasai.inkedblade.net/style/talks/filing-good-bugs/).

As a general rule of thumb, please try to create bug reports that are:

- *Reproducible.* Include steps to reproduce the problem.
- *Specific.* Include as much detail as possible: which version, what environment, etc.
- *Unique.* Do not duplicate existing tickets.
- *Scoped to a Single Bug.* One bug per report.
