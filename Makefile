BINARY_NAME=gospy

.PHONY: build build-linux build-darwin clean install-deps

build: install-deps
	CGO_ENABLED=1 go build -o $(BINARY_NAME) .

build-linux: build-linux-amd64 build-linux-arm64

build-linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BINARY_NAME)-linux-amd64 .

build-linux-arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(BINARY_NAME)-linux-arm64 .

build-darwin:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -o $(BINARY_NAME)-darwin-arm64 .

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux-amd64 $(BINARY_NAME)-linux-arm64 $(BINARY_NAME)-darwin-*

install-deps:
	go mod tidy

ai:
	aider

# Docker操作
docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

test-servers: docker-build docker-up
	@echo "  AMD64: http://localhost:8080" 
	@echo "  ARM64: http://localhost:8081"

test-summary-in-docker: build-linux-arm64 docker-up
	@echo "Copying binary to ARM64 container..."
	@docker cp $(BINARY_NAME)-linux-arm64 gospyv2-test-server-1:/gospy
	@echo "Getting test server PID..."
	@$(eval PID := $(shell curl -s http://localhost:8081/ | jq -r .pid))
	@echo "Running summary against pid $(PID)..."
	@docker exec gospyv2-test-server-1 /gospy summary --pid $(PID)
