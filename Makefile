# On macOS, use external linker to avoid dyld "missing LC_UUID" when running tests or binary.
LDFLAGS :=
ifeq ($(shell uname),Darwin)
  LDFLAGS := -ldflags=-linkmode=external
endif

.PHONY: build test test-short fmt vet clean web-install web-build build-all docker-build ci-go ci-web ci-docker

build:
	go build $(LDFLAGS) -o tracker ./cmd/tracker

test:
	go test $(LDFLAGS) ./...

test-short:
	go test $(LDFLAGS) -short ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -f tracker
	rm -rf web/dist

web-install:
	cd web && if [ -f package-lock.json ]; then npm ci; else npm install; fi

web-build:
	cd web && npm run build

build-all: web-build build

docker-build:
	docker build -t tracker:local .

ci-go:
	go test $(LDFLAGS) ./...
	go vet ./...

ci-web:
	cd web && npm ci
	cd web && npm run build

ci-docker:
	docker build -t tracker:local .
