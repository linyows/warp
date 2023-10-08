SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=

GOOS?=linux
GOARCH?=amd64

default: build

setup:
	go mod download

build:
	env GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go build -o warp ./cmd/warp/main.go

run:
	env GOOS=$(GOOS) GOARCH=$(GOARCH) go run ./cmd/warp/main.go

test:
	go test -v -short ./...

integration: key
	go test -v -run TestIntegration

test-all:
	go test $(TEST_OPTIONS) -failfast -race -coverpkg=./... -covermode=atomic -coverprofile=coverage.txt $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=5m

slqck-plugin:
	env GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go build -buildmode=plugin -o plugins/slack.so plugins/slack/main.go

key:
	@rm -rf testdata/server.*
	@openssl req -x509 -days 10 -newkey ED25519 -nodes -out ./testdata/server.crt -keyout ./testdata/server.key -subj "/C=/ST=/L=/O=/OU=/CN=example.local" &>/dev/null

release:
	@test -z $(GITHUB_TOKEN) || goreleaser --rm-dist

dist:
	goreleaser --snapshot --skip-publish --rm-dist

clean:
	rm -rf plugin/*.so

.PHONY: plugin integration
