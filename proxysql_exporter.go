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
	Client_Connections_aborted = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Client_Connections_aborted",
			Subsystem: "",
			Help:      "Client_Connections_aborted from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Client_Connections_connected = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Client_Connections_connected",
			Subsystem: "",
			Help:      "Client_Connections_connected from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Client_Connections_created = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Client_Connections_created",
			Subsystem: "",
			Help:      "Client_Connections_created from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Com_autocommit = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Com_autocommit",
			Subsystem: "",
			Help:      "Com_autocommit from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Com_autocommit_filtered = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Com_autocommit_filtered",
			Subsystem: "",
			Help:      "Com_autocommit_filtered from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Com_commit = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Com_commit",
			Subsystem: "",
			Help:      "Com_commit from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Com_commit_filtered = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Com_commit_filtered",
			Subsystem: "",
			Help:      "Com_commit_filtered from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Com_rollback = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Com_rollback",
			Subsystem: "",
			Help:      "Com_rollback from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Com_rollback_filtered = prometheus.NewGauge(
		prometheus.GaugeOpts{
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
	Queries_backends_bytes_recv = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Queries_backends_bytes_recv",
			Subsystem: "",
			Help:      "Queries_backends_bytes_recv from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Queries_backends_bytes_sent = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Queries_backends_bytes_sent",
			Subsystem: "",
			Help:      "Queries_backends_bytes_sent from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	// Query_Processor_time_nsec not included for the same reason as Backend_query_time_nsec
	Questions = prometheus.NewGauge(
		prometheus.GaugeOpts{
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
	Server_Connections_aborted = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Server_Connections_aborted",
			Subsystem: "",
			Help:      "Server_Connections_aborted from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Server_Connections_connected = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Server_Connections_connected",
			Subsystem: "",
			Help:      "Server_Connections_connected from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	Server_Connections_created = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Server_Connections_created",
			Subsystem: "",
			Help:      "Server_Connections_created from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	// Servers_table_version
	Slow_queries = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "Slow_queries",
			Subsystem: "",
			Help:      "Slow_queries from SHOW MYSQL STATUS",
			Namespace: namespace,
		},
	)
	// mysql_backend|frontend|session bytes seem per-thread
	// next are from stats.stats_mysql_connection_pool
	Stats_MySQL_Connection_Pool_ConnUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "Stats_MySQL_Connection_Pool_ConnUsed",
			Subsystem: "",
			Help:      "stats.stats_mysql_connection_pool.ConnUsed",
			Namespace: namespace,
		}, []string{"hostgroup", "srv_host", "srv_port"},
	)
	Stats_MySQL_Connection_Pool_ConnFree = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "Stats_MySQL_Connection_Pool_ConnFree",
			Subsystem: "",
			Help:      "stats.stats_mysql_connection_pool.ConnFree",
			Namespace: namespace,
		}, []string{"hostgroup", "srv_host", "srv_port"},
	)
	Stats_MySQL_Connection_Pool_ConnOK = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "Stats_MySQL_Connection_Pool_ConnOK",
			Subsystem: "",
			Help:      "stats.stats_mysql_connection_pool.ConnOK",
			Namespace: namespace,
		}, []string{"hostgroup", "srv_host", "srv_port"},
	)
	Stats_MySQL_Connection_Pool_ConnERR = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "Stats_MySQL_Connection_Pool_ConnERR",
			Subsystem: "",
			Help:      "stats.stats_mysql_connection_pool.ConnERR",
			Namespace: namespace,
		}, []string{"hostgroup", "srv_host", "srv_port"},
	)
	Stats_MySQL_Connection_Pool_Queries = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "Stats_MySQL_Connection_Pool_Queries",
			Subsystem: "",
			Help:      "stats.stats_mysql_connection_pool.Queries",
			Namespace: namespace,
		}, []string{"hostgroup", "srv_host", "srv_port"},
	)
	Stats_MySQL_Connection_Pool_Bytes_data_sent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "Stats_MySQL_Connection_Pool_Bytes_data_sent",
			Subsystem: "",
			Help:      "stats.stats_mysql_connection_pool.Bytes_data_sent",
			Namespace: namespace,
		}, []string{"hostgroup", "srv_host", "srv_port"},
	)
	Stats_MySQL_Connection_Pool_Bytes_data_recv = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "Stats_MySQL_Connection_Pool_Bytes_data_recv",
			Subsystem: "",
			Help:      "stats.stats_mysql_connection_pool.Bytes_data_recv",
			Namespace: namespace,
		}, []string{"hostgroup", "srv_host", "srv_port"},
	)
	Stats_MySQL_Connection_Pool_Latency_ms = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:      "Stats_MySQL_Connection_Pool_Latency_ms",
			Subsystem: "",
			Help:      "stats.stats_mysql_connection_pool.Latency_ms",
			Namespace: namespace,
		}, []string{"hostgroup", "srv_host", "srv_port"},
	)
)

