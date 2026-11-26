FROM alpine AS builder
RUN apk add --no-cache ca-certificates

FROM scratch AS final
ARG TARGETPLATFORM
# GoReleaser stages the binaries under $TARGETPLATFORM in its build context.
ARG BIN_DIR=$TARGETPLATFORM
USER 65535:65535
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY $BIN_DIR/proxysql_exporter /
EXPOSE 42004
ENTRYPOINT ["/proxysql_exporter"]
