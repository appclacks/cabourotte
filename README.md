# Cabourotte

This daemon can be configured to periodically perform various kind of healthchecks and report failures.

The healthchecks can be defined in a configuration file, or dynamically managed through an API.

It's also possible to execute `one-off` healthchecks through the API. This type of healthcheck is only executed once, and the HTTP response will contain the healthcheck result.

Healthchecks results can also be exported to other systems by configuring exporters. The latest result for each healthcheck is also stored in memory for 2 minutes, and can be retrieved through the API.

## Quickstart

Configure the HTTP server and your healthchecks using the configuration file:

```yaml
---
http:
  host: "127.0.0.1"
  port: 9013
dns_checks:
  - name: "mcorbin-dns-check"
    description: "dns healthcheck example"
    domain: "mcorbin.fr"
    interval: 5s
http_checks:
  - name: "mcorbin-http-check"
    description: "http healthcheck example"
    valid_status:
      - 200
      - 201
    target: "mcorbin.fr"
    port: 443
    protocol: "https"
    path: "/"
    timeout: 5s
    interval: 10s
tcp_checks:
  - name: "mcorbin-tcp-check"
    description: "tcp healthcheck example"
    target: "mcorbin.fr"
    port: 443
    timeout: 2s
    interval: 10s
exporters:
  http:
    - host: "127.0.0.1"
      port: 9595
      path: "/"
      protocol: "http"
```

Starts the daemon:

```shell
./cabourotte daemon --config ~/cabourotte.yml
```

You should now see your healthchecks being executed:

```json
{"level":"info","ts":1590768585.321339,"caller":"exporter/root.go:51","msg":"Healthcheck successful","name":"mcorbin-dns-check","date":"2020-05-29 18:09:45.321302373 +0200 CEST m=+10.312898322"}
{"level":"info","ts":1590768585.3507266,"caller":"exporter/root.go:51","msg":"Healthcheck successful","name":"mcorbin-tcp-check","date":"2020-05-29 18:09:45.350692848 +0200 CEST m=+10.342288814"}
{"level":"info","ts":1590768585.5289285,"caller":"exporter/root.go:51","msg":"Healthcheck successful","name":"mcorbin-http-check","date":"2020-05-29 18:09:45.528896364 +0200 CEST m=+10.520492364"}
```

## HTTP Server configuration

```yaml
# The HTTP server host
host: "127.0.0.1"
# The HTTP server port
port: 9013
# A cacert for mTLS (optional)
cacert: "/tmp/cacert.pem"
# A cert for mTLS (optional)
cert: "/tmp/cert.pem"
# A key for mTLS (optional)
key: "/tmp/foo.key"
```

## Healthchecks types:

Cabourotte supports multiple healthchecks types. The healthchecks names should be unique. When a new healthcheck is added with a name of an existing healthcheck, the old one will be replaced.

### HTTP

```yaml
 The healthcheck name
name: "mcorbin-http-check"
# The healthcheck description
description: "http healthcheck example"
# The list of HTTP status codes to consider the healthcheck successful
valid_status:
  - 200
  - 201
# The healthcheck target. It can be an IP (v4 or v6) or a domain
target: "mcorbin.fr"
# The target port
port: 443
# The protocol (http or https)
protocol: "https"
# The HTTP path of the healthcheck
path: "/"
# The healthcheck timeout
timeout: 5s
# The healthcheck interval
interval: 10s
# A cacert for mTLS (optional)
cacert: "/tmp/cacert.pem"
# A cert for mTLS (optional)
cert: "/tmp/cert.pem"
# A key for mTLS (optional)
key: "/tmp/foo.key"
```

### TCP

```yaml
 The healthcheck name
name: "mcorbin-http-check"
# The healthcheck description
description: "http healthcheck example"
# The healthcheck target. It can be an IP (v4 or v6) or a domain
target: "mcorbin.fr"
# The target port
port: 443
# The healthcheck timeout
timeout: 5s
# The healthcheck interval
interval: 10s
```

### DNS

```yaml
# The healthcheck name
name: "mcorbin-http-check"
# The healthcheck description
description: "http healthcheck example"
# The healthcheck domain
domain: "mcorbin.fr"
# The healthcheck interval
interval: 10s
```

## Exporters

By default, all healthchecks results are logged.

Cabourotte can also export healthchecks results to other systems using exporters, which can be added into the configuration file.

### HTTP

The HTTP exporter will send healthchecks results to an HTTP server as json

```yaml
# The exporter name
name: http-exporter
# The exporter endpoint
host: "127.0.0.1"
# The exporter port
port: 9000
# The exporter path
path: "/"
# The exporter protocol
protocol: "https"
```

The HTTP endpoint will receive payloads containing the healthchecks results, for example:

```json
[
  {
    "name": "mcorbin-http-check",
    "success": true,
    "timestamp": "2020-05-29T18:45:50.724076768+02:00",
    "message": "success"
  }
]
```

Exporters names should be unique.

## API

### Get healthchecks

Return all healthchecks currently running:

```
curl 127.0.0.1:9013/healthcheck

[{"name":"mcorbin-dns-check","description":"dns healthcheck example","domain":"mcorbin.fr","interval":"5s","one-off":false}]
```

### Add an healthcheck

Dynamically add an healthcheck. A POST request to `/healthcheck/dns`, `/healthcheck/http` or `/healthcheck/tcp` should be executed. The request payload should contain a valid JSON healthcheck definition.

The `one-off` parameter can be set to `true` if you want a healthcheck which will be only executed once.

```
curl -H "Content-Type: application/json" 127.0.0.1:9013/healthcheck/dns -d '{"name":"mcorbin-dns-check","description":"dns healthcheck example","domain":"mcorbin.fr","interval":"5s","one-off":true}'

{"message":"One-off healthcheck mcorbin-dns-check successfully executed"}

```

### Get healthchecks results

You can retrieve the current healthchecks results saved into the memory store by sending requests to the `/result` endpoint:

```
curl 127.0.0.1:9013/healthcheck

[{"name":"mcorbin-dns-check","success":true,"timestamp":"2020-05-30T18:51:01.472044448+02:00","message":"success"},{"name":"mcorbin-tcp-check","success":true,"timestamp":"2020-05-30T18:51:01.502173876+02:00","message":"success"}]
```

You can also retrieve the result for a specific healthcheck:

```
curl 127.0.0.1:9013/result/mcorbin-dns-check

{"name":"mcorbin-dns-check","success":true,"timestamp":"2020-05-30T18:52:31.472050312+02:00","message":"success"}
```

## Hot reload

The daemon supports hot reloading of its configuration file. When the daemon is reloaded:

- Dynamically added healthchecks will not be removed
- Healthchecks which were added, modified, or removed from the configuration file will be updated as expected.
- If the configuration of the HTTP server changed, it will be restarted with the new configuration.
- All exporters will be stopped and started again (I will work on only stop exporters which were modified or removed later).
