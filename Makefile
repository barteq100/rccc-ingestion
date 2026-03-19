run:
	go run ./cmd/worker

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal
