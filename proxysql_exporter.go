package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
)

var (
	version = "1.0.0"

	showVersion = flag.Bool(
		"version", false,
		"Print version information.",
	)
	listenAddress = flag.String(
		"web.listen-address", ":42004",
		"Address to listen on for web interface and telemetry.",
	)
	metricPath = flag.String(
		"web.telemetry-path", "/metrics",
		"Path under which to expose metrics.",
	)

	collectMySQLStatus = flag.Bool(
		"collect.mysql_status", true,
		"Collect from SHOW MYSQL STATUS",
	)
	collectMySQLConnectionPool = flag.Bool(
		"collect.mysql_connection_pool", true,
		"Collect from stats_mysql_connection_pool",
	)
)

const (
	namespace                = "proxysql"
	exporter                 = "exporter"
	mysqlStatusQuery         = "SHOW MYSQL STATUS"
	mysqlConnectionPoolQuery = `
		SELECT
			hostgroup, srv_host, srv_port, status,
			ConnUsed, ConnFree, ConnOK, ConnERR, Queries,
			Bytes_data_sent, Bytes_data_recv, Latency_ms
		FROM stats_mysql_connection_pool
	`
)

var landingPage = []byte(`<html>
<head><title>ProxySQL exporter</title></head>
<body>
<h1>ProxySQL exporter</h1>
<p><a href='` + *metricPath + `'>Metrics</a></p>
</body>
</html>
`)

type basicAuthHandler struct {
	handler  http.HandlerFunc
	user     string
	password string
}

func (h *basicAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user, password, ok := r.BasicAuth()
	if !ok || password != h.password || user != h.user {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"metrics\"")
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}
	h.handler(w, r)
}

type Exporter struct {
	dsn             string
	duration        prometheus.Gauge
	error           prometheus.Gauge
	totalScrapes    prometheus.Counter
	scrapeErrors    *prometheus.CounterVec
	proxysqlUP      prometheus.Gauge
}

func NewExporter(dsn string) *Exporter {
	return &Exporter{
		dsn: dsn,
		duration: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: exporter,
			Name:      "last_scrape_duration_seconds",
			Help:      "Duration of the last scrape of metrics from ProxySQL.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: exporter,
			Name:      "scrapes_total",
			Help:      "Total number of times ProxySQL was scraped for metrics.",
		}),
		scrapeErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: exporter,
			Name:      "scrape_errors_total",
			Help:      "Total number of times an error occured scraping a ProxySQL.",
		}, []string{"collector"}),
		error: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: exporter,
			Name:      "last_scrape_error",
			Help:      "Whether the last scrape of metrics from ProxySQL resulted in an error (1 for error, 0 for success).",
		}),
		proxysqlUP: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Whether ProxySQL is up.",
		}),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	metricCh := make(chan prometheus.Metric)
	doneCh := make(chan struct{})

	go func() {
		for m := range metricCh {
			ch <- m.Desc()
		}
		close(doneCh)
	}()

	e.Collect(metricCh)
	close(metricCh)
	<-doneCh
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.scrape(ch)

	ch <- e.duration
	ch <- e.totalScrapes
	ch <- e.error
	e.scrapeErrors.Collect(ch)
	ch <- e.proxysqlUP
}

func (e *Exporter) scrape(ch chan<- prometheus.Metric) {
	e.totalScrapes.Inc()
	var err error
	defer func(begun time.Time) {
		e.duration.Set(time.Since(begun).Seconds())
		if err == nil {
			e.error.Set(0)
		} else {
			e.error.Set(1)
		}
	}(time.Now())

	db, err := sql.Open("mysql", e.dsn)
	if err != nil {
		log.Errorln("Error opening connection to database:", err)
		return
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Errorln("Error pinging ProxySQL:", err)
		e.proxysqlUP.Set(0)
		return
	}
	e.proxysqlUP.Set(1)

	if *collectMySQLStatus {
		if err = scrapeMySQLStatus(db, ch); err != nil {
			log.Errorln("Error scraping for collect.mysql_status:", err)
			e.scrapeErrors.WithLabelValues("collect.mysql_status").Inc()
		}
	}
	if *collectMySQLConnectionPool {
		if err = scrapeMySQLConnectionPool(db, ch); err != nil {
			log.Errorln("Error scraping for collect.mysql_connection_pool:", err)
			e.scrapeErrors.WithLabelValues("collect.mysql_connection_pool").Inc()
		}
	}
}

