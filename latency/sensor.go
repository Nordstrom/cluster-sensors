package latency

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Sensor struct{}

func (s Sensor) Start() {
	log.Println("Starting latency sensor")
	metrics := initMetrics()
	paths := initPaths()
	millisecondsBetweenRequestsAsString := os.Getenv("LATENCY_MILLISECONDS_BETWEEN_REQUESTS")
	millisecondsBetweenRequests, err := strconv.Atoi(millisecondsBetweenRequestsAsString)
	if err != nil {
		log.Fatal("latency sensor failed to parse environment variable:", millisecondsBetweenRequestsAsString)
	}
	for {
		for _, p := range paths {
			go p.measureLatencyAndRecord(metrics)
		}
		time.Sleep(time.Duration(millisecondsBetweenRequests) * time.Millisecond)
	}
}

func initPaths() []path {
	return []path{
		{name: "ingress", request: buildRequest(os.Getenv("LATENCY_INGRESS_URL"))},
		{name: "loadbalancer", request: buildRequest(os.Getenv("LATENCY_LOADBALANCER_URL"))},
		{name: "internal", request: buildRequest(os.Getenv("LATENCY_INTERNAL_URL"))},
	}
}

func buildRequest(urlStr string) *http.Request {
	request, _ := http.NewRequest("GET", urlStr, nil)
	request.Close = true
	return request
}

func initMetrics() (m metricsType) {
	m.histogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "sensors_latency_milliseconds_histogram",
		Help: "Histogram of latency in milliseconds. Stages not intended to be aggregated as they measure very different things",
		Buckets: []float64{
			1,
			2,
			3,
			4,
			5,
			6,
			7,
			8,
			9,
			10,
			11,
			12,
			13,
			14,
			15,
			16,
			17,
			18,
			19,
			20,
			25,
			30,
			35,
			40,
			45,
			50,
			55,
			60,
			65,
			70,
			75,
			80,
			85,
			90,
			95,
			100,
			110,
			120,
			130,
			140,
			150,
			160,
			170,
			180,
			190,
			200,
			250,
			300,
			350,
			400,
			450,
			500,
			600,
			700,
			800,
			900,
			1000,
			2000,
			3000,
			4000,
			5000,
			10000,
		},
	}, []string{"stage", "path", "backend"})
	prometheus.MustRegister(m.histogram)

	m.errors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "sensors_latency_errors",
		Help: "Number of times there was an error",
	}, []string{"stage", "path", "backend"})

	prometheus.MustRegister(m.errors)
	return
}

type metricsType struct {
	histogram *prometheus.HistogramVec
	errors    *prometheus.CounterVec
}

type path struct {
	name    string
	request *http.Request
	lock    sync.Mutex
}

func (p path) measureLatencyAndRecord(metrics metricsType) {
	measurement, err := p.measureLatency()
	if err != nil {
		stage := "without_name_lookup"
		if measurement.dnsErr != nil {
			stage = "name_lookup"
		}
		metrics.errors.With(prometheus.Labels{"stage": stage, "path": p.name, "backend": measurement.backendAddress.String()}).Inc()
		return
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	measurement.Record(metrics.histogram, p.name)
}

func (p path) measureLatency() (measurement results, err error) {
	var dnsStart, dnsDone, firstByte time.Time
	trace := &httptrace.ClientTrace{
		DNSStart: func(dnsInfo httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			dnsDone = time.Now() // Measure time immediately and once
			if dnsInfo.Err != nil {
				measurement.dnsErr = err
			}
		},
		GotFirstResponseByte: func() {
			firstByte = time.Now()
		},
	}

	request := p.request.WithContext(httptrace.WithClientTrace(p.request.Context(), trace))
	response, err := http.DefaultTransport.RoundTrip(request)
	if err != nil {
		log.Println("error measuring latency: RoundTrip:", p, err)
		return
	}
	// If we measured time from start of connection we are leaving a larger opportunity
	// where a coordinated delay could selectively omit bad data. We are still
	// vulnerable to this until we use an external observer.
	// One way to beat coordinated ommission is if we started dns time from the last recorded observation, compensating for the sleep time between measurements.
	// That would eliminate gaps in data between measurements except for slowdowns in the system clock itself.
	measurement.dnsTime = dnsDone.Sub(dnsStart)
	measurement.dnsDoneToCloseTime = firstByte.Sub(dnsDone)
	response.Body.Close()

	if response.StatusCode != 200 && response.StatusCode != 404 {
		err = fmt.Errorf("unexpected status code while measuring latency: %v, %d", p, response.StatusCode)
		log.Println(err)
		return
	}
	backendAddr, err := parseBackend(response.Header.Get("X-Backend-Server"))
	if err != nil {
		log.Println(err)
		return
	}
	measurement.backendAddress = *backendAddr
	//TODO: Measure error latencies

	return
}

func parseBackend(backendHostPort string) (*net.TCPAddr, error) {
	backendHost, backendPort, err := net.SplitHostPort(backendHostPort)
	if err != nil {
		return &net.TCPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 0,
		}, nil
	}
	backendIPAddress := net.ParseIP(backendHost)
	if backendIPAddress == nil {
		return nil, fmt.Errorf("Backend host not a valid IP address: \"%s\". Response: %v", backendIPAddress)
	}
	backendPortNum, err := strconv.Atoi(backendPort)
	if err != nil {
		return nil, fmt.Errorf("Backend host not a valid TCP port: %s", backendPort)
	}
	return &net.TCPAddr{
		IP:   backendIPAddress,
		Port: backendPortNum,
	}, nil
}

type results struct {
	dnsTime            time.Duration
	dnsDoneToCloseTime time.Duration
	dnsErr             error
	backendAddress     net.TCPAddr
}

func (r results) Record(summary *prometheus.HistogramVec, pathName string) {
	summary.With(prometheus.Labels{"stage": "name_lookup", "path": pathName, "backend": ""}).Observe(r.NameLookupTimeMilliseconds())
	summary.With(prometheus.Labels{"stage": "without_name_lookup", "path": pathName, "backend": r.backendAddress.String()}).Observe(r.TimeWithoutNameLookup())
}

func (r results) NameLookupTimeMilliseconds() float64 {
	return r.dnsTime.Seconds() * 1e3
}

func (r results) TimeWithoutNameLookup() float64 {
	return r.dnsDoneToCloseTime.Seconds() * 1e3
}