func waitBeforeRetry() {
	time.Sleep(time.Duration(retry_millis) * time.Millisecond)
}

// connectToAdmin will attempt to connect to the dsn, retrying indefinitely every retry_millis ms if there is an error
func connectToAdmin() *sql.DB {
	for {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			fmt.Println(err)
			waitBeforeRetry()
		} else {
			return db
		}
	}
}

func scrapeStatsMySQLConnectionPool(db *sql.DB) {
	for {
		rows, err := db.Query("select * from stats.stats_mysql_connection_pool")
		if err != nil {
			fmt.Println(err)
			waitBeforeRetry()
			db.Close()
			db = connectToAdmin()
			continue
		}
		defer rows.Close()
		for rows.Next() {
			var (
				hostgroup       string
				srv_host        string
				srv_port        string
				status          string
				ConnUsed        float64
				ConnFree        float64
				ConnOK          float64
				ConnERR         float64
				Queries         float64
				Bytes_data_sent float64
				Bytes_data_recv float64
				Latency_ms      float64
			)
			err := rows.Scan(&hostgroup, &srv_host, &srv_port, &status, &ConnUsed, &ConnFree, &ConnOK, &ConnERR, &Queries, &Bytes_data_sent, &Bytes_data_recv, &Latency_ms)
			if err != nil {
				fmt.Println(err)
				waitBeforeRetry()
				continue
			}
			Stats_MySQL_Connection_Pool_ConnUsed.With(prometheus.Labels{
				"hostgroup": hostgroup,
				"srv_host":  srv_host,
				"srv_port":  srv_port,
			}).Set(ConnUsed)
			Stats_MySQL_Connection_Pool_ConnFree.With(prometheus.Labels{
				"hostgroup": hostgroup,
				"srv_host":  srv_host,
				"srv_port":  srv_port,
			}).Set(ConnFree)
			Stats_MySQL_Connection_Pool_ConnOK.With(prometheus.Labels{
				"hostgroup": hostgroup,
				"srv_host":  srv_host,
				"srv_port":  srv_port,
			}).Set(ConnOK)
			Stats_MySQL_Connection_Pool_ConnERR.With(prometheus.Labels{
				"hostgroup": hostgroup,
				"srv_host":  srv_host,
				"srv_port":  srv_port,
			}).Set(ConnERR)
			Stats_MySQL_Connection_Pool_Queries.With(prometheus.Labels{
				"hostgroup": hostgroup,
				"srv_host":  srv_host,
				"srv_port":  srv_port,
			}).Set(Queries)
			Stats_MySQL_Connection_Pool_Bytes_data_sent.With(prometheus.Labels{
				"hostgroup": hostgroup,
				"srv_host":  srv_host,
				"srv_port":  srv_port,
			}).Set(Bytes_data_sent)
			Stats_MySQL_Connection_Pool_Bytes_data_recv.With(prometheus.Labels{
				"hostgroup": hostgroup,
				"srv_host":  srv_host,
				"srv_port":  srv_port,
			}).Set(Bytes_data_recv)
			Stats_MySQL_Connection_Pool_Latency_ms.With(prometheus.Labels{
				"hostgroup": hostgroup,
				"srv_host":  srv_host,
				"srv_port":  srv_port,
			}).Set(Latency_ms)
		}
	}
}

