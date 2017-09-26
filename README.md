# Cluster Sensors

The Cluster Sensors Service.

## Description
Runs a bunch of sensors and generate metrics for them. Currently, the only sensor implemented is latency sensor.

## Latency Sensor
The latency sensor calls 3 endpoints every 'n'(specified by LATENCY_MILLISECONDS_BETWEEN_REQUESTS environment variable) milliseconds:
  - Kubernetes ingress endpoint(specified by LATENCY_INGRESS_URL env var)
  - Kubernetes service endpoint(specified by LATENCY_INTERNAL_URL env var)
  - Loadbalancer endpoint(specified by LATENCY_LOADBALANCER_URL env var)

It measures the latency for DNS lookup and the first response byte for each request for each endpoint.

## Metrics exposed
The cluster sensor exposes the following prometheus metrics:

- sensors_latency_milliseconds_histogram: Histogram of latency in milliseconds. Stages not intended to be aggregated as they measure very different things
- sensors_latency_errors: Number of times there was an error

## How to Deploy
You can use [helm] (https://github.com/kubernetes/helm) to deploy the Cluster Sensors service to a kubernetes cluster as follows:

```
helm upgrade --namespace=<namespace> --install \
  --kube-context <cluster_id> \
  --set="latency.ingress.url=<ingress_url>" \
  --set="latency.internal.url=<service_url>" \
  --set="latency.loadbalancer.url=<loadbalancer_url>" \
  --set="image=<image_tag>" \
  sensors ./cluster-sensors
```
