FROM golang:1.11.4-alpine3.8 AS build

RUN apk add git curl

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

WORKDIR /go/src/superbalist/proxysql-exporter

COPY Gopkg.lock ./
COPY Gopkg.toml ./
RUN dep ensure -v -vendor-only

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o ./bin/proxysql_exporter -v ./*.go


FROM alpine:3.8
RUN apk add --no-cache ca-certificates
COPY --from=build /go/src/superbalist/proxysql-exporter/bin/proxysql_exporter /proxysql_exporter
CMD ["./proxysql_exporter"]
