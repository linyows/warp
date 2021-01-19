default: build

build:
	env GOOS=linux GOARCH=amd64 go build -o warp ./cmd/warp/main.go

test:
	go test -v -cover
