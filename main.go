package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/nordstrom/cluster-sensors/latency"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	config := getFlags()
	startSensors()
	log.Println("Listening on port", config.port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.port), getRoutes()))
}

func getFlags() (f flags) {
	flag.IntVar(&f.port, "port", 8080, "listening port")
	flag.Parse()
	return
}

type flags struct {
	port int
}

func startSensors() {
	for _, s := range sensorRegistry {
		go s.Start()
	}
}

// TODO consider making the sensor not resonsible for looping / sleeping but rather just measurement
type sensor interface {
	Start()
}

var sensorRegistry = []sensor{
	latency.Sensor{},
}

func getRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/metrics", prometheus.Handler())
	return mux
}
