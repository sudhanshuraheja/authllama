tidy:
	go mod tidy

test:
	curl -H "X-Service-Name: service-a" -H "Authorization: wrong-token" http://localhost:8080/
	curl -H "X-Service-Name: service-a" -H "Authorization: Bearer abc123" http://localhost:8080/