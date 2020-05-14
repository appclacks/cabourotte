package http

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"go.uber.org/zap"

	"cabourotte/healthcheck"
)

func TestHandlers(t *testing.T) {
	healthcheck, err := healthcheck.New(zap.NewExample(), make(chan *healthcheck.Result, 10))
	if err != nil {
		t.Errorf("Fail to create the healthcheck component\n%v", err)
	}
	component, err := New(zap.NewExample(), &Configuration{Host: "127.0.0.1", Port: 2000}, healthcheck)
	if err != nil {
		t.Errorf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Errorf("Fail to start the component\n%v", err)
	}
	cases := []struct {
		endpoint string
		payload  string
	}{
		{
			endpoint: "/healthcheck/dns",
			payload:  `{"name":"foo","description":"bar","domain":"mcorbin.fr","interval":"10m","oneoff":false}`,
		},
		{
			endpoint: "/healthcheck/tcp",
			payload:  `{"name":"bar","description":"bar","domain":"mcorbin.fr","interval":"10m","oneoff":false,"target":"mcorbin.fr","port":9999,"timeout":"10s"}`,
		},
		{
			endpoint: "/healthcheck/http",
			payload:  `{"name":"baz","description":"bar","domain":"mcorbin.fr","interval":"10m","oneoff":false,"target":"mcorbin.fr","port":9999,"timeout":"10s","protocol":"http","validstatus":[200]}`,
		},
	}
	client := &http.Client{}
	for _, c := range cases {
		req, err := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:2000%s", c.endpoint), bytes.NewBuffer([]byte(c.payload)))
		req.Header.Set("Content-Type", "application/json")
		if err != nil {
			t.Errorf("Fail to build the HTTP request\n%v", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("HTTP request failed\n%v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("HTTP request failed, status %d", resp.StatusCode)
		}
	}
	if len(healthcheck.Healthchecks) != 3 {
		t.Errorf("Healthchecks were not successfully created: %d", len(healthcheck.Healthchecks))
	}
	err = component.Stop()
	if err != nil {
		t.Errorf("Fail to stop the component\n%v", err)
	}
}
