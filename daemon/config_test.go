package daemon

import (
	"net"
	"reflect"
	"regexp"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"cabourotte/exporter"
	"cabourotte/healthcheck"
	"cabourotte/http"
)

func TestUnmarshalConfig(t *testing.T) {
	r := regexp.MustCompile("foo*")
	regexp := healthcheck.Regexp(*r)
	cases := []struct {
		in   string
		want Configuration
	}{
		{
			in: `
http:
  host: "127.0.0.1"
  port: 2000
`,
			want: Configuration{
				ResultBuffer: DefaultBufferSize,
				HTTP: http.Configuration{
					Host: "127.0.0.1",
					Port: 2000,
				},
			},
		},
		{
			in: `
http:
  host: "127.0.0.1"
  port: 2000
dns-checks:
  - name: foo
    description: bar
    domain: mcorbin.fr
    interval: 10s
  - name: bar
    description: bar
    domain: mcorbin.fr
    expected-ips:
      - 10.0.0.1
      - 10.0.0.2
    interval: 10s
`,
			want: Configuration{
				ResultBuffer: DefaultBufferSize,
				HTTP: http.Configuration{
					Host: "127.0.0.1",
					Port: 2000},
				DNSChecks: []healthcheck.DNSHealthcheckConfiguration{
					healthcheck.DNSHealthcheckConfiguration{
						Name:        "foo",
						Description: "bar",
						Domain:      "mcorbin.fr",
						Interval:    healthcheck.Duration(time.Second * 10),
					},
					healthcheck.DNSHealthcheckConfiguration{
						Name:        "bar",
						Description: "bar",
						Domain:      "mcorbin.fr",
						Interval:    healthcheck.Duration(time.Second * 10),
						ExpectedIPs: []healthcheck.IP{
							healthcheck.IP(net.ParseIP("10.0.0.1")),
							healthcheck.IP(net.ParseIP("10.0.0.2")),
						},
					},
				},
			},
		},
		{
			in: `
http:
  host: "127.0.0.1"
  port: 2000
dns-checks:
  - name: foo
    description: bar
    domain: mcorbin.fr
    interval: 10s
    labels:
      environment: prod
tcp-checks:
  - name: foo
    description: bar
    target: "127.0.0.1"
    port: 8080
    source-ip: "10.0.0.4"
    interval: 10s
    timeout: 5s
    labels:
      environment: prod
tls-checks:
  - name: tls
    description: bar
    insecure: true
    target: "127.0.0.1"
    port: 8080
    source-ip: "10.0.0.4"
    cert: /tmp/foo.cert
    cacert: /tmp/bar.cacert
    key: /tmp/bar.key
    server-name: mcorbin.fr
    expiration-delay: 24h
    interval: 10s
    timeout: 5s
    labels:
      environment: prod
  - name: tls2
    description: bar
    insecure: true
    target: "127.0.0.1"
    port: 8080
    source-ip: "10.0.0.4"
    cacert: /tmp/bar.cacert
    server-name: mcorbin.fr
    expiration-delay: 24h
    interval: 10s
    timeout: 5s
    labels:
      environment: prod
http-checks:
  - name: foo
    description: bar
    insecure: true
    target: "mcorbin.fr"
    port: 443
    body-regexp:
      - "foo*"
    interval: 10s
    timeout: 5s
    path: "/foo"
    protocol: https
    redirect: true
    source-ip: 127.0.0.3
    headers:
      foo: bar
    body: foobar
    valid-status:
      - 200
      - 201
    labels:
      environment: prod
  - name: bar
    cacert: /tmp/foo
    description: bar
    insecure: true
    target: "mcorbin.fr"
    port: 443
    body-regexp:
      - "foo*"
    interval: 10s
    timeout: 5s
    path: "/foo"
    protocol: https
    redirect: true
    source-ip: 127.0.0.3
    headers:
      foo: bar
    body: foobar
    valid-status:
      - 200
      - 201
    labels:
      environment: prod
result-buffer: 1000
exporters:
  http:
    - host: "127.0.0.1"
      port: 2000
      name: foo
      protocol: https
`,
			want: Configuration{
				ResultBuffer: 1000,
				HTTP: http.Configuration{
					Host: "127.0.0.1",
					Port: 2000,
				},
				Exporters: exporter.Configuration{
					HTTP: []exporter.HTTPConfiguration{
						exporter.HTTPConfiguration{
							Name:     "foo",
							Host:     "127.0.0.1",
							Port:     2000,
							Protocol: healthcheck.HTTPS,
						},
					},
				},
				DNSChecks: []healthcheck.DNSHealthcheckConfiguration{
					healthcheck.DNSHealthcheckConfiguration{
						Name:        "foo",
						Description: "bar",
						Domain:      "mcorbin.fr",
						Interval:    healthcheck.Duration(time.Second * 10),
						Labels: map[string]string{
							"environment": "prod",
						},
					},
				},
				TCPChecks: []healthcheck.TCPHealthcheckConfiguration{
					healthcheck.TCPHealthcheckConfiguration{
						Name:        "foo",
						Description: "bar",
						Target:      "127.0.0.1",
						Port:        8080,
						SourceIP:    healthcheck.IP(net.ParseIP("10.0.0.4")),
						Timeout:     healthcheck.Duration(time.Second * 5),
						Interval:    healthcheck.Duration(time.Second * 10),
						Labels: map[string]string{
							"environment": "prod",
						},
					},
				},
				TLSChecks: []healthcheck.TLSHealthcheckConfiguration{
					healthcheck.TLSHealthcheckConfiguration{
						Name:            "tls",
						Cert:            "/tmp/foo.cert",
						Cacert:          "/tmp/bar.cacert",
						Key:             "/tmp/bar.key",
						ExpirationDelay: healthcheck.Duration(time.Hour * 24),
						ServerName:      "mcorbin.fr",
						Insecure:        true,
						Description:     "bar",
						Target:          "127.0.0.1",
						Port:            8080,
						SourceIP:        healthcheck.IP(net.ParseIP("10.0.0.4")),
						Timeout:         healthcheck.Duration(time.Second * 5),
						Interval:        healthcheck.Duration(time.Second * 10),
						Labels: map[string]string{
							"environment": "prod",
						},
					},
					healthcheck.TLSHealthcheckConfiguration{
						Name:            "tls2",
						Cacert:          "/tmp/bar.cacert",
						ExpirationDelay: healthcheck.Duration(time.Hour * 24),
						ServerName:      "mcorbin.fr",
						Insecure:        true,
						Description:     "bar",
						Target:          "127.0.0.1",
						Port:            8080,
						SourceIP:        healthcheck.IP(net.ParseIP("10.0.0.4")),
						Timeout:         healthcheck.Duration(time.Second * 5),
						Interval:        healthcheck.Duration(time.Second * 10),
						Labels: map[string]string{
							"environment": "prod",
						},
					},
				},
				HTTPChecks: []healthcheck.HTTPHealthcheckConfiguration{
					healthcheck.HTTPHealthcheckConfiguration{
						Name:        "foo",
						Insecure:    true,
						Description: "bar",
						Body:        "foobar",
						Path:        "/foo",
						BodyRegexp:  []healthcheck.Regexp{regexp},
						SourceIP:    healthcheck.IP(net.ParseIP("127.0.0.3")),
						Target:      "mcorbin.fr",
						Port:        443,
						Redirect:    true,
						Headers: map[string]string{
							"foo": "bar",
						},
						Protocol:    healthcheck.HTTPS,
						Timeout:     healthcheck.Duration(time.Second * 5),
						Interval:    healthcheck.Duration(time.Second * 10),
						ValidStatus: []uint{200, 201},
						Labels: map[string]string{
							"environment": "prod",
						},
					},
					healthcheck.HTTPHealthcheckConfiguration{
						Name:        "bar",
						Cacert:      "/tmp/foo",
						Insecure:    true,
						Description: "bar",
						Body:        "foobar",
						Path:        "/foo",
						BodyRegexp:  []healthcheck.Regexp{regexp},
						SourceIP:    healthcheck.IP(net.ParseIP("127.0.0.3")),
						Target:      "mcorbin.fr",
						Port:        443,
						Redirect:    true,
						Headers: map[string]string{
							"foo": "bar",
						},
						Protocol:    healthcheck.HTTPS,
						Timeout:     healthcheck.Duration(time.Second * 5),
						Interval:    healthcheck.Duration(time.Second * 10),
						ValidStatus: []uint{200, 201},
						Labels: map[string]string{
							"environment": "prod",
						},
					},
				},
			},
		},
	}
	for _, c := range cases {
		var result Configuration
		if err := yaml.Unmarshal([]byte(c.in), &result); err != nil {
			t.Fatalf("Unmarshal yaml error:\n%v", err)
		}
		if !reflect.DeepEqual(result, c.want) {
			t.Fatalf("Invalid configuration: \n%s\n%v\n%v", c.in, c.want, result)
		}
	}
}

