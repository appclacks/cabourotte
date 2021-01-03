package exporter

import (
	"testing"

	"gopkg.in/yaml.v2"

	"cabourotte/healthcheck"
)

func TestUnmarshalConfig(t *testing.T) {
	cases := []struct {
		in   string
		want HTTPConfiguration
	}{
		{
			in: `
host: "127.0.0.1"
port: 2000
protocol: https
name: foo
`,
			want: HTTPConfiguration{
				Name:     "foo",
				Host:     "127.0.0.1",
				Port:     2000,
				Protocol: healthcheck.HTTPS,
			},
		},
		{
			in: `
host: "127.0.0.2"
port: 2003
protocol: http
name: foo
`,
			want: HTTPConfiguration{
				Name:     "foo",
				Host:     "127.0.0.2",
				Port:     2003,
				Protocol: healthcheck.HTTP,
			},
		},
		{
			in: `
host: "127.0.0.2"
port: 2003
protocol: http
name: foo
key: /tmp/key
cert: /tmp/cert
cacert: /tmp/cacert
`,
			want: HTTPConfiguration{
				Name:     "foo",
				Host:     "127.0.0.2",
				Port:     2003,
				Protocol: healthcheck.HTTP,
				Key:      "/tmp/key",
				Cert:     "/tmp/cert",
				Cacert:   "/tmp/cacert",
			},
		},
		{
			in: `
host: "127.0.0.2"
port: 2003
protocol: http
name: foo
cacert: /tmp/cacert
insecure: true
`,
			want: HTTPConfiguration{
				Name:     "foo",
				Host:     "127.0.0.2",
				Port:     2003,
				Protocol: healthcheck.HTTP,
				Cacert:   "/tmp/cacert",
				Insecure: true,
			},
		},
	}
	for _, c := range cases {
		var result HTTPConfiguration
		if err := yaml.Unmarshal([]byte(c.in), &result); err != nil {
			t.Fatalf("Unmarshal yaml error:\n%v", err)
		}
		if result != c.want {
			t.Fatalf("Invalid configuration: \n%s\n%v", c.in, c.want)
		}
	}
}

func TestUnmarshalConfigError(t *testing.T) {
	cases := []string{
		`
host: "127.0.0.1"
port: 2000
protocol: lol
`,

		`
host: "127.0.0.1"
port: 0
protocol: http
`,
		`
host: "127.0.0.1"
port: 0
protocol: tcp
`,

		`
host: ""
port: 2003
protocol: http
`,
		`
host: ""
port: 2003
protocol: http
key: /tmp/key
`,
		`
host: ""
port: 2003
protocol: http
key: /tmp/key
cert: /tmp/cert
`,
	}
	for _, c := range cases {
		var result HTTPConfiguration
		if err := yaml.Unmarshal([]byte(c), &result); err == nil {
			t.Fatalf("Was expecting an error for:\n%s", c)
		}
	}
}
