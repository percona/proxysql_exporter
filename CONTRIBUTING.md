# Contributing notes

## Local setup

The easiest way to make a local development setup is to use Docker Compose.

```
docker-compose up
<wait>
mysql --host=127.0.0.1 --port=16032 --user=admin --password=admin < proxysql.sql
export DATA_SOURCE_NAME='admin:admin@tcp(127.0.0.1:16032)/'
```

## Vendoring

We use [Glide](https://glide.sh) to vendor dependencies. Please use released version of Glide (i.e. do not `go get`
from `master` branch). Also please use `--strip-vendor` flag.
