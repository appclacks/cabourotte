package discovery

import (
	"testing"

	"go.uber.org/zap"

	"github.com/mcorbin/cabourotte/healthcheck"
	"github.com/mcorbin/cabourotte/prometheus"
)

func TestAddHealthcheck(t *testing.T) {
	logger := zap.NewExample()
	prom, err := prometheus.New()
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	component, err := healthcheck.New(logger, make(chan *healthcheck.Result, 10), prom)
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Fatalf("Fail to start the component\n%v", err)
	}
	newChecks := make(map[string]bool)
	configString := "{\"name\":\"mcorbin-http-check\",\"description\":\"http healthcheck example\",\"target\":\"mcorbin.fr\",\"interval\":\"5s\",\"timeout\": \"3s\",\"port\":443,\"protocol\":\"https\",\"valid-status\":[200]}"
	err = addCheck(component, logger, newChecks, "http", configString, "", healthcheck.SourceKubernetesPod, nil)
	if err != nil {
		t.Fatalf("Fail to add the check\n%v", err)
	}
	listResult := component.ListChecks()
	if len(listResult) != 1 {
		t.Fatalf("The healthcheck is not in the healthcheck list")
	}
	check, err := component.GetCheck("mcorbin-http-check")
	if err != nil {
		t.Fatalf("Fail to get the check\n%v", err)
	}
	config := check.GetConfig().(*healthcheck.HTTPHealthcheckConfiguration)
	if config.Target != "mcorbin.fr" {
		t.Fatalf("Invalid target %s", config.Target)
	}

	configString = "{\"name\":\"mcorbin-tcp-check\",\"description\":\"tcp healthcheck example\",\"interval\":\"5s\",\"timeout\": \"3s\",\"port\":443}"
	err = addCheck(component, logger, newChecks, "tcp", configString, "test.mcorbin.fr", healthcheck.SourceKubernetesPod, nil)
	if err != nil {
		t.Fatalf("Fail to add the check\n%v", err)
	}
	listResult = component.ListChecks()
	if len(listResult) != 2 {
		t.Fatalf("The healthcheck is not in the healthcheck list")
	}
	check, err = component.GetCheck("mcorbin-tcp-check")
	if err != nil {
		t.Fatalf("Fail to get the check\n%v", err)
	}
	tcpConfig := check.GetConfig().(*healthcheck.TCPHealthcheckConfiguration)
	if tcpConfig.Target != "test.mcorbin.fr" {
		t.Fatalf("Invalid target %s", tcpConfig.Target)
	}
}
