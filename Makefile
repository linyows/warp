SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=

GOOS?=linux
GOARCH?=amd64

default: build

setup:
	go mod download

build:
	env GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o warp ./cmd/warp/main.go

run:
	env GOOS=$(GOOS) GOARCH=$(GOARCH) go run ./cmd/warp/main.go

test:
	@go test -v $(shell go list ./... | grep -v integration)

integration:
	go test -v ./integration

test-all:
	go test $(TEST_OPTIONS) -failfast -race -coverpkg=./... -covermode=atomic -coverprofile=coverage.txt $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=5m

mysql-plugin:
	env GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go build -buildmode=plugin -o plugin/mysql.so plugin/mysql/main.go

file-plugin:
	env GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 go build -buildmode=plugin -o plugin/file.so plugin/file/main.go

release:
	@test -z $(GITHUB_TOKEN) || goreleaser --rm-dist

dist:
	goreleaser --snapshot --skip-publish --rm-dist

clean:
	rm -rf plugin/*.so

.PHONY: plugin integration
