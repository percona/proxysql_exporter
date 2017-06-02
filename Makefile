all: test

build:
	go build -v -race
	go vet

test: build
	go test -v

run: build
	./proxysql_exporter
