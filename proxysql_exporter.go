package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"time"
)

var (
	addr          string
	user          string
	password      string
	dsn           string
	retry_millis  int
	scrape_millis int
)

const (
	namespace = "proxysql"
	exporter  = "exporter"
)

// The exported variables
var (
	Active_Transactions = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Active_Transactions",
			Subsystem: "",
			Help:      "Active_Transactions from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	// Backend_query_time_nsec : This seems per-thread and it does not make sense to export for a monitoring client.
	Client_Connections_aborted = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Client_Connections_aborted",
			Subsystem: "",
			Help:      "Client_Connections_aborted from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Client_Connections_connected = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Client_Connections_connected",
			Subsystem: "",
			Help:      "Client_Connections_connected from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Com_autocommit = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Com_autocommit",
			Subsystem: "",
			Help:      "Com_autocommit from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Com_autocommit_filtered = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Com_autocommit_filtered",
			Subsystem: "",
			Help:      "Com_autocommit_filtered from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Com_commit = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Com_commit",
			Subsystem: "",
			Help:      "Com_commit from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Com_commit_filtered = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Com_commit_filtered",
			Subsystem: "",
			Help:      "Com_commit_filtered from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Com_rollback = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Com_rollback",
			Subsystem: "",
			Help:      "Com_rollback from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Com_rollback_filtered = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Com_rollback_filtered",
			Subsystem: "",
			Help:      "Com_rollback_filtered from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	ConPool_memory_bytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "ConPool_memory_bytes",
			Subsystem: "",
			Help:      "ConPool_memory_bytes from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	MySQL_Monitor_Workers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "MySQL_Monitor_Workers",
			Subsystem: "",
			Help:      "MySQL_Monitor_Workers from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	MySQL_Thread_Workers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "MySQL_Thread_Workers",
			Subsystem: "",
			Help:      "MySQL_Thread_Workers from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Queries_backends_bytes_recv = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Queries_backends_bytes_recv",
			Subsystem: "",
			Help:      "Queries_backends_bytes_recv from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Queries_backends_bytes_sent = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Queries_backends_bytes_sent",
			Subsystem: "",
			Help:      "Queries_backends_bytes_sent from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	// Query_Processor_time_nsec not included for the same reason as Backend_query_time_nsec
	Questions = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Questions",
			Subsystem: "",
			Help:      "Questions from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	SQLite3_memory_bytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "SQLite3_memory_bytes",
			Subsystem: "",
			Help:      "SQLite3_memory_bytes from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Server_Connections_aborted = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Server_Connections_aborted",
			Subsystem: "",
			Help:      "Server_Connections_aborted from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Server_Connections_connected = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Server_Connections_connected",
			Subsystem: "",
			Help:      "Server_Connections_connected from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Server_Connections_created = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Server_Connections_created",
			Subsystem: "",
			Help:      "Server_Connections_created from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	// Servers_table_version
	Slow_queries = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name:      "Slow_queries",
			Subsystem: "",
			Help:      "Slow_queries from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	// mysql_backend|frontend|session bytes seem per-thread

)

// connectToAdmin will attempt to connect to the dsn, retrying indefinitely every retry_millis ms if there is an error
func connectToAdmin() *sql.DB {
	for {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Duration(retry_millis) * time.Millisecond)
		} else {
			return db
		}
	}
}

func scrapeShowMySQLStatus(db *sql.DB) {
	for {
		rows, err := db.Query("SHOW MYSQL STATUS")
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Duration(retry_millis) * time.Millisecond)
			db.Close()
			db = connectToAdmin()
			continue
		}
		defer rows.Close()
		for rows.Next() {
			var Variable_name string
			var Value interface{}
			err := rows.Scan(&Variable_name, &Value)
			if err != nil {
				fmt.Println(err)
				time.Sleep(time.Duration(retry_millis) * time.Millisecond)
				continue
			}
			switch Variable_name {
			case "Active_transactions":
				Active_Transactions.Set(Value.(float64))
			}
		}
		time.Sleep(time.Duration(scrape_millis) * time.Millisecond)
	}
}

func init() {
	flag.StringVar(&addr, "listen-address", ":2314", "The address to listen on for HTTP requests.")
	flag.StringVar(&user, "user", "admin", "The ProxySQL admin interface username.")
	flag.StringVar(&password, "password", "admin", "The ProxySQL admin interface password.")
	flag.StringVar(&dsn, "dsn", "localhost:6032/admin", "The dsn to use to connect to ProxySQL's admin interface.")
	flag.IntVar(&retry_millis, "retry_millis", 1000, "The number of milliseconds to wait before retrying after a database failure.")
	flag.IntVar(&scrape_millis, "scrape_millis", 1000, "The number of milliseconds to wait between scraping runs. ")
	prometheus.MustRegister(Active_Transactions)
}

func main() {
	flag.Parse()
	db := connectToAdmin()
	defer db.Close()
	go scrapeShowMySQLStatus(db)
	http.Handle("/metrics", prometheus.Handler())
	http.ListenAndServe(addr, nil)
}
