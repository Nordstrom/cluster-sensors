FROM quay.io/nordstrom/baseimage-alpine:3.6
MAINTAINER Nordstrom Kubernetes Platform Team "techk8s@nordstrom.com"

ADD cluster-sensors /cluster-sensors

CMD [ "/cluster-sensors" ]
