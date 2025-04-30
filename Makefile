tidy:
	go mod tidy

run:
	go run ./cmd/server

test:
	go test -coverprofile=coverage.out ./...

coverage:
	go tool cover -html=coverage.out

race:
	go test -race ./...