tidy:
	go mod tidy

run:
	go run ./cmd/server

test:
	curl -H "X-Service-Name: service-a" -H "Authorization: wrong-token" http://localhost:8080/
	curl -H "X-Service-Name: service-a" -H "Authorization: Bearer abc123" http://localhost:8080/
	curl -X POST http://localhost:8080/api/generate -H "X-Service-Name: service-a" -H "Authorization: Bearer abc123" -d '{"model": "gemma:2b", "prompt": "What is Go?"}'
	curl -X POST http://localhost:8080/api/chat -H "X-Service-Name: service-a" -H "Authorization: Bearer abc123" -H "Content-Type: application/json" -d '{"model": "gemma:2b", "messages": [{"role": "user", "content": "What is Go?"}]}'