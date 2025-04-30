tidy:
	go mod tidy

run:
	go run ./cmd/server

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

race:
	go test -race ./...