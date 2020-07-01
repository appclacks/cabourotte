package http

import (
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
`,
			want: Configuration{
				Host:   "127.0.0.1",
				Port:   2000,
				Key:    "/tmp/foo",
				Cert:   "/tmp/bar",
				Cacert: "/tmp/baz",
			},
		},
	}
	for _, c := range cases {
		var result Configuration
		if err := yaml.Unmarshal([]byte(c.in), &result); err != nil {
			t.Fatalf("Unmarshal yaml error:\n%v", err)
		}
		if result != c.want {
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
	}
	for _, c := range cases {
		var result Configuration
		if err := yaml.Unmarshal([]byte(c.in), &result); err == nil {
			t.Fatalf("Was expecting an error for:\n%s", c.in)
		}
	}
}
