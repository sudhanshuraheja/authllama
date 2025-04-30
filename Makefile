tidy:
	go mod tidy

run:
	go run ./cmd/server

test:
	go test ./...
	go test -race ./...