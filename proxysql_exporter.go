package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"gopkg.in/yaml.v2"
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
	webAuthFile = flag.String(
		"web.auth-file", "",
		"Path to YAML file with server_user, server_password options for http basic auth (overrides HTTP_AUTH env var).",
	)
	sslCertFile = flag.String(
		"web.ssl-cert-file", "",
		"Path to SSL certificate file.",
	)
	sslKeyFile = flag.String(
		"web.ssl-key-file", "",
		"Path to SSL key file.",
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
	mysqlStatusQuery         = "SHOW MYSQL STATUS"
	mysqlConnectionPoolQuery = `
		SELECT
			hostgroup, srv_host, srv_port, status,
			ConnUsed, ConnFree, ConnOK, ConnERR, Queries,
			Bytes_data_sent, Bytes_data_recv, Latency_us
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

type webAuth struct {
	User     string `yaml:"server_user,omitempty"`
	Password string `yaml:"server_password,omitempty"`
}

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

type exporter struct {
	dsn          string
	duration     prometheus.Gauge
	error        prometheus.Gauge
	totalScrapes prometheus.Counter
	scrapeErrors *prometheus.CounterVec
	proxysqlUP   prometheus.Gauge
}

func newExporter(dsn string) *exporter {
	return &exporter{
		dsn: dsn,
		duration: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "last_scrape_duration_seconds",
			Help:      "Duration of the last scrape of metrics from ProxySQL.",
		}),
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "scrapes_total",
			Help:      "Total number of times ProxySQL was scraped for metrics.",
		}),
		scrapeErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "exporter",
			Name:      "scrape_errors_total",
			Help:      "Total number of times an error occurred scraping a ProxySQL.",
		}, []string{"collector"}),
		error: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "exporter",
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

func (e *exporter) Describe(ch chan<- *prometheus.Desc) {
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

func (e *exporter) Collect(ch chan<- prometheus.Metric) {
	e.scrape(ch)

	ch <- e.duration
	ch <- e.totalScrapes
	ch <- e.error
	e.scrapeErrors.Collect(ch)
	ch <- e.proxysqlUP
}

func (e *exporter) scrape(ch chan<- prometheus.Metric) {
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
		hostgroup     string
		host          string
		port          string
		status        string
		statusNum     float64
		ConnUsed      float64
		ConnFree      float64
		ConnOK        float64
		ConnERR       float64
		Queries       float64
		BytesDataSent float64
		BytesDataRecv float64
		LatencyUs     float64
	)
	for rows.Next() {
		if err := rows.Scan(&hostgroup, &host, &port, &status, &ConnUsed, &ConnFree, &ConnOK, &ConnERR, &Queries, &BytesDataSent, &BytesDataRecv, &LatencyUs); err != nil {
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
		newConnPoolMetric("bytes_data_sent", hostgroup, endpoint, BytesDataSent, prometheus.CounterValue, ch)
		newConnPoolMetric("bytes_data_recv", hostgroup, endpoint, BytesDataRecv, prometheus.CounterValue, ch)
		newConnPoolMetric("latency_us", hostgroup, endpoint, LatencyUs, prometheus.GaugeValue, ch)
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

	cfg := &webAuth{}
	httpAuth := os.Getenv("HTTP_AUTH")
	if *webAuthFile != "" {
		bytes, err := ioutil.ReadFile(*webAuthFile)
		if err != nil {
			log.Fatal("Cannot read auth file: ", err)
		}
		if err := yaml.Unmarshal(bytes, cfg); err != nil {
			log.Fatal("Cannot parse auth file: ", err)
		}
	} else if httpAuth != "" {
		data := strings.SplitN(httpAuth, ":", 2)
		if len(data) != 2 || data[0] == "" || data[1] == "" {
			log.Fatal("HTTP_AUTH should be formatted as user:password")
		}
		cfg.User = data[0]
		cfg.Password = data[1]
	}

	exporter := newExporter(dsn)
	prometheus.MustRegister(exporter)

	handler := prometheus.Handler()
	if cfg.User != "" && cfg.Password != "" {
		handler = &basicAuthHandler{handler: handler.ServeHTTP, user: cfg.User, password: cfg.Password}
		log.Infoln("HTTP basic authentication is enabled")
	}

	if *sslCertFile != "" && *sslKeyFile == "" || *sslCertFile == "" && *sslKeyFile != "" {
		log.Fatal("One of the flags -web.ssl-cert or -web.ssl-key is missed to enable HTTPS/TLS")
	}
	ssl := false
	if *sslCertFile != "" && *sslKeyFile != "" {
		if _, err := os.Stat(*sslCertFile); os.IsNotExist(err) {
			log.Fatal("SSL certificate file does not exist: ", *sslCertFile)
		}
		if _, err := os.Stat(*sslKeyFile); os.IsNotExist(err) {
			log.Fatal("SSL key file does not exist: ", *sslKeyFile)
		}
		ssl = true
		log.Infoln("HTTPS/TLS is enabled")
	}

	log.Infoln("Listening on", *listenAddress)
	if ssl {
		// https
		mux := http.NewServeMux()
		mux.Handle(*metricPath, handler)
		mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			w.Write(landingPage)
		})
		tlsCfg := &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}
		srv := &http.Server{
			Addr:         *listenAddress,
			Handler:      mux,
			TLSConfig:    tlsCfg,
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		}
		log.Fatal(srv.ListenAndServeTLS(*sslCertFile, *sslKeyFile))
	} else {
		// http
		http.Handle(*metricPath, handler)
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write(landingPage)
		})
		log.Fatal(http.ListenAndServe(*listenAddress, nil))
	}
}
