# Contributing notes

## Local setup

The easiest way to make a local development setup is to use Docker Compose.

```
docker-compose up
make all testall
export DATA_SOURCE_NAME='admin:admin@tcp(127.0.0.1:16032)/'
./proxysql_exporter
```

`testall` make target will run integration tests and also leave ProxySQL inside Docker container in configured state.


## Vendoring

We use [Glide](https://glide.sh) to vendor dependencies. Please use released version of Glide (i.e. do not `go get`
from `master` branch). Also please use `--strip-vendor` flag.
