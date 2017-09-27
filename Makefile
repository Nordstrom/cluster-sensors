build := build
git_sha_commit := $(shell git rev-parse --short HEAD)
local_changes := $(shell git status --porcelain)

export app_name := cluster-sensors
export namespace := utils
image_name := quay.io/nordstrom/$(app_name)
image_tag := v0.0.1

.PHONY: build_app test refresh_ecr_token
.PHONY: build_image push_image deploy teardown clean

build_app: $(build)/$(app_name) $(build)/Dockerfile

$(build):
	mkdir -p "$@"

$(build)/$(app_name): *.go | $(build)
	# Build your golang app for the target OS
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $@

$(build)/Dockerfile: Dockerfile
		cp Dockerfile "$@"

$(app_name): *.go | $(build)
	# Build golang app for local OS
	go build -o $(app_name) -ldflags "-X main.Version=$(image_tag)"

test:
	go test -v `go list ./... | grep -v /vendor/`

build_image: $(build)/$(app_name) $(build)/Dockerfile | $(build)
	docker build -t $(image_name):$(image_tag) $(build)

push_image: build_image refresh_ecr_token
	docker push $(image_name):$(image_tag)

clean:
	rm -rf $(build)