// scrapeMySQLStatus collects `SHOW MYSQL STATUS`.
func scrapeMySQLStatus(db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.Query(mysqlStatusQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	var (
		key string
		val float64
	)
	for rows.Next() {
		if err := rows.Scan(&key, &val); err != nil {
			return err
		}
		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "mysql_status", strings.ToLower(key)),
				"Global status metric.",
				nil, nil,
			),
			prometheus.UntypedValue, val,
		)
	}

	return nil
}

func newConnPoolMetric(name, hostgroup, endpoint string, value float64, valueType prometheus.ValueType, ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "connection_pool", name),
			"Connection pool usage statistic.",
			[]string{"hostgroup", "endpoint"}, nil,
		),
		valueType, value,
		hostgroup, endpoint,
	)
}

// scrapeMySQLConnectionPool collects from `stats_mysql_connection_pool`.
func scrapeMySQLConnectionPool(db *sql.DB, ch chan<- prometheus.Metric) error {
	rows, err := db.Query(mysqlConnectionPoolQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	var (
		hostgroup       string
		host            string
		port            string
		status          string
		statusNum       float64
		ConnUsed        float64
		ConnFree        float64
		ConnOK          float64
		ConnERR         float64
		Queries         float64
		Bytes_data_sent float64
		Bytes_data_recv float64
		Latency_ms      float64
	)
	for rows.Next() {
		if err := rows.Scan(&hostgroup, &host, &port, &status, &ConnUsed, &ConnFree, &ConnOK, &ConnERR, &Queries, &Bytes_data_sent, &Bytes_data_recv, &Latency_ms); err != nil {
			return err
		}
		// Map status to ids.
		switch status {
		case "ONLINE":
			statusNum = 1
		case "SHUNNED":
			statusNum = 2
		case "OFFLINE_SOFT":
			statusNum = 3
		case "OFFLINE_HARD":
			statusNum = 4
		}

		endpoint := host + ":" + port
		newConnPoolMetric("status", hostgroup, endpoint, statusNum, prometheus.GaugeValue, ch)
		newConnPoolMetric("conn_used", hostgroup, endpoint, ConnUsed, prometheus.GaugeValue, ch)
		newConnPoolMetric("conn_free", hostgroup, endpoint, ConnFree, prometheus.GaugeValue, ch)
		newConnPoolMetric("conn_ok", hostgroup, endpoint, ConnOK, prometheus.CounterValue, ch)
		newConnPoolMetric("conn_err", hostgroup, endpoint, ConnERR, prometheus.CounterValue, ch)
		newConnPoolMetric("queries", hostgroup, endpoint, Queries, prometheus.CounterValue, ch)
		newConnPoolMetric("bytes_data_sent", hostgroup, endpoint, Bytes_data_sent, prometheus.CounterValue, ch)
		newConnPoolMetric("bytes_data_recv", hostgroup, endpoint, Bytes_data_recv, prometheus.CounterValue, ch)
		newConnPoolMetric("latency_ms", hostgroup, endpoint, Latency_ms, prometheus.GaugeValue, ch)
	}

	return nil
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version)
		os.Exit(0)
	}

	log.Infoln("Starting proxysql_exporter", version)

	dsn := os.Getenv("DATA_SOURCE_NAME")
	if dsn == "" {
		dsn = "stats:stats@tcp(localhost:6032)/"
	}

	var authUser, authPass string
	httpAuth := os.Getenv("HTTP_AUTH")
	if httpAuth != "" {
		data := strings.SplitN(httpAuth, ":", 2)
		if len(data) != 2 || data[0] == "" || data[1] == "" {
			log.Fatal("HTTP_AUTH should be formatted as user:password")
		}
		authUser = data[0]
		authPass = data[1]
		log.Infoln("HTTP basic authentication is enabled")
	}

	exporter := NewExporter(dsn)
	prometheus.MustRegister(exporter)

	handler := prometheus.Handler()
	if authUser != "" && authPass != "" {
		handler = &basicAuthHandler{handler: handler.ServeHTTP, user: authUser, password: authPass}
	}
	http.Handle(*metricPath, handler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(landingPage)
	})

	log.Infoln("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
