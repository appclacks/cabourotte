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
				BaseConfig: healthcheck.BaseConfig{
					Name:        "foo",
					Description: "bar",
					Timeout:     healthcheck.Duration(time.Second * 5),
					Interval:    healthcheck.Duration(time.Second * 10),
				},
				Path:        "/foo",
				Target:      "mcorbin.fr",
				Port:        443,
				Protocol:    healthcheck.HTTPS,
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
				BaseConfig: healthcheck.BaseConfig{
					Name:        "foo",
					Description: "bar",
					Timeout:     healthcheck.Duration(time.Second * 5),
					Interval:    healthcheck.Duration(time.Second * 10),
				},
				Path:        "/foo",
				Target:      "mcorbin.fr",
				Port:        443,
				Protocol:    healthcheck.HTTPS,
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
				BaseConfig: healthcheck.BaseConfig{
					Name:        "toto",
					Description: "bar",
					Timeout:     healthcheck.Duration(time.Second * 5),
					Interval:    healthcheck.Duration(time.Second * 10),
				},
				Target: "mcorbin.fr",
				Port:   443,
			},
		},
		CommandChecks: []healthcheck.CommandHealthcheckConfiguration{
			healthcheck.CommandHealthcheckConfiguration{
				BaseConfig: healthcheck.BaseConfig{
					Name:        "command1",
					Description: "bar",
					Timeout:     healthcheck.Duration(time.Second * 3),
					Interval:    healthcheck.Duration(time.Second * 10),
					Labels: map[string]string{
						"type": "command",
					},
				},
				Command:   "ls",
				Arguments: []string{"-l", "/"},
			},
		},
		HTTPChecks: []healthcheck.HTTPHealthcheckConfiguration{
			healthcheck.HTTPHealthcheckConfiguration{
				BaseConfig: healthcheck.BaseConfig{
					Name:        "bar",
					Description: "bar",
					Timeout:     healthcheck.Duration(time.Second * 5),
					Interval:    healthcheck.Duration(time.Second * 10),
				},
				Path:        "/foo",
				Target:      "mcorbin.fr",
				Port:        80,
				Protocol:    healthcheck.HTTPS,
				ValidStatus: []uint{200, 201},
			},
			healthcheck.HTTPHealthcheckConfiguration{
				BaseConfig: healthcheck.BaseConfig{
					Name:        "bar3",
					Description: "bar",
					Timeout:     healthcheck.Duration(time.Second * 5),
					Interval:    healthcheck.Duration(time.Second * 10),
				},
				Path:        "/foo",
				Target:      "mcorbin.fr",
				Port:        80,
				Protocol:    healthcheck.HTTPS,
				ValidStatus: []uint{200, 201},
			},
		},
	})
	if component.Config.HTTP.Host != "127.0.0.2" {
		t.Fatal("Invalid HTTP address after reload")
	}
	if err != nil {
		t.Fatalf("Fail to reload the component\n%v", err)
	}
	if len(component.Healthcheck.ListChecks()) != 4 {
		t.Fatalf("The healthcheck was not added correctly")
	}
	dnsCheck := healthcheck.NewDNSHealthcheck(zap.NewExample(),
		&healthcheck.DNSHealthcheckConfiguration{
			BaseConfig: healthcheck.BaseConfig{
				Name:     "new-dns-check",
				Interval: healthcheck.Duration(time.Second * 10),
			},
		})
	err = component.Healthcheck.AddCheck(dnsCheck)
	if err != nil {
		t.Fatalf("Fail to add dns healthcheck")
	}
	if len(component.Healthcheck.ListChecks()) != 5 {
		t.Fatalf("The DNS healthcheck was not added correctly")
	}
	err = component.Healthcheck.AddCheck(dnsCheck)
	if err != nil {
		t.Fatalf("Fail to add dns healthcheck")
	}
	if len(component.Healthcheck.ListChecks()) != 5 {
		t.Fatalf("The DNS healthcheck was not overrided")
	}
	err = component.Reload(&Configuration{
		HTTP: http.Configuration{
			Host: "127.0.0.2",
			Port: 2002,
		},
	})
	if component.Config.HTTP.Host != "127.0.0.2" {
		t.Fatal("Invalid HTTP address after reload")
	}
	if err != nil {
		t.Fatalf("Fail to reload the component\n%v", err)
	}
	if len(component.Healthcheck.ListChecks()) != 1 {
		t.Fatalf("Only one check should exists")
	}
	if component.Healthcheck.ListChecks()[0].Name() != "new-dns-check" {
		t.Fatalf("Invalid name for the remaining healthcheck")
	}
	err = component.Stop()
	if err != nil {
		t.Fatalf("Fail to start the component\n%v", err)
	}
}
