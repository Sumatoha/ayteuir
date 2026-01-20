.PHONY: build run test clean dev docker-build docker-run lint lambda-build deploy build-AyteuirFunction

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
BINARY_NAME=api
LAMBDA_BINARY=bootstrap
MAIN_PATH=./cmd/api
LAMBDA_PATH=./cmd/lambda

# ==========================================
# LOCAL DEVELOPMENT
# ==========================================

# Build the binary for local development
build:
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)

# Run the application locally
run: build
	./$(BINARY_NAME)

# Development mode with hot reload (requires air)
dev:
	air

# ==========================================
# TESTING
# ==========================================

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# ==========================================
# AWS LAMBDA
# ==========================================

# Build Lambda binary (Linux ARM64 for Graviton)
lambda-build:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -tags lambda.norpc -ldflags="-s -w" -o $(LAMBDA_BINARY) $(LAMBDA_PATH)

# SAM build target (required for BuildMethod: makefile)
build-AyteuirFunction:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -tags lambda.norpc -ldflags="-s -w" -o $(ARTIFACTS_DIR)/bootstrap $(LAMBDA_PATH)

# Build with SAM
sam-build:
	sam build

# Deploy to AWS (interactive first time)
deploy: sam-build
	sam deploy --guided

# Deploy to production
deploy-prod: sam-build
	sam deploy --config-env production

# Deploy to staging
deploy-staging: sam-build
	sam deploy --config-env staging

# Local Lambda testing with SAM
sam-local:
	sam local start-api --env-vars env.json

# Invoke Lambda locally
sam-invoke:
	sam local invoke AyteuirFunction --event events/api-gateway-event.json

# View Lambda logs
logs:
	sam logs -n AyteuirFunction --stack-name ayteuir-api --tail

# Delete stack
destroy:
	sam delete --stack-name ayteuir-api

# ==========================================
# DOCKER (for local testing)
# ==========================================

# Build Docker image
docker-build:
	docker build -t ayteuir-api .

# Run Docker container
docker-run:
	docker run -p 8080:8080 --env-file .env ayteuir-api

# ==========================================
# UTILITIES
# ==========================================

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME) $(LAMBDA_BINARY)
	rm -f coverage.out coverage.html
	rm -rf .aws-sam

# Download dependencies
deps:
	$(GOMOD) download

# Tidy dependencies
tidy:
	$(GOMOD) tidy

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Format code
fmt:
	$(GOCMD) fmt ./...

# Generate mocks (requires mockgen)
mocks:
	mockgen -source=internal/repository/interfaces.go -destination=internal/repository/mocks/mocks.go -package=mocks

# All-in-one setup
setup: deps tidy build
	@echo "Setup complete!"

# ==========================================
# HELP
# ==========================================

help:
	@echo "Available targets:"
	@echo ""
	@echo "Local Development:"
	@echo "  make build        - Build local binary"
	@echo "  make run          - Build and run locally"
	@echo "  make dev          - Run with hot reload (requires air)"
	@echo ""
	@echo "AWS Lambda:"
	@echo "  make lambda-build - Build Lambda binary"
	@echo "  make sam-build    - Build with SAM"
	@echo "  make deploy       - Deploy to AWS (guided)"
	@echo "  make deploy-prod  - Deploy to production"
	@echo "  make sam-local    - Run Lambda locally"
	@echo "  make logs         - Tail Lambda logs"
	@echo "  make destroy      - Delete AWS stack"
	@echo ""
	@echo "Testing:"
	@echo "  make test         - Run tests"
	@echo "  make test-coverage- Run tests with coverage"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make deps         - Download dependencies"
	@echo "  make tidy         - Tidy go.mod"
	@echo "  make lint         - Run linter"
