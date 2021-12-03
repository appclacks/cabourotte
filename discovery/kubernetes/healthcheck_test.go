package kubernetes

import (
	"reflect"
	"strings"
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
	labels := map[string]string{"foo": "bar"}
	configString := "{\"name\":\"mcorbin-http-check\",\"description\":\"http healthcheck example\",\"target\":\"mcorbin.fr\",\"interval\":\"5s\",\"timeout\": \"3s\",\"port\":443,\"protocol\":\"https\",\"valid-status\":[200]}"
	err = addCheck(component, logger, newChecks, "http", configString, "", healthcheck.SourceKubernetesPod, labels, false)
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
	if !reflect.DeepEqual(config.Labels, labels) {
		t.Fatalf("Invalid labels")
	}

	configString = "{\"name\":\"mcorbin-tcp-check\",\"description\":\"tcp healthcheck example\",\"interval\":\"5s\",\"timeout\": \"3s\",\"port\":443}"
	err = addCheck(component, logger, newChecks, "tcp", configString, "test.mcorbin.fr", healthcheck.SourceKubernetesPod, nil, false)
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

	configString = "{\"name\":\"mcorbin-command-check\",\"description\":\"command healthcheck example\",\"interval\":\"5s\",\"timeout\": \"3s\",\"command\":\"ls\"}"
	err = addCheck(component, logger, newChecks, "command", configString, "", healthcheck.SourceKubernetesPod, nil, true)
	if err == nil {
		t.Fatalf("Was expecting an error: commands checks are disabled\n%v", err)
	}
	if !strings.Contains(err.Error(), "Command checks are not allowed") {
		t.Fatalf("Invalid error message %s", err.Error())
	}
}
