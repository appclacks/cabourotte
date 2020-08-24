FROM golang:1.15-buster as build-env

ADD . /app
WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

# -----------------------------------------------------------------------------

FROM scratch

COPY --from=build-env /app/cabourotte /bin/cabourotte

USER 1664

ENTRYPOINT ["/bin/cabourotte"]
CMD ["daemon", "--config", "/cabourotte.yaml"]
