.DEFAULT_GOAL := help

help: ## Show all Makefile commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

tidy: ## Run go mod tidy
	go mod tidy

run: ## Run the server locally
	go run ./cmd/server

build: ## Build Go binary
	go build -o authllama ./cmd/server

test: ## Run all tests with coverage
	go test -coverprofile=coverage.out ./...

coverage: ## Open HTML coverage report
	go tool cover -html=coverage.out

clean: ## Remove coverage output
	rm -f coverage.out

race: ## Run race detector on tests
	go test -race ./...

publictest:
	curl -X POST http://localhost:3000/api/generate -H "Authorization: Bearer TWkj!8_CCNNdRoM@g.E" -H "Content-Type: application/json" -H "X-Service-Name: service-a" -d '{"model":"gemma:2b","prompt":"What is the capital of India?","stream":false}'
	curl -X POST http://localhost:3000/api/generate -H "Authorization: Bearer TWkj!8_CCNNdRoM@g.E" -H "Content-Type: application/json" -H "X-Service-Name: service-a" -d '{"model":"gemma:2b","prompt":"What is the capital of India?","stream":true}'

# Docker image registry settings
GITHUB_USER=sudhanshuraheja
IMAGE_NAME=authllama
GITHUB_REGISTRY=ghcr.io

docker_build: ## Build Docker image and load into local Docker (for local testing)
	docker buildx build --load -t $(IMAGE_NAME):latest .

docker_build_push: ## Build and push image to GitHub Container Registry
	docker buildx build --push -t $(GITHUB_REGISTRY)/$(GITHUB_USER)/$(IMAGE_NAME):latest .

docker_run: ## Run Docker container locally
	docker run -d --name authllama -p 8080:8080 -v $(pwd)/config:/app/config authllama

docker_pull: ## Pull image from GitHub Container Registry
	docker pull $(GITHUB_REGISTRY)/$(GITHUB_USER)/$(IMAGE_NAME)

docker_logs: ## Show logs from the authllama container
	docker logs -f authllama

docker_stop: ## Stop the running authllama container
	docker stop authllama || true

docker_rm: ## Remove the authllama container
	docker rm authllama || true

docker_login: ## Login to GitHub Container Registry
	echo $$GITHUB_TOKEN | docker login ghcr.io -u $(GITHUB_USER) --password-stdin

ci_push: docker_login docker_build_push ## Build and push (CI shortcut)

.PHONY: tidy run test coverage race clean help \
        docker_build docker_build_push docker_run docker_pull docker_logs docker_login \
		docker_stop docker_rm