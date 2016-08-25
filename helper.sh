for metric in ConnUsed ConnFree ConnOK ConnERR Queries Bytes_data_sent Bytes_data_recv Latency_ms; do

cat <<EOF
	Stats_MySQL_Connection_Pool_$metric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "Stats_MySQL_Connection_Pool_$metric",
			Subsystem: "",
			Help: "stats.stats_mysql_connection_pool.$metric"
			Namespace: namespace,
		},[]string{"hostgroup","srv_host","srv_port"}
	)
EOF

echo "prometheus.MustRegister(Stats_MySQL_Connection_Pool_$metric)"

done

for metric in ConnUsed ConnFree ConnOK ConnERR Queries Bytes_data_sent Bytes_data_recv Latency_ms; do

cat <<EOF
			Stats_MySQL_Connection_Pool_$metric.With(prometheus.Labels{
				"hostgroup":hostgroup,
				"srv_host":srv_host,
				"srv_port":srv_port
			}).Set($metric)
EOF

done
