package healthcheck

import (
	"context"
	"net"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/appclacks/cabourotte/prometheus"
)

func TestDNSExecuteSuccess(t *testing.T) {
	h := DNSHealthcheck{
		Logger: zap.NewExample(),
		Config: &DNSHealthcheckConfiguration{
			// it will hopefully resolve ^^
			Domain:  "mcorbin.fr",
			Timeout: Duration(time.Second * 2),
		},
	}
	ctx := context.Background()
	err := h.Execute(&ctx)
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
}

func TestDNSExecuteFailure(t *testing.T) {
	h := DNSHealthcheck{
		Logger: zap.NewExample(),
		Config: &DNSHealthcheckConfiguration{
			Domain:  "doesnotexist.mcorbin.fr",
			Timeout: Duration(time.Second * 2),
		},
	}
	ctx := context.Background()
	err := h.Execute(&ctx)
	if err == nil {
		t.Fatalf("Was expecting an error: the domain does not exist")
	}
}

func TestDNSStartStop(t *testing.T) {
	logger := zap.NewExample()
	healthcheck := NewDNSHealthcheck(
		logger,
		&DNSHealthcheckConfiguration{
			Base: Base{
				Name:        "foo",
				Description: "bar",
				Interval:    Duration(time.Second * 5),
				OneOff:      false,
			},
			Timeout: Duration(time.Second * 2),
			Domain:  "mcorbin.fr",
		},
	)
	wrapper := NewWrapper(healthcheck)
	prom, err := prometheus.New()
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	component, err := New(zap.NewExample(), make(chan *Result, 10), prom, []string{})
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	component.startWrapper(wrapper)
	err = wrapper.Stop()
	if err != nil {
		t.Fatalf("Fail to stop the healthcheck\n%v", err)
	}
}

func TestVerifyIPs(t *testing.T) {
	lookupIPs := []net.IP{net.ParseIP("10.0.0.1")}
	expectedIPs := []IP{IP(net.ParseIP("10.0.0.1"))}
	err := verifyIPs(expectedIPs, lookupIPs)
	if err != nil {
		t.Fatalf("Fail verify the IP\n%v", err)
	}

	lookupIPs = []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.3")}
	expectedIPs = []IP{IP(net.ParseIP("10.0.0.1"))}
	err = verifyIPs(expectedIPs, lookupIPs)
	if err != nil {
		t.Fatalf("Fail verify the IP\n%v", err)
	}

	lookupIPs = []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.3")}
	expectedIPs = []IP{IP(net.ParseIP("10.0.0.3"))}
	err = verifyIPs(expectedIPs, lookupIPs)
	if err != nil {
		t.Fatalf("Fail verify the IP\n%v", err)
	}

	lookupIPs = []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.3")}
	expectedIPs = []IP{IP(net.ParseIP("10.0.0.3")), IP(net.ParseIP("10.0.0.1"))}
	err = verifyIPs(expectedIPs, lookupIPs)
	if err != nil {
		t.Fatalf("Fail verify the IP\n%v", err)
	}

	lookupIPs = []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.3")}
	expectedIPs = []IP{IP(net.ParseIP("10.0.0.4"))}
	err = verifyIPs(expectedIPs, lookupIPs)
	if err == nil {
		t.Fatalf("Was expecting an error")
	}

	lookupIPs = []net.IP{net.ParseIP("10.0.0.1"), net.ParseIP("10.0.0.3")}
	expectedIPs = []IP{IP(net.ParseIP("10.0.0.1")), IP(net.ParseIP("10.0.0.4"))}
	err = verifyIPs(expectedIPs, lookupIPs)
	if err == nil {
		t.Fatalf("Was expecting an error")
	}
}
