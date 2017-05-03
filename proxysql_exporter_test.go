package main

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/smartystreets/goconvey/convey"
	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

type labelMap map[string]string

type metricResult struct {
	labels     labelMap
	value      float64
	metricType dto.MetricType
}

func readMetric(m prometheus.Metric) metricResult {
	pb := &dto.Metric{}
	m.Write(pb)
	labels := make(labelMap, len(pb.Label))
	for _, v := range pb.Label {
		labels[v.GetName()] = v.GetValue()
	}
	if pb.Gauge != nil {
		return metricResult{labels: labels, value: pb.GetGauge().GetValue(), metricType: dto.MetricType_GAUGE}
	}
	if pb.Counter != nil {
		return metricResult{labels: labels, value: pb.GetCounter().GetValue(), metricType: dto.MetricType_COUNTER}
	}
	if pb.Untyped != nil {
		return metricResult{labels: labels, value: pb.GetUntyped().GetValue(), metricType: dto.MetricType_UNTYPED}
	}
	panic("Unsupported metric type")
}

func sanitizeQuery(q string) string {
	q = strings.Join(strings.Fields(q), " ")
	q = strings.Replace(q, "(", "\\(", -1)
	q = strings.Replace(q, ")", "\\)", -1)
	q = strings.Replace(q, "*", "\\*", -1)
	return q
}

func TestScrapeMySQLStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error opening a stub database connection: %s", err)
	}
	defer db.Close()

	columns := []string{"Variable_name", "Value"}
	rows := sqlmock.NewRows(columns).
		AddRow("Active_Transactions", "3").
		AddRow("Backend_query_time_nsec", "76355784684851").
		AddRow("Client_Connections_aborted", "0").
		AddRow("Client_Connections_connected", "64").
		AddRow("Client_Connections_created", "1087931").
		AddRow("Servers_table_version", "2019470")
	mock.ExpectQuery(mysqlStatusQuery).WillReturnRows(rows)

	ch := make(chan prometheus.Metric)
	go func() {
		if err = scrapeMySQLStatus(db, ch); err != nil {
			t.Errorf("error calling function on test: %s", err)
		}
		close(ch)
	}()

	counterExpected := []metricResult{
		{labels: labelMap{}, value: 3, metricType: dto.MetricType_UNTYPED},
		{labels: labelMap{}, value: 76355784684851, metricType: dto.MetricType_UNTYPED},
		{labels: labelMap{}, value: 0, metricType: dto.MetricType_UNTYPED},
		{labels: labelMap{}, value: 64, metricType: dto.MetricType_UNTYPED},
		{labels: labelMap{}, value: 1087931, metricType: dto.MetricType_UNTYPED},
		{labels: labelMap{}, value: 2019470, metricType: dto.MetricType_UNTYPED},
	}
	convey.Convey("Metrics comparison", t, func() {
		for _, expect := range counterExpected {
			got := readMetric(<-ch)
			convey.So(got, convey.ShouldResemble, expect)
		}
	})

	// Ensure all SQL queries were executed
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}

func TestScrapeMySQLConnectionPool(t *testing.T) {
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
	mock.ExpectQuery(sanitizeQuery(mysqlConnectionPoolQuery)).WillReturnRows(rows)

	ch := make(chan prometheus.Metric)
	go func() {
		if err = scrapeMySQLConnectionPool(db, ch); err != nil {
			t.Errorf("error calling function on test: %s", err)
		}
		close(ch)
	}()

	counterExpected := []metricResult{
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, value: 1, metricType: dto.MetricType_GAUGE},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, value: 0, metricType: dto.MetricType_GAUGE},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, value: 45, metricType: dto.MetricType_GAUGE},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, value: 1895677, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, value: 46, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, value: 197941647, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, value: 10984550806, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, value: 321063484988, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.80:3306"}, value: 163, metricType: dto.MetricType_GAUGE},

		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, value: 2, metricType: dto.MetricType_GAUGE},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, value: 0, metricType: dto.MetricType_GAUGE},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, value: 97, metricType: dto.MetricType_GAUGE},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, value: 39859, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, value: 0, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, value: 386686994, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, value: 21643682247, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, value: 641406745151, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "0", "endpoint": "10.91.142.82:3306"}, value: 255, metricType: dto.MetricType_GAUGE},

		{labels: labelMap{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, value: 3, metricType: dto.MetricType_GAUGE},
		{labels: labelMap{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, value: 0, metricType: dto.MetricType_GAUGE},
		{labels: labelMap{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, value: 18, metricType: dto.MetricType_GAUGE},
		{labels: labelMap{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, value: 31471, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, value: 6391, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, value: 255993467, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, value: 14327840185, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, value: 420795691329, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "1", "endpoint": "10.91.142.88:3306"}, value: 283, metricType: dto.MetricType_GAUGE},

		{labels: labelMap{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, value: 4, metricType: dto.MetricType_GAUGE},
		{labels: labelMap{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, value: 0, metricType: dto.MetricType_GAUGE},
		{labels: labelMap{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, value: 18, metricType: dto.MetricType_GAUGE},
		{labels: labelMap{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, value: 31471, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, value: 6391, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, value: 255993467, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, value: 14327840185, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, value: 420795691329, metricType: dto.MetricType_COUNTER},
		{labels: labelMap{"hostgroup": "2", "endpoint": "10.91.142.89:3306"}, value: 283, metricType: dto.MetricType_GAUGE},
	}
	convey.Convey("Metrics comparison", t, func() {
		for _, expect := range counterExpected {
			got := readMetric(<-ch)
			convey.So(got, convey.ShouldResemble, expect)
		}
	})

	// Ensure all SQL queries were executed
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expections: %s", err)
	}
}
