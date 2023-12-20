FROM golang:1.21.5 as build-env

ADD . /app
WORKDIR /app

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

# -----------------------------------------------------------------------------

FROM alpine:3.19.0

RUN addgroup -S -g 10000 api \
 && adduser -S -D -u 10000 -s /sbin/nologin -G api api

RUN mkdir /app
RUN chown -R 10000:10000 /app

USER 10000

COPY --from=build-env /app/cabourotte /app/cabourotte

ENTRYPOINT ["/app/cabourotte"]
CMD ["daemon", "--config", "/cabourotte.yaml"]
