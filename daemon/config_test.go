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
dns_checks:
  - name: foo
    description: bar
    domain: mcorbin.fr
    interval: 10s
  - name: bar
    description: bar
    domain: mcorbin.fr
    expected_ips:
      - 10.0.0.1
      - 10.0.0.2
    interval: 10s
`,
			want: Configuration{
				ResultBuffer: DefaultBufferSize,
				HTTP: http.Configuration{
					Host: "127.0.0.1",
					Port: 2000,
				},
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
dns_checks:
  - name: foo
    description: bar
    domain: mcorbin.fr
    interval: 10s
tcp_checks:
  - name: foo
    description: bar
    target: "127.0.0.1"
    port: 8080
    source_ip: "10.0.0.4"
    interval: 10s
    timeout: 5s
tls_checks:
  - name: tls
    description: bar
    insecure: true
    target: "127.0.0.1"
    port: 8080
    source_ip: "10.0.0.4"
    cert: /tmp/foo.cert
    cacert: /tmp/bar.cacert
    key: /tmp/bar.key
    server_name: mcorbin.fr
    expiration_delay: 24h
    interval: 10s
    timeout: 5s
http_checks:
  - name: foo
    description: bar
    target: "mcorbin.fr"
    port: 443
    body_regexp:
      - "foo*"
    interval: 10s
    timeout: 5s
    path: "/foo"
    protocol: https
    redirect: true
    source_ip: 127.0.0.3
    headers:
      foo: bar
    body: foobar
    valid_status:
      - 200
      - 201
result_buffer: 1000
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
					},
				},
				HTTPChecks: []healthcheck.HTTPHealthcheckConfiguration{
					healthcheck.HTTPHealthcheckConfiguration{
						Name:        "foo",
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
dns_checks:
  - name: foo
    description: bar
    domain: ""
    interval: 10s
`,
		`
http:
  host: "127.0.0.1"
  port: 2000
dns_checks:
  - name: foo
    description: bar
    domain: mcorbin.fr
    interval: 1s
`,
		`
http:
  host: "127.0.0.1"
  port: 2000
tcp_checks:
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
tls_checks:
  - name: foo
    description: bar
    target: 127.0.0.1
    port: 2000
    interval: 10s
    timeout: 5s
    expiration_delay: foo
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
