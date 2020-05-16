package daemon

import (
	"reflect"
	"testing"
	"time"

	"gopkg.in/yaml.v2"

	"cabourotte/healthcheck"
	"cabourotte/http"
)

func TestUnmarshalConfig(t *testing.T) {
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
`,
			want: Configuration{
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
    interval: 10s
    timeout: 5s
http_checks:
  - name: foo
    description: bar
    target: "mcorbin.fr"
    port: 443
    interval: 10s
    timeout: 5s
    path: "/foo"
    protocol: https
    valid_status:
      - 200
      - 201
`,
			want: Configuration{
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
				},
				TCPChecks: []healthcheck.TCPHealthcheckConfiguration{
					healthcheck.TCPHealthcheckConfiguration{
						Name:        "foo",
						Description: "bar",
						Target:      "127.0.0.1",
						Port:        8080,
						Timeout:     healthcheck.Duration(time.Second * 5),
						Interval:    healthcheck.Duration(time.Second * 10),
					},
				},
				HTTPChecks: []healthcheck.HTTPHealthcheckConfiguration{
					healthcheck.HTTPHealthcheckConfiguration{
						Name:        "foo",
						Description: "bar",
						Path:        "/foo",
						Target:      "mcorbin.fr",
						Port:        443,
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
			t.Errorf("Unmarshal yaml error:\n%v", err)
		}
		if !reflect.DeepEqual(result, c.want) {
			t.Errorf("Invalid configuration: \n%s\n%v\n%v", c.in, c.want, result)
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
	}
	for _, c := range cases {
		var result Configuration
		err := yaml.Unmarshal([]byte(c), &result)

		if err == nil {
			t.Errorf("Was expected an error when decoding the configuration: \n%s", c)
		}
	}
}
