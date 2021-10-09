package http

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestUnmarshalConfig(t *testing.T) {
	cases := []struct {
		in   string
		want Configuration
	}{
		{
			in: `
host: "127.0.0.1"
port: 2000
`,
			want: Configuration{
				Host: "127.0.0.1",
				Port: 2000,
			},
		},
		{
			in: `
host: "127.0.0.1"
port: 2000
key: /tmp/foo
cert: /tmp/bar
cacert: /tmp/baz
disable-healthcheck-api: true
disable-result-api: true
basic-auth:
  username: "foo"
  password: "bar"
allowed-cn:
  - "mcorbin"
  - "aaa"
`,
			want: Configuration{
				Host:                  "127.0.0.1",
				Port:                  2000,
				Key:                   "/tmp/foo",
				Cert:                  "/tmp/bar",
				Cacert:                "/tmp/baz",
				DisableResultAPI:      true,
				DisableHealthcheckAPI: true,
				AllowedCN:             []string{"mcorbin", "aaa"},
				BasicAuth: BasicAuth{
					Username: "foo",
					Password: "bar",
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
			t.Fatalf("Invalid configuration: \n%s\n%v", c.in, c.want)
		}
	}
}

func TestUnmarshalConfigFail(t *testing.T) {
	cases := []struct {
		in string
	}{
		{
			in: `
{}
`,
		},
		{
			in: `
host: "127.0.0.1"
`,
		},
		{
			in: `
port: 2000
`,
		},
		{
			in: `
host: ""
`,
		},
		{
			in: `
host: "127.0.0.1"
port: 2000
key: "/tmp/foo"
`,
		},
		{
			in: `
host: "127.0.0.1"
port: 2000
cert: "/tmp/foo"
`,
		},
		{
			in: `

host: "127.0.0.1"
port: 2000
basic-auth:
  password: "foo"
`},

		{
			in: `

host: "127.0.0.1"
port: 2000
basic-auth:
  username: "foo"
`},
	}
	for _, c := range cases {
		var result Configuration
		if err := yaml.Unmarshal([]byte(c.in), &result); err == nil {
			t.Fatalf("Was expecting an error for:\n%s", c.in)
		}
	}
}