func scrapeShowMySQLStatus(db *sql.DB) {
	for {
		rows, err := db.Query("SHOW MYSQL STATUS")
		if err != nil {
			fmt.Println(err)
			waitBeforeRetry()
			db.Close()
			db = connectToAdmin()
			continue
		}
		defer rows.Close()
		for rows.Next() {
			var Variable_name string
			var Value float64
			err := rows.Scan(&Variable_name, &Value)
			if err != nil {
				fmt.Println(err)
				waitBeforeRetry()
				continue
			}
			switch Variable_name {
			case "Active_transactions":
				Active_Transactions.Set(Value)
			case "Client_Connections_aborted":
				Client_Connections_aborted.Set(Value)
			case "Client_Connections_connected":
				Client_Connections_connected.Set(Value)
			case "Client_Connections_created":
				Client_Connections_created.Set(Value)
			case "Com_autocommit":
				Com_autocommit.Set(Value)
			case "Com_autocommit_filtered":
				Com_autocommit_filtered.Set(Value)
			case "Com_commit":
				Com_commit.Set(Value)
			case "Com_commit_filtered":
				Com_commit_filtered.Set(Value)
			case "Com_rollback":
				Com_rollback.Set(Value)
			case "Com_rollback_filtered":
				Com_rollback_filtered.Set(Value)
			case "ConnPool_memory_bytes":
				ConPool_memory_bytes.Set(Value)
			case "MySQL_Monitor_Workers":
				MySQL_Monitor_Workers.Set(Value)
			case "MySQL_Thread_Workers":
				MySQL_Thread_Workers.Set(Value)
			case "Queries_backends_bytes_recv":
				Queries_backends_bytes_recv.Set(Value)
			case "Queries_backends_bytes_sent":
				Queries_backends_bytes_sent.Set(Value)
			case "Questions":
				Questions.Set(Value)
			case "SQLite3_memory_bytes":
				SQLite3_memory_bytes.Set(Value)
			case "Server_Connections_aborted":
				Server_Connections_aborted.Set(Value)
			case "Server_Connections_connected":
				Server_Connections_connected.Set(Value)
			case "Server_Connections_created":
				Server_Connections_created.Set(Value)
			case "Slow_queries":
				Slow_queries.Set(Value)

			}
		}
		time.Sleep(time.Duration(scrape_millis) * time.Millisecond)
	}
}

func init() {
	flag.StringVar(&addr, "listen-address", ":2314", "The address to listen on for HTTP requests.")
	flag.StringVar(&user, "user", "admin", "The ProxySQL admin interface username.")
	flag.StringVar(&password, "password", "admin", "The ProxySQL admin interface password.")
	flag.StringVar(&dsn, "dsn", "admin:admin@tcp(localhost:6032)/admin", "The dsn to use to connect to ProxySQL's admin interface.")
	flag.IntVar(&retry_millis, "retry_millis", 1000, "The number of milliseconds to wait before retrying after a database failure.")
	flag.IntVar(&scrape_millis, "scrape_millis", 1000, "The number of milliseconds to wait between scraping runs. ")
	prometheus.MustRegister(Active_Transactions)
	prometheus.MustRegister(Client_Connections_aborted)
	prometheus.MustRegister(Client_Connections_connected)
	prometheus.MustRegister(Client_Connections_created)
	prometheus.MustRegister(Com_autocommit)
	prometheus.MustRegister(Com_autocommit_filtered)
	prometheus.MustRegister(Com_commit)
	prometheus.MustRegister(Com_commit_filtered)
	prometheus.MustRegister(Com_rollback)
	prometheus.MustRegister(Com_rollback_filtered)
	prometheus.MustRegister(ConPool_memory_bytes)
	prometheus.MustRegister(MySQL_Monitor_Workers)
	prometheus.MustRegister(MySQL_Thread_Workers)
	prometheus.MustRegister(Queries_backends_bytes_recv)
	prometheus.MustRegister(Queries_backends_bytes_sent)
	prometheus.MustRegister(Questions)
	prometheus.MustRegister(SQLite3_memory_bytes)
	prometheus.MustRegister(Server_Connections_aborted)
	prometheus.MustRegister(Server_Connections_connected)
	prometheus.MustRegister(Server_Connections_created)
	prometheus.MustRegister(Slow_queries)
	prometheus.MustRegister(Stats_MySQL_Connection_Pool_ConnUsed)
	prometheus.MustRegister(Stats_MySQL_Connection_Pool_ConnFree)
	prometheus.MustRegister(Stats_MySQL_Connection_Pool_ConnOK)
	prometheus.MustRegister(Stats_MySQL_Connection_Pool_ConnERR)
	prometheus.MustRegister(Stats_MySQL_Connection_Pool_Queries)
	prometheus.MustRegister(Stats_MySQL_Connection_Pool_Bytes_data_sent)
	prometheus.MustRegister(Stats_MySQL_Connection_Pool_Bytes_data_recv)
	prometheus.MustRegister(Stats_MySQL_Connection_Pool_Latency_ms)
}

func main() {
	flag.Parse()
	db := connectToAdmin()
	defer db.Close()
	go scrapeShowMySQLStatus(db)
	go scrapeStatsMySQLConnectionPool(db)
	http.Handle("/metrics", prometheus.Handler())
	http.ListenAndServe(addr, nil)
}
