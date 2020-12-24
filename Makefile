PROJECT ?= chipmusic-cli
VERSION ?= development
TARGET ?= target

GOOS ?= darwin
GOARCH ?= amd64
LDFLAGS ?= -X 'main.version=$(VERSION)'
BINARY ?= $(PROJECT)-$(GOOS)-$(GOARCH)

.PHONY: build
build:
	GO111MODULE=on GOOS="$(GOOS)" GOARCH="$(GOARCH)" go build -ldflags "$(LDFLAGS)" -o "$(TARGET)/$(BINARY)" main.go
	chmod u+x "$(TARGET)/$(BINARY)"

.PHONY: test
test:
	GO111MODULE=on go test -v -cover ./...

.PHONY: generate
generate:
	GO111MODULE=off go generate ./...

.PHONY: tools
tools:
	go list -f '{{range .Imports}}{{.}} {{end}}' tools.go | xargs go install

.PHONY: clean
clean:
	rm -rf "$(TARGET)"
