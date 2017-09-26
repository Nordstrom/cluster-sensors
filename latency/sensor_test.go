package latency

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBuildRequest(t *testing.T) {
	Convey("converts a string-based URL to http request", t, func() {
		So(buildRequest("http://google.com").URL.String(), ShouldEqual, "http://google.com")
	})

	Convey("is a GET request", t, func() {
		So(buildRequest("http://localhost").Method, ShouldEqual, "GET")
	})

	Convey("requests connection to close when done", t, func() {
		So(buildRequest("http://nowhere").Close, ShouldBeTrue)
	})
}

func BenchmarkMemory(b *testing.B) {
	paths := initPaths()
	for i := 0; i < b.N; i++ {
		go paths[0].measureLatency()
	}
}

func BenchmarkMeasureLatency(b *testing.B) {
	r := buildRequest("http://default-http-backend.athena.platform.r53.nordstrom.net")
	if r == nil {
		b.Fail()
	}
	p := path{
		name:    "ingress",
		request: r,
	}
	for i := 0; i < b.N; i++ {
		_, _ = p.measureLatency()
	}
}

func TestMeasureLatency(t *testing.T) {
	p := path{
		name:    "ingress",
		request: buildRequest("http://default-http-backend.athena.platform.r53.nordstrom.net"),
	}
	for i := 0; i < 50; i++ {
		meas, _ := p.measureLatency()
		fmt.Printf("%v\n", meas.dnsTime.Seconds())
	}
}
