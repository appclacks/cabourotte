package healthcheck

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"cabourotte/prometheus"
)

func TestIsSuccessfulOK(t *testing.T) {
	h := HTTPHealthcheck{
		Base: Base{
			Config: &HTTPHealthcheckConfiguration{
				ValidStatus: []uint{200},
			},
		},
	}
	response := http.Response{StatusCode: 200}
	if !h.isSuccessful(&response) {
		t.Fatalf("Invalid status check")
	}

	h = HTTPHealthcheck{
		Base: Base{
			Config: &HTTPHealthcheckConfiguration{
				ValidStatus: []uint{200, 201, 400},
			},
		},
	}
	response = http.Response{StatusCode: 400}
	if !h.isSuccessful(&response) {
		t.Fatalf("Invalid status check")
	}
}

func TestIssuccessfulFailure(t *testing.T) {
	h := HTTPHealthcheck{
		Base: Base{
			Config: &HTTPHealthcheckConfiguration{
				ValidStatus: []uint{200},
			},
		},
	}
	response := http.Response{StatusCode: 201}
	if h.isSuccessful(&response) {
		t.Fatalf("Invalid status check")
	}

	h = HTTPHealthcheck{
		Base: Base{
			Config: &HTTPHealthcheckConfiguration{
				ValidStatus: []uint{200, 201, 400},
			},
		},
	}
	response = http.Response{StatusCode: 500}
	if h.isSuccessful(&response) {
		t.Fatalf("Invalid status check")
	}
}

