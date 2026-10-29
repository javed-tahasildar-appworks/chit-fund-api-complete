.PHONY: help build run test clean docker-up docker-down migrate-up migrate-down

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building application..."
	@go build -o bin/api cmd/api/main.go

run: ## Run the application
	@echo "Running application..."
	@go run cmd/api/main.go

test: ## Run tests
	@echo "Running tests..."
	@go test -v -cover ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@go clean

docker-up: ## Start Docker containers
	@echo "Starting Docker containers..."
	@docker-compose up -d

docker-down: ## Stop Docker containers
	@echo "Stopping Docker containers..."
	@docker-compose down

docker-logs: ## Show Docker logs
	@docker-compose logs -f api

migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	@migrate -path migrations -database "postgresql://chitfund:chitfund123@localhost:5432/chitfund?sslmode=disable" up

migrate-down: ## Run database migrations down
	@echo "Running migrations down..."
	@migrate -path migrations -database "postgresql://chitfund:chitfund123@localhost:5432/chitfund?sslmode=disable" down

migrate-create: ## Create new migration (usage: make migrate-create name=migration_name)
	@echo "Creating migration..."
	@migrate create -ext sql -dir migrations -seq $(name)

swagger: ## Generate Swagger documentation
	@echo "Generating Swagger docs..."
	@swag init -g cmd/api/main.go -o docs

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

dev: ## Run in development mode with hot reload
	@echo "Running in development mode..."
	@air

install-tools: ## Install development tools
	@echo "Installing tools..."
	@go install github.com/cosmtrek/air@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest