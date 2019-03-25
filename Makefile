PROJ=edgestore
ORG_PATH=github.com/edgestore
REPO_PATH=$(ORG_PATH)/$(PROJ)
export PATH := $(PWD)/bin:$(PATH)

DOCKER_IMAGE=$(PROJ)

$( shell mkdir -p bin )
$( shell mkdir -p release/bin )
$( shell mkdir -p release/images )
$( shell mkdir -p results )

user=$(shell id -u -n)
group=$(shell id -g -n)

export GOBIN=$(PWD)/bin
# Prefer ./bin instead of system packages for things like protoc, where we want
# to use the version Envoyee uses, not whatever a developer has installed.
export PATH=$(GOBIN):$(shell printenv PATH)

# Version
VERSION ?= $(shell git rev-parse --abbrev-ref HEAD)
COMMIT_HASH ?= $(shell git rev-parse HEAD 2>/dev/null)
BUILD_TIME ?= $(shell date +%FT%T%z)

LD_FLAGS="-w -X $(REPO_PATH)/version.BuildTime=$(BUILD_TIME) -X $(REPO_PATH)/version.CommitHash=$(COMMIT_HASH) -X $(REPO_PATH)/version.Version=$(VERSION)"
LDFLAGS += -X main.Version=${VERSION} -X main.CommitHash=${COMMIT_HASH} -X main.BuildDate=${BUILD_DATE}

# Inject env file
include .env
export $(shell sed 's/=.*//' .env)

build: clean bin/master

bin/master:
	@echo "Building Master Service"
	@go install -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/master

clean:
	@echo "Cleaning Binary Folders"
	@rm -rf bin/*
	@rm -rf release/*
	@rm -rf results/*

release-binary:
	@echo "Releasing binary files"
	@go build -race -o release/bin/master -v -ldflags $(LD_FLAGS) $(REPO_PATH)/cmd/master

revendor:
	@echo "Install dependencies"
	@go get -v ./...

master-serve: build
	@echo "Running Master Service"
	@bin/master serve

.PHONY: docker-image
docker-image: clean
	@echo "Building $(DOCKER_IMAGE) image"
	@docker build -t $(DOCKER_IMAGE) --rm -f Dockerfile .

test:
	@echo "Testing"
	@go test -v --short -race ./...

testcoverage:
	@echo "Testing with coverage"
	@mkdir -p results
	@go test -v $(REPO_PATH)/... | go2xunit -output results/tests.xml
	@gocov test $(REPO_PATH)/... | gocov-xml > results/cobertura-coverage.xml

testrace:
	@echo "Testing with race detection"
	@go test -v --race $(REPO_PATH)/...

vet:
	@echo "Running go tool vet on packages"
	go vet $(REPO_PATH)/...

fmt:
	@echo "Running gofmt on package sources"
	go fmt $(REPO_PATH)/...

lint:
	@echo "Lint"
	go get -v github.com/golang/lint/golint
	for file in $$(find . -name '*.go' | grep -v '\.pb\.go\|\.pb\.gw\.go'); do \
		golint $${file}; \
		if [ -n "$$(golint $${file})" ]; then \
			exit 1; \
		fi; \
	done

testall: testcoverage testrace vet fmt # lint

.PHONY: fmt \
		lint \
		release-binary \
		revendor \
		test \
		testall \
		testcoverage \
		testrace \
		vet