func TestHTTPExecuteGetSuccess(t *testing.T) {
	count := 0
	headersOK := false
	bodyOK := false
	expectedBody := "my custom body"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			if r.Header.Get("Foo") == "Bar" && r.Header.Get("User-agent") == "Cabourotte" {
				headersOK = true
			}
			bodyBytes, _ := ioutil.ReadAll(r.Body)
			body := string(bodyBytes)
			if body == expectedBody {
				bodyOK = true
			}
			count++
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := HTTPHealthcheck{
		Base: Base{
			Logger: zap.NewExample(),
			Config: &HTTPHealthcheckConfiguration{
				ValidStatus: []uint{200},
				Headers:     map[string]string{"Foo": "Bar"},
				Port:        uint(port),
				Target:      "127.0.0.1",
				Protocol:    HTTP,
				Body:        expectedBody,
				Path:        "/",
				BaseConfig: BaseConfig{
					Timeout: Duration(time.Second * 2),
				},
			},
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	err = h.Execute()
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
	if count != 1 {
		t.Fatal("The request counter is invalid")
	}
	if !headersOK {
		t.Fatal("Invalid headers")
	}
	if !bodyOK {
		t.Fatal("Invalid body")
	}
}

func TestHTTPExecuteRegexpSuccess(t *testing.T) {
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("cabourotte !"))
		if err != nil {
			t.Fatalf("Error writing :\n%v", err)
		}
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	r := regexp.MustCompile("cabourotte*")
	regexp := Regexp(*r)
	h := HTTPHealthcheck{
		Base: Base{
			Logger: zap.NewExample(),
			Config: &HTTPHealthcheckConfiguration{
				ValidStatus: []uint{200},
				Headers:     map[string]string{"Foo": "Bar"},
				Port:        uint(port),
				Target:      "127.0.0.1",
				BodyRegexp:  []Regexp{regexp},
				Protocol:    HTTP,
				Path:        "/",
				BaseConfig: BaseConfig{
					Timeout: Duration(time.Second * 2)},
			},
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	err = h.Execute()
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
	if count != 1 {
		t.Fatal("The request counter is invalid")
	}
}

func TestHTTPExecuteRegexpFailure(t *testing.T) {
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("cabourotte !"))
		if err != nil {
			t.Fatalf("Error writing :\n%v", err)
		}
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	r := regexp.MustCompile("trololo*")
	regexp := Regexp(*r)
	h := HTTPHealthcheck{
		Base: Base{
			Logger: zap.NewExample(),
			Config: &HTTPHealthcheckConfiguration{
				ValidStatus: []uint{200},
				Headers:     map[string]string{"Foo": "Bar"},
				Port:        uint(port),
				Target:      "127.0.0.1",
				BodyRegexp:  []Regexp{regexp},
				Protocol:    HTTP,
				Path:        "/",
				BaseConfig: BaseConfig{
					Timeout: Duration(time.Second * 2),
				},
			},
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	err = h.Execute()
	if err == nil {
		t.Fatalf("Was expecting an error")
	}

}

func TestHTTPv6ExecuteSuccess(t *testing.T) {
	count := 0
	l, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Error("fail to listen :\n/v", err)
	}
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	defer ts.Close()

	splitted := strings.Split(ts.URL, ":")
	port, err := strconv.ParseUint(splitted[len(splitted)-1], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := HTTPHealthcheck{
		Base: Base{
			Logger: zap.NewExample(),
			Config: &HTTPHealthcheckConfiguration{
				ValidStatus: []uint{200},
				Port:        uint(port),
				Target:      "::1",
				Protocol:    HTTP,
				Path:        "/",
				BaseConfig: BaseConfig{
					Timeout: Duration(time.Second * 2),
				},
			},
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	err = h.Execute()
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
	if count != 1 {
		t.Fatalf("The request counter is invalid")
	}
}

func TestHTTPExecuteFailure(t *testing.T) {
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := HTTPHealthcheck{
		Base: Base{
			Logger: zap.NewExample(),
			Config: &HTTPHealthcheckConfiguration{
				ValidStatus: []uint{200},
				Port:        uint(port),
				Target:      "127.0.0.1",
				Protocol:    HTTP,
				Path:        "/",
				BaseConfig: BaseConfig{
					Name:    "foo",
					Timeout: Duration(time.Second * 2)},
			},
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	err = h.Execute()
	if err == nil {
		t.Fatalf("Was expecting an error")
	}
	if count != 1 {
		t.Fatalf("The request counter is invalid")
	}
}

func TestHTTPBuildURL(t *testing.T) {
	h := HTTPHealthcheck{
		Base: Base{
			Config: &HTTPHealthcheckConfiguration{
				ValidStatus: []uint{200},
				Port:        2000,
				Target:      "127.0.0.1",
				Protocol:    HTTP,
				Path:        "/",
				BaseConfig: BaseConfig{
					Timeout: Duration(time.Second * 2),
				},
			},
		},
	}
	h.buildURL()
	expectedURL := "http://127.0.0.1:2000/"
	if h.URL != expectedURL {
		t.Fatalf("Invalid URL\nexpected: %s\nactual: %s", expectedURL, h.URL)
	}
}

func TestHTTPSBuildURL(t *testing.T) {
	h := HTTPHealthcheck{
		Base: Base{
			Config: &HTTPHealthcheckConfiguration{
				ValidStatus: []uint{200},
				Port:        2000,
				Target:      "127.0.0.1",
				Protocol:    HTTPS,
				Path:        "/foo",
				BaseConfig: BaseConfig{
					Timeout: Duration(time.Second * 2),
				},
			},
		},
	}
	h.buildURL()
	expectedURL := "https://127.0.0.1:2000/foo"
	if h.URL != expectedURL {
		t.Fatalf("Invalid URL\nexpected: %s\nactual: %s", expectedURL, h.URL)
	}
}

func TestHTTPStartStop(t *testing.T) {
	logger := zap.NewExample()
	healthcheck := NewHTTPHealthcheck(
		logger,
		&HTTPHealthcheckConfiguration{
			BaseConfig: BaseConfig{
				Name:        "foo",
				Description: "bar",
				Timeout:     Duration(time.Second * 3),
				Interval:    Duration(time.Second * 5),
				OneOff:      false,
			},
			Target:   "127.0.0.1",
			Path:     "/",
			Protocol: HTTP,
			Port:     9000,
		},
	)
	prom, err := prometheus.New()
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	component, err := New(zap.NewExample(), make(chan *Result, 10), prom)
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	component.startWrapper(healthcheck)
	err = healthcheck.Stop()
	if err != nil {
		t.Fatalf("Fail to stop the healthcheck\n%v", err)
	}
}

func TestHTTPExecuteSourceIP(t *testing.T) {
	count := 0
	headersOK := false
	bodyOK := false
	sourceIPOK := false
	expectedBody := "my custom body"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Foo") == "Bar" && r.Header.Get("User-agent") == "Cabourotte" {
			headersOK = true
		}
		bodyBytes, _ := ioutil.ReadAll(r.Body)
		body := string(bodyBytes)
		if body == expectedBody {
			bodyOK = true
		}
		if strings.Split(r.RemoteAddr, ":")[0] == "127.0.0.1" {
			sourceIPOK = true
		}
		count++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := HTTPHealthcheck{
		Base: Base{
			Logger: zap.NewExample(),
			Config: &HTTPHealthcheckConfiguration{
				SourceIP:    IP(net.ParseIP("127.0.0.1")),
				ValidStatus: []uint{200},
				Headers:     map[string]string{"Foo": "Bar"},
				Port:        uint(port),
				Target:      "127.0.0.1",
				Protocol:    HTTP,
				Body:        expectedBody,
				Path:        "/",
				BaseConfig: BaseConfig{
					Timeout: Duration(time.Second * 2),
				},
			},
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	err = h.Execute()
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
	if count != 1 {
		t.Fatal("The request counter is invalid")
	}
	if !headersOK {
		t.Fatal("Invalid headers")
	}
	if !bodyOK {
		t.Fatal("Invalid body")
	}
	if !sourceIPOK {
		t.Fatalf("Invalid source IP")
	}
}

func TestHTTPExecutePostSuccess(t *testing.T) {
	count := 0
	headersOK := false
	bodyOK := false
	expectedBody := "my custom body"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			if r.Header.Get("Foo") == "Bar" && r.Header.Get("User-agent") == "Cabourotte" {
				headersOK = true
			}
			bodyBytes, _ := ioutil.ReadAll(r.Body)
			body := string(bodyBytes)
			if body == expectedBody {
				bodyOK = true
			}
			count++
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := HTTPHealthcheck{
		Base: Base{
			Logger: zap.NewExample(),
			Config: &HTTPHealthcheckConfiguration{
				ValidStatus: []uint{200},
				Headers:     map[string]string{"Foo": "Bar"},
				Port:        uint(port),
				Target:      "127.0.0.1",
				Method:      "POST",
				Protocol:    HTTP,
				Body:        expectedBody,
				Path:        "/",
				BaseConfig: BaseConfig{
					Timeout: Duration(time.Second * 2),
				},
			},
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	err = h.Execute()
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
	if count != 1 {
		t.Fatal("The request counter is invalid")
	}
	if !headersOK {
		t.Fatal("Invalid headers")
	}
	if !bodyOK {
		t.Fatal("Invalid body")
	}
}
