package daemon

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"cabourotte/healthcheck"
	"cabourotte/http"
)

func TestNewStop(t *testing.T) {
	component, err := New(zap.NewExample(), &Configuration{
		HTTP: http.Configuration{
			Host: "127.0.0.1",
			Port: 2002,
		},
	})
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	err = component.Stop()
	if err != nil {
		t.Fatalf("Fail to start the component\n%v", err)
	}
}

func TestReload(t *testing.T) {
	component, err := New(zap.NewExample(), &Configuration{
		HTTP: http.Configuration{
			Host: "127.0.0.1",
			Port: 2002,
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
	})
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}

	if len(component.Healthcheck.ListChecks()) != 1 {
		t.Fatalf("The healthcheck was not added correctly")
	}

	err = component.Reload(&Configuration{
		HTTP: http.Configuration{
			Host: "127.0.0.1",
			Port: 2002,
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
	})
	if err != nil {
		t.Fatalf("Fail to reload the component\n%v", err)
	}
	if len(component.Healthcheck.ListChecks()) != 1 {
		t.Fatalf("The healthcheck was not added correctly")
	}
	err = component.Reload(&Configuration{
		HTTP: http.Configuration{
			Host: "127.0.0.2",
			Port: 2002,
		},
		TCPChecks: []healthcheck.TCPHealthcheckConfiguration{
			healthcheck.TCPHealthcheckConfiguration{
				Name:        "toto",
				Description: "bar",
				Target:      "mcorbin.fr",
				Port:        443,
				Timeout:     healthcheck.Duration(time.Second * 5),
				Interval:    healthcheck.Duration(time.Second * 10),
			},
		},
		HTTPChecks: []healthcheck.HTTPHealthcheckConfiguration{
			healthcheck.HTTPHealthcheckConfiguration{
				Name:        "bar",
				Description: "bar",
				Path:        "/foo",
				Target:      "mcorbin.fr",
				Port:        80,
				Protocol:    healthcheck.HTTPS,
				Timeout:     healthcheck.Duration(time.Second * 5),
				Interval:    healthcheck.Duration(time.Second * 10),
				ValidStatus: []uint{200, 201},
			},
			healthcheck.HTTPHealthcheckConfiguration{
				Name:        "bar3",
				Description: "bar",
				Path:        "/foo",
				Target:      "mcorbin.fr",
				Port:        80,
				Protocol:    healthcheck.HTTPS,
				Timeout:     healthcheck.Duration(time.Second * 5),
				Interval:    healthcheck.Duration(time.Second * 10),
				ValidStatus: []uint{200, 201},
			},
		},
	})
	if err != nil {
		t.Fatalf("Fail to reload the component\n%v", err)
	}
	if len(component.Healthcheck.ListChecks()) != 3 {
		t.Fatalf("The healthcheck was not added correctly")
	}
	err = component.Stop()
	if err != nil {
		t.Fatalf("Fail to start the component\n%v", err)
	}
}
