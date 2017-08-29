// Copyright 2016-2017 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"strings"
	"testing"
	"time"

	"github.com/percona/exporter_shared/helpers"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/smartystreets/goconvey/convey"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func sanitizeQuery(q string) string {
	q = strings.Join(strings.Fields(q), " ")
	q = strings.Replace(q, "(", "\\(", -1)
	q = strings.Replace(q, ")", "\\)", -1)
	q = strings.Replace(q, "*", "\\*", -1)
	return q
}

func TestScrapeMySQLGlobal(t *testing.T) {
	convey.Convey("Metrics are lowercase", t, convey.FailureContinues, func() {
		for c, m := range mySQLGlobalMetrics {
			convey.So(c, convey.ShouldEqual, strings.ToLower(c))
			convey.So(m.name, convey.ShouldEqual, strings.ToLower(m.name))
		}
	})

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error opening a stub database connection: %s", err)
	}
	defer db.Close()

	columns := []string{"Variable_Name", "Variable_Value"}
	rows := sqlmock.NewRows(columns).
		AddRow("Active_Transactions", "3").
		AddRow("Backend_query_time_nsec", "76355784684851").
		AddRow("Client_Connections_aborted", "0").
		AddRow("Client_Connections_connected", "64").
		AddRow("Client_Connections_created", "1087931").
		AddRow("Servers_table_version", "2019470")
	mock.ExpectQuery(mySQLGlobalQuery).WillReturnRows(rows)

	ch := make(chan prometheus.Metric)
	go func() {
		if err = scrapeMySQLGlobal(db, ch); err != nil {
			t.Errorf("error calling function on test: %s", err)
		}
		close(ch)
	}()

	counterExpected := []helpers.Metric{
		{"proxysql_mysql_status_active_transactions", prometheus.Labels{}, dto.MetricType_GAUGE, 3},
		{"proxysql_mysql_status_backend_query_time_nsec", prometheus.Labels{}, dto.MetricType_UNTYPED, 76355784684851},
		{"proxysql_mysql_status_client_connections_aborted", prometheus.Labels{}, dto.MetricType_COUNTER, 0},
		{"proxysql_mysql_status_client_connections_connected", prometheus.Labels{}, dto.MetricType_GAUGE, 64},
		{"proxysql_mysql_status_client_connections_created", prometheus.Labels{}, dto.MetricType_COUNTER, 1087931},
		{"proxysql_mysql_status_servers_table_version", prometheus.Labels{}, dto.MetricType_UNTYPED, 2019470},
	}
	convey.Convey("Metrics comparison", t, convey.FailureContinues, func() {
		for _, expect := range counterExpected {
			got := *helpers.ReadMetric(<-ch)
			convey.So(got, convey.ShouldResemble, expect)
		}
	})

	// Ensure all SQL queries were executed
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestScrapeMySQLConnectionPool(t *testing.T) {
	convey.Convey("Metrics are lowercase", t, convey.FailureContinues, func() {
		for c, m := range mySQLconnectionPoolMetrics {
			convey.So(c, convey.ShouldEqual, strings.ToLower(c))
			convey.So(m.name, convey.ShouldEqual, strings.ToLower(m.name))
		}
	})

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error opening a stub database connection: %s", err)
	}
	defer db.Close()

	columns := []string{"hostgroup", "srv_host", "srv_port", "status", "ConnUsed", "ConnFree", "ConnOK", "ConnERR",
		"Queries", "Bytes_data_sent", "Bytes_data_recv", "Latency_us"}
	rows := sqlmock.NewRows(columns).
		AddRow("0", "10.91.142.80", "3306", "ONLINE", "0", "45", "1895677", "46", "197941647", "10984550806", "321063484988", "163").
		AddRow("0", "10.91.142.82", "3306", "SHUNNED", "0", "97", "39859", "0", "386686994", "21643682247", "641406745151", "255").
		AddRow("1", "10.91.142.88", "3306", "OFFLINE_SOFT", "0", "18", "31471", "6391", "255993467", "14327840185", "420795691329", "283").
		AddRow("2", "10.91.142.89", "3306", "OFFLINE_HARD", "0", "18", "31471", "6391", "255993467", "14327840185", "420795691329", "283")
	mock.ExpectQuery(sanitizeQuery(mySQLconnectionPoolQuery)).WillReturnRows(rows)

	ch := make(chan prometheus.Metric)
	go func() {
		if err = scrapeMySQLConnectionPool(db, ch); err != nil {
			t.Errorf("error calling function on test: %s", err)
		}
		close(ch)
	}()

	counterExpected := []helpers.Metric{
		{"proxysql_connection_pool_status", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, dto.MetricType_GAUGE, 1},
		{"proxysql_connection_pool_conn_used", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, dto.MetricType_GAUGE, 0},
		{"proxysql_connection_pool_conn_free", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, dto.MetricType_GAUGE, 45},
		{"proxysql_connection_pool_conn_ok", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, dto.MetricType_COUNTER, 1895677},
		{"proxysql_connection_pool_conn_err", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, dto.MetricType_COUNTER, 46},
		{"proxysql_connection_pool_queries", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, dto.MetricType_COUNTER, 197941647},
		{"proxysql_connection_pool_bytes_data_sent", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, dto.MetricType_COUNTER, 10984550806},
		{"proxysql_connection_pool_bytes_data_recv", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, dto.MetricType_COUNTER, 321063484988},
		{"proxysql_connection_pool_latency_us", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, dto.MetricType_GAUGE, 163},

		{"proxysql_connection_pool_status", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, dto.MetricType_GAUGE, 2},
		{"proxysql_connection_pool_conn_used", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, dto.MetricType_GAUGE, 0},
		{"proxysql_connection_pool_conn_free", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, dto.MetricType_GAUGE, 97},
		{"proxysql_connection_pool_conn_ok", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, dto.MetricType_COUNTER, 39859},
		{"proxysql_connection_pool_conn_err", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, dto.MetricType_COUNTER, 0},
		{"proxysql_connection_pool_queries", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, dto.MetricType_COUNTER, 386686994},
		{"proxysql_connection_pool_bytes_data_sent", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, dto.MetricType_COUNTER, 21643682247},
		{"proxysql_connection_pool_bytes_data_recv", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, dto.MetricType_COUNTER, 641406745151},
		{"proxysql_connection_pool_latency_us", prometheus.Labels{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, dto.MetricType_GAUGE, 255},

		{"proxysql_connection_pool_status", prometheus.Labels{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, dto.MetricType_GAUGE, 3},
		{"proxysql_connection_pool_conn_used", prometheus.Labels{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, dto.MetricType_GAUGE, 0},
		{"proxysql_connection_pool_conn_free", prometheus.Labels{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, dto.MetricType_GAUGE, 18},
		{"proxysql_connection_pool_conn_ok", prometheus.Labels{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, dto.MetricType_COUNTER, 31471},
		{"proxysql_connection_pool_conn_err", prometheus.Labels{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, dto.MetricType_COUNTER, 6391},
		{"proxysql_connection_pool_queries", prometheus.Labels{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, dto.MetricType_COUNTER, 255993467},
		{"proxysql_connection_pool_bytes_data_sent", prometheus.Labels{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, dto.MetricType_COUNTER, 14327840185},
		{"proxysql_connection_pool_bytes_data_recv", prometheus.Labels{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, dto.MetricType_COUNTER, 420795691329},
		{"proxysql_connection_pool_latency_us", prometheus.Labels{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, dto.MetricType_GAUGE, 283},

		{"proxysql_connection_pool_status", prometheus.Labels{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, dto.MetricType_GAUGE, 4},
		{"proxysql_connection_pool_conn_used", prometheus.Labels{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, dto.MetricType_GAUGE, 0},
		{"proxysql_connection_pool_conn_free", prometheus.Labels{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, dto.MetricType_GAUGE, 18},
		{"proxysql_connection_pool_conn_ok", prometheus.Labels{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, dto.MetricType_COUNTER, 31471},
		{"proxysql_connection_pool_conn_err", prometheus.Labels{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, dto.MetricType_COUNTER, 6391},
		{"proxysql_connection_pool_queries", prometheus.Labels{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, dto.MetricType_COUNTER, 255993467},
		{"proxysql_connection_pool_bytes_data_sent", prometheus.Labels{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, dto.MetricType_COUNTER, 14327840185},
		{"proxysql_connection_pool_bytes_data_recv", prometheus.Labels{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, dto.MetricType_COUNTER, 420795691329},
		{"proxysql_connection_pool_latency_us", prometheus.Labels{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, dto.MetricType_GAUGE, 283},
	}
	convey.Convey("Metrics comparison", t, convey.FailureContinues, func() {
		for _, expect := range counterExpected {
			got := *helpers.ReadMetric(<-ch)
			convey.So(got, convey.ShouldResemble, expect)
		}
	})

	// Ensure all SQL queries were executed
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestExporter(t *testing.T) {
	if testing.Short() {
		t.Skip("-short is passed, skipping integration test")
	}

	// wait up to 30 seconds for ProxySQL to become available
	exporter := NewExporter("admin:admin@tcp(127.0.0.1:16032)/", true, true)
	for i := 0; i < 30; i++ {
		db, err := exporter.db()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		// configure ProxySQL
		for _, q := range strings.Split(`
DELETE FROM mysql_servers;
INSERT INTO mysql_servers(hostgroup_id, hostname, port) VALUES (1, 'mysql', 3306);
INSERT INTO mysql_servers(hostgroup_id, hostname, port) VALUES (1, 'percona-server', 3306);
LOAD MYSQL SERVERS TO RUNTIME;
SAVE MYSQL SERVERS TO DISK;

DELETE FROM mysql_users;
INSERT INTO mysql_users(username, password, default_hostgroup) VALUES ('root', '', 1);
INSERT INTO mysql_users(username, password, default_hostgroup) VALUES ('monitor', 'monitor', 1);
LOAD MYSQL USERS TO RUNTIME;
SAVE MYSQL USERS TO DISK;
`, ";") {
			q = strings.TrimSpace(q)
			if q == "" {
				continue
			}
			_, err = db.Exec(q)
			if err != nil {
				t.Fatalf("Failed to execute %q\n%s", q, err)
			}
		}
		break
	}

	convey.Convey("Metrics descriptions", t, convey.FailureContinues, func() {
		ch := make(chan *prometheus.Desc)
		go func() {
			exporter.Describe(ch)
			close(ch)
		}()

		descs := make(map[string]struct{})
		for d := range ch {
			descs[d.String()] = struct{}{}
		}

		convey.So(descs, convey.ShouldContainKey,
			`Desc{fqName: "proxysql_connection_pool_latency_us", help: "The currently ping time in microseconds, as reported from Monitor.", constLabels: {}, variableLabels: [hostgroup endpoint]}`)
	})

	convey.Convey("Metrics data", t, convey.FailureContinues, func() {
		ch := make(chan prometheus.Metric)
		go func() {
			exporter.Collect(ch)
			close(ch)
		}()

		var metrics []helpers.Metric
		for m := range ch {
			got := *helpers.ReadMetric(m)
			got.Value = 0 // ignore actual values in comparison for now
			metrics = append(metrics, got)
		}

		for _, m := range metrics {
			convey.So(m.Name, convey.ShouldEqual, strings.ToLower(m.Name))
			for k := range m.Labels {
				convey.So(k, convey.ShouldEqual, strings.ToLower(k))
			}
		}

		convey.So(helpers.Metric{"proxysql_connection_pool_latency_us", prometheus.Labels{"hostgroup": "1", "endpoint": "mysql:3306"}, dto.MetricType_GAUGE, 0},
			convey.ShouldBeIn, metrics)
		convey.So(helpers.Metric{"proxysql_connection_pool_latency_us", prometheus.Labels{"hostgroup": "1", "endpoint": "percona-server:3306"}, dto.MetricType_GAUGE, 0},
			convey.ShouldBeIn, metrics)
	})
}
