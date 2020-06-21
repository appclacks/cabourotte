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
	}
	for _, c := range cases {
		var result HTTPConfiguration
		if err := yaml.Unmarshal([]byte(c.in), &result); err != nil {
			t.Errorf("Unmarshal yaml error:\n%v", err)
		}
		if result != c.want {
			t.Errorf("Invalid configuration: \n%s\n%v", c.in, c.want)
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
	}
	for _, c := range cases {
		var result HTTPConfiguration
		if err := yaml.Unmarshal([]byte(c), &result); err == nil {
			t.Errorf("Was expecting an error for:\n%s", c)
		}
	}
}
