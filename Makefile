SOURCE_FILES?=./...
TEST_PATTERN?=.
TEST_OPTIONS?=

default: build

setup:
	go mod download

build:
	env GOOS=linux GOARCH=amd64 go build -o warp ./cmd/warp/main.go

test:
	go test $(TEST_OPTIONS) -failfast -race -coverpkg=./... -covermode=atomic \
		-coverprofile=coverage.txt $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=5m

release:
	@test -z $(GITHUB_TOKEN) || goreleaser --snapshot --rm-dist

dist:
	goreleaser --snapshot --skip-publish --rm-dist
