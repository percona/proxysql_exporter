// Copyright 2016-2017 Percona LLC.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/subtle"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/yaml.v2"
)

const (
	program           = "proxysql_exporter"
	defaultDataSource = "stats:stats@tcp(localhost:6032)/"
)

var (
	versionF       = flag.Bool("version", false, "Print version information and exit.")
	listenAddressF = flag.String("web.listen-address", ":42004", "Address to listen on for web interface and telemetry.")
	telemetryPathF = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	authFileF      = flag.String("web.auth-file", "", "Path to YAML file with server_user, server_password options for http basic auth (overrides HTTP_AUTH env var).")
	sslCertFileF   = flag.String("web.ssl-cert-file", "", "Path to SSL certificate file.")
	sslKeyFileF    = flag.String("web.ssl-key-file", "", "Path to SSL key file.")

	mysqlStatusF         = flag.Bool("collect.mysql_status", true, "Collect from stats_mysql_global (SHOW MYSQL STATUS).")
	mysqlConnectionPoolF = flag.Bool("collect.mysql_connection_pool", true, "Collect from stats_mysql_connection_pool.")
)

var landingPage = []byte(`<html>
<head><title>ProxySQL exporter</title></head>
<body>
<h1>ProxySQL exporter</h1>
<p><a href='` + *telemetryPathF + `'>Metrics</a></p>
</body>
</html>
`)

type webAuth struct {
	Username string `yaml:"server_user,omitempty"`
	Password string `yaml:"server_password,omitempty"`
}

type basicAuthHandler struct {
	webAuth
	handler http.HandlerFunc
}

func (h *basicAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	username, password, _ := r.BasicAuth()
	usernameOk := subtle.ConstantTimeCompare([]byte(h.Username), []byte(username)) == 1
	passwordOk := subtle.ConstantTimeCompare([]byte(h.Password), []byte(password)) == 1
	if !usernameOk || !passwordOk {
		w.Header().Set("WWW-Authenticate", `Basic realm="metrics"`)
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}
	h.handler(w, r)
}

// logger adapts log.Logger interface to promhttp.Logger interface.
// See https://github.com/prometheus/common/issues/86.
type logger struct {
	log.Logger
}

func (l logger) Println(v ...interface{}) {
	l.Errorln(v...)
}

// check interfaces
var (
	_ log.Logger      = logger{}
	_ promhttp.Logger = logger{}
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s %s exports various ProxySQL metrics in Prometheus format.\n", os.Args[0], version.Version)
		fmt.Fprintf(os.Stderr, "It uses DATA_SOURCE_NAME environment variable with following format: https://github.com/go-sql-driver/mysql#dsn-data-source-name\n")
		fmt.Fprintf(os.Stderr, "Default value is %q.\n\n", defaultDataSource)
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *versionF {
		fmt.Println(version.Print(program))
		os.Exit(0)
	}

	dsn := os.Getenv("DATA_SOURCE_NAME")
	if dsn == "" {
		dsn = defaultDataSource
	}

	log.Infof("Starting %s %s for %s", program, version.Version, dsn)

	cfg := &webAuth{}
	httpAuth := os.Getenv("HTTP_AUTH")
	switch {
	case *authFileF != "":
		bytes, err := ioutil.ReadFile(*authFileF)
		if err != nil {
			log.Fatal("Cannot read auth file: ", err)
		}
		if err := yaml.Unmarshal(bytes, cfg); err != nil {
			log.Fatal("Cannot parse auth file: ", err)
		}
	case httpAuth != "":
		data := strings.SplitN(httpAuth, ":", 2)
		if len(data) != 2 || data[0] == "" || data[1] == "" {
			log.Fatal("HTTP_AUTH should be formatted as user:password")
		}
		cfg.Username = data[0]
		cfg.Password = data[1]
	}

	exporter := NewExporter(dsn, *mysqlStatusF, *mysqlConnectionPoolF)
	prometheus.MustRegister(exporter)

	handler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
		ErrorLog:      logger{log.Base()},
		ErrorHandling: promhttp.HTTPErrorOnError,
	})
	if cfg.Username != "" && cfg.Password != "" {
		handler = &basicAuthHandler{webAuth: *cfg, handler: handler.ServeHTTP}
		log.Infoln("HTTP basic authentication is enabled")
	}

	if (*sslCertFileF == "") != (*sslKeyFileF == "") {
		log.Fatal("One of the flags -web.ssl-cert-file or -web.ssl-key-file is missing to enable HTTPS/TLS")
	}
	ssl := false
	if *sslCertFileF != "" && *sslKeyFileF != "" {
		if _, err := os.Stat(*sslCertFileF); os.IsNotExist(err) {
			log.Fatal("SSL certificate file does not exist: ", *sslCertFileF)
		}
		if _, err := os.Stat(*sslKeyFileF); os.IsNotExist(err) {
			log.Fatal("SSL key file does not exist: ", *sslKeyFileF)
		}
		ssl = true
		log.Infoln("HTTPS/TLS is enabled")
	}

	log.Infoln("Listening on", *listenAddressF)
	if ssl {
		// https
		mux := http.NewServeMux()
		mux.Handle(*telemetryPathF, handler)
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
			Addr:         *listenAddressF,
			Handler:      mux,
			TLSConfig:    tlsCfg,
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		}
		log.Fatal(srv.ListenAndServeTLS(*sslCertFileF, *sslKeyFileF))
	} else {
		// http
		http.Handle(*telemetryPathF, handler)
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write(landingPage)
		})
		log.Fatal(http.ListenAndServe(*listenAddressF, nil))
	}
}
