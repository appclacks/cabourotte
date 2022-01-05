FROM golang:1.17-buster as build-env

ADD . /app
WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

# -----------------------------------------------------------------------------

FROM scratch

COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /app/cabourotte /bin/cabourotte

USER 1664

ENTRYPOINT ["/bin/cabourotte"]
CMD ["daemon", "--config", "/cabourotte.yaml"]