func TestInvalidConfig(t *testing.T) {
	cases := []string{
		`
http:
  host: ""
  port: 2000
`,
		`
http:
  port: 2000
`,
		`
http:
  host: 127.0.0.1
`,
		`
http:
  host: 127.0.0.1
  port: 0
`,
		`
http:
  host: 127.0.0.1
  port: 200
dns-checks:
  - name: foo
    description: bar
    domain: ""
    interval: 10s
`,
		`
http:
  host: "127.0.0.1"
  port: 2000
dns-checks:
  - name: foo
    description: bar
    domain: mcorbin.fr
    interval: 1s
`,
		`
http:
  host: "127.0.0.1"
  port: 2000
tcp-checks:
  - name: foo
    description: bar
    target: 127.0.0.1
    port: 2000
    interval: 10s
    timeout: 20s
`,
		`
http:
  host: "127.0.0.1"
  port: 2000
tls-checks:
  - name: foo
    description: bar
    target: 127.0.0.1
    port: 2000
    interval: 10s
    timeout: 5s
    expiration-delay: foo
`,

		`
http:
  host: "127.0.0.1"
  port: 2000
http-checks:
  - name: foo
    description: bar
    target: 127.0.0.1
    port: 2000
    interval: 10s
    timeout: 5s
    expiration-delay: foo
`,
		`
http:
  host: "127.0.0.1"
  port: 2000
tls-checks:
  - name: tls
    description: bar
    insecure: true
    target: "127.0.0.1"
    port: 8080
    source-ip: "10.0.0.4"
    key: /tmp/bar.key
    server-name: mcorbin.fr
    expiration-delay: 24h
    interval: 10s
    timeout: 5s
    labels:
      environment: prod
`,
		`
http:
  host: "127.0.0.1"
  port: 2000
http-checks:
  - name: foo
    description: bar
    insecure: true
    target: "mcorbin.fr"
    port: 443
    body-regexp:
      - "foo*"
    interval: 10s
    timeout: 5s
    path: "/foo"
    protocol: https
    cert: /tmp/foo
    redirect: true
    source-ip: 127.0.0.3
    headers:
      foo: bar
    body: foobar
    valid-status:
      - 200
      - 201
    labels:
      environment: prod
`,
	}
	for _, c := range cases {
		var result Configuration
		err := yaml.Unmarshal([]byte(c), &result)

		if err == nil {
			t.Fatalf("Was expected an error when decoding the configuration: \n%s", c)
		}
	}
}
