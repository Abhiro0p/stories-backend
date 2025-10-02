# Stories Backend Makefile
# Comprehensive build and development automation

# Variables
APP_NAME := stories-backend
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GO_VERSION := $(shell go version | awk '{print $$3}')

# Build flags
LDFLAGS := -w -s -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME) -X main.GoVersion=$(GO_VERSION)
BUILD_FLAGS := -ldflags "$(LDFLAGS)" -trimpath

# Docker
DOCKER_REGISTRY := ghcr.io
DOCKER_NAMESPACE := your-username
DOCKER_TAG := $(VERSION)
DOCKER_PLATFORMS := linux/amd64,linux/arm64

# Directories
BUILD_DIR := bin
SCRIPTS_DIR := scripts
DEPLOYMENTS_DIR := deployments
MIGRATIONS_DIR := migrations

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# Default target
.DEFAULT_GOAL := help

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\n🎯 $(YELLOW)Stories Backend$(NC) - Development Commands\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: info
info: ## Show project information
	@echo "$(BLUE)📊 Project Information$(NC)"
	@echo "  Name:        $(APP_NAME)"
	@echo "  Version:     $(VERSION)"
	@echo "  Commit:      $(COMMIT)"
	@echo "  Build Time:  $(BUILD_TIME)"
	@echo "  Go Version:  $(GO_VERSION)"

##@ Development

.PHONY: setup-dev
setup-dev: ## Setup development environment
	@echo "$(BLUE)🚀 Setting up development environment...$(NC)"
	@if [ ! -f .env ]; then cp .env.example .env; echo "$(GREEN)✅ Created .env file from template$(NC)"; fi
	@$(MAKE) install-tools
	@$(MAKE) services-up
	@$(MAKE) wait-for-services
	@$(MAKE) migrate-up
	@$(MAKE) seed
	@echo "$(GREEN)✅ Development environment ready!$(NC)"
	@echo "$(YELLOW)💡 Run 'make dev' to start the development server$(NC)"

.PHONY: dev
dev: ## Start development server with live reload
	@echo "$(BLUE)🔄 Starting development server...$(NC)"
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		@echo "$(YELLOW)⚠️ Air not found, running without live reload$(NC)"; \
		go run cmd/api/main.go; \
	fi

.PHONY: build
build: clean ## Build all binaries
	@echo "$(BLUE)🔨 Building binaries...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@echo "$(BLUE)  Building API server...$(NC)"
	@go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-api cmd/api/main.go
	@echo "$(BLUE)  Building worker...$(NC)"
	@go build $(BUILD_FLAGS) -o $(BUILD_DIR)/$(APP_NAME)-worker cmd/worker/main.go
	@echo "$(GREEN)✅ Build completed!$(NC)"
	@ls -la $(BUILD_DIR)/

.PHONY: run-api
run-api: build ## Run API server
	@echo "$(BLUE)🚀 Starting API server...$(NC)"
	@./$(BUILD_DIR)/$(APP_NAME)-api

.PHONY: run-worker
run-worker: build ## Run background worker
	@echo "$(BLUE)🔄 Starting worker...$(NC)"
	@./$(BUILD_DIR)/$(APP_NAME)-worker

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(BLUE)🧹 Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@go clean -cache
	@go clean -testcache
	@echo "$(GREEN)✅ Clean completed$(NC)"

##@ Dependencies

.PHONY: deps
deps: ## Download and tidy dependencies
	@echo "$(BLUE)📦 Managing dependencies...$(NC)"
	@go mod download
	@go mod tidy
	@go mod verify
	@echo "$(GREEN)✅ Dependencies updated$(NC)"

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "$(BLUE)⬆️ Updating dependencies...$(NC)"
	@go get -u ./...
	@go mod tidy
	@echo "$(GREEN)✅ Dependencies updated$(NC)"

.PHONY: install-tools
install-tools: ## Install development tools
	@echo "$(BLUE)🛠️ Installing development tools...$(NC)"
	@go install github.com/cosmtrek/air@latest
	@go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securecodewarrior/sast-scanner/cmd/gosec@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/golang/mock/mockgen@latest
	@echo "$(GREEN)✅ Development tools installed$(NC)"

##@ Testing

.PHONY: test
test: ## Run unit tests
	@echo "$(BLUE)🧪 Running unit tests...$(NC)"
	@go test -v -race -short ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "$(BLUE)📊 Running tests with coverage...$(NC)"
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Coverage report generated: coverage.html$(NC)"

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "$(BLUE)🔗 Running integration tests...$(NC)"
	@go test -v -race -tags=integration ./tests/integration/...

.PHONY: test-load
test-load: ## Run load tests
	@echo "$(BLUE)⚡ Running load tests...$(NC)"
	@if command -v k6 >/dev/null 2>&1; then \
		k6 run tests/load/k6-load-test.js; \
	else \
		echo "$(YELLOW)⚠️ k6 not found, skipping load tests$(NC)"; \
	fi

.PHONY: test-all
test-all: test test-integration test-load ## Run all tests

.PHONY: bench
bench: ## Run benchmarks
	@echo "$(BLUE)🏃 Running benchmarks...$(NC)"
	@go test -bench=. -benchmem ./...

##@ Code Quality

.PHONY: lint
lint: ## Run linter
	@echo "$(BLUE)🔍 Running linter...$(NC)"
	@golangci-lint run

.PHONY: lint-fix
lint-fix: ## Fix linting issues
	@echo "$(BLUE)🔧 Fixing linting issues...$(NC)"
	@golangci-lint run --fix

.PHONY: fmt
fmt: ## Format code
	@echo "$(BLUE)✨ Formatting code...$(NC)"
	@go fmt ./...
	@goimports -w .

.PHONY: vet
vet: ## Run go vet
	@echo "$(BLUE)🔍 Running go vet...$(NC)"
	@go vet ./...

.PHONY: security-scan
security-scan: ## Run security scan
	@echo "$(BLUE)🔒 Running security scan...$(NC)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "$(YELLOW)⚠️ gosec not found, skipping security scan$(NC)"; \
	fi

##@ Database

.PHONY: db-start
db-start: ## Start database services
	@echo "$(BLUE)🗄️ Starting database services...$(NC)"
	@docker-compose up -d postgres redis minio

.PHONY: db-stop
db-stop: ## Stop database services
	@echo "$(BLUE)⏹️ Stopping database services...$(NC)"
	@docker-compose stop postgres redis minio

.PHONY: db-reset
db-reset: ## Reset database (⚠️ DESTRUCTIVE)
	@echo "$(RED)⚠️ This will destroy all data. Are you sure? [y/N]$(NC)" && read ans && [ $${ans:-N} = y ]
	@docker-compose down -v
	@docker-compose up -d postgres redis minio
	@$(MAKE) wait-for-services
	@$(MAKE) migrate-up
	@echo "$(GREEN)✅ Database reset completed$(NC)"

.PHONY: migrate-up
migrate-up: ## Apply database migrations
	@echo "$(BLUE)⬆️ Applying database migrations...$(NC)"
	@$(SCRIPTS_DIR)/migrate.sh up

.PHONY: migrate-down
migrate-down: ## Rollback last migration
	@echo "$(YELLOW)⬇️ Rolling back last migration...$(NC)"
	@$(SCRIPTS_DIR)/migrate.sh down 1

.PHONY: migrate-reset
migrate-reset: ## Reset all migrations (⚠️ DESTRUCTIVE)
	@echo "$(RED)⚠️ This will destroy all data. Are you sure? [y/N]$(NC)" && read ans && [ $${ans:-N} = y ]
	@$(SCRIPTS_DIR)/migrate.sh drop

.PHONY: migrate-create
migrate-create: ## Create new migration (usage: make migrate-create NAME=add_users_table)
	@if [ -z "$(NAME)" ]; then echo "$(RED)❌ NAME is required. Usage: make migrate-create NAME=add_users_table$(NC)"; exit 1; fi
	@echo "$(BLUE)📝 Creating migration: $(NAME)$(NC)"
	@$(SCRIPTS_DIR)/migrate.sh create $(NAME)

.PHONY: seed
seed: ## Seed database with test data
	@echo "$(BLUE)🌱 Seeding database...$(NC)"
	@$(SCRIPTS_DIR)/seed.sh

##@ Services

.PHONY: services-up
services-up: ## Start all services (postgres, redis, minio)
	@echo "$(BLUE)🚀 Starting services...$(NC)"
	@docker-compose up -d postgres redis minio prometheus
	@echo "$(GREEN)✅ Services started$(NC)"

.PHONY: services-down
services-down: ## Stop all services
	@echo "$(BLUE)⏹️ Stopping services...$(NC)"
	@docker-compose down

.PHONY: services-logs
services-logs: ## View service logs
	@docker-compose logs -f

.PHONY: wait-for-services
wait-for-services: ## Wait for services to be ready
	@echo "$(BLUE)⏳ Waiting for services to be ready...$(NC)"
	@timeout=60; \
	while [ $$timeout -gt 0 ]; do \
		if docker-compose exec -T postgres pg_isready -U stories_user > /dev/null 2>&1 && \
		   docker-compose exec -T redis redis-cli ping > /dev/null 2>&1; then \
			echo "$(GREEN)✅ Services are ready$(NC)"; \
			break; \
		fi; \
		timeout=$$((timeout-1)); \
		sleep 1; \
	done; \
	if [ $$timeout -eq 0 ]; then \
		echo "$(RED)❌ Services failed to start$(NC)"; \
		exit 1; \
	fi

##@ Docker

.PHONY: docker-build
docker-build: ## Build Docker images
	@echo "$(BLUE)🐳 Building Docker images...$(NC)"
	@$(SCRIPTS_DIR)/build.sh docker

.PHONY: docker-build-prod
docker-build-prod: ## Build production Docker images
	@echo "$(BLUE)🐳 Building production Docker images...$(NC)"
	@docker build -f deployments/docker/Dockerfile.api -t $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(APP_NAME)-api:$(DOCKER_TAG) .
	@docker build -f deployments/docker/Dockerfile.worker -t $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(APP_NAME)-worker:$(DOCKER_TAG) .

.PHONY: docker-push
docker-push: ## Push Docker images
	@echo "$(BLUE)📤 Pushing Docker images...$(NC)"
	@docker push $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(APP_NAME)-api:$(DOCKER_TAG)
	@docker push $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$(APP_NAME)-worker:$(DOCKER_TAG)

.PHONY: docker-run
docker-run: ## Run with Docker Compose
	@echo "$(BLUE)🐳 Running with Docker Compose...$(NC)"
	@docker-compose up -d

.PHONY: docker-logs
docker-logs: ## View Docker logs
	@docker-compose logs -f

.PHONY: docker-clean
docker-clean: ## Clean Docker artifacts
	@echo "$(BLUE)🧹 Cleaning Docker artifacts...$(NC)"
	@docker system prune -f
	@docker volume prune -f

##@ Kubernetes

.PHONY: deploy-k8s
deploy-k8s: ## Deploy to Kubernetes
	@echo "$(BLUE)☸️ Deploying to Kubernetes...$(NC)"
	@$(SCRIPTS_DIR)/deploy.sh k8s all

.PHONY: undeploy-k8s
undeploy-k8s: ## Remove from Kubernetes
	@echo "$(BLUE)🗑️ Removing from Kubernetes...$(NC)"
	@kubectl delete namespace stories-backend --ignore-not-found

.PHONY: k8s-status
k8s-status: ## Show Kubernetes status
	@echo "$(BLUE)📊 Kubernetes status:$(NC)"
	@kubectl get all -n stories-backend

.PHONY: k8s-logs
k8s-logs: ## View Kubernetes logs
	@kubectl logs -f deployment/stories-backend-api -n stories-backend

.PHONY: k8s-scale
k8s-scale: ## Scale Kubernetes deployment (usage: make k8s-scale REPLICAS=5)
	@if [ -z "$(REPLICAS)" ]; then echo "$(RED)❌ REPLICAS is required. Usage: make k8s-scale REPLICAS=5$(NC)"; exit 1; fi
	@kubectl scale deployment stories-backend-api --replicas=$(REPLICAS) -n stories-backend

##@ Production

.PHONY: deploy-prod
deploy-prod: ## Deploy to production
	@echo "$(BLUE)🚀 Deploying to production...$(NC)"
	@$(SCRIPTS_DIR)/deploy.sh docker -e production

.PHONY: deploy-staging
deploy-staging: ## Deploy to staging
	@echo "$(BLUE)🎭 Deploying to staging...$(NC)"
	@$(SCRIPTS_DIR)/deploy.sh docker -e staging

.PHONY: health-check
health-check: ## Check application health
	@echo "$(BLUE)🏥 Checking application health...$(NC)"
	@curl -f http://localhost:8080/health || (echo "$(RED)❌ Health check failed$(NC)"; exit 1)
	@echo "$(GREEN)✅ Application is healthy$(NC)"

##@ Monitoring

.PHONY: metrics
metrics: ## View metrics
	@echo "$(BLUE)📊 Application metrics:$(NC)"
	@curl -s http://localhost:9090/metrics | head -20

.PHONY: logs
logs: ## View application logs
	@echo "$(BLUE)📋 Application logs:$(NC)"
	@docker-compose logs -f --tail=50 stories-api

.PHONY: monitor
monitor: ## Start monitoring stack
	@echo "$(BLUE)📊 Starting monitoring stack...$(NC)"
	@docker-compose -f deployments/docker/docker-compose.prod.yml up -d prometheus grafana

##@ Utilities

.PHONY: docs
docs: ## Generate documentation
	@echo "$(BLUE)📚 Generating documentation...$(NC)"
	@go doc -all . > GODOC.md

.PHONY: mocks
mocks: ## Generate mocks
	@echo "$(BLUE)🎭 Generating mocks...$(NC)"
	@go generate ./...

.PHONY: certificates
certificates: ## Generate development certificates
	@echo "$(BLUE)🔐 Generating development certificates...$(NC)"
	@mkdir -p certs
	@openssl req -x509 -nodes -newkey rsa:2048 -keyout certs/server.key -out certs/server.crt -days 365 -subj "/CN=localhost"

.PHONY: benchmark-api
benchmark-api: ## Benchmark API endpoints
	@echo "$(BLUE)🏃 Benchmarking API...$(NC)"
	@ab -n 1000 -c 10 http://localhost:8080/health

.PHONY: profile
profile: ## Start profiling server
	@echo "$(BLUE)📊 Starting profiling server...$(NC)"
	@echo "$(YELLOW)💡 Access profiler at http://localhost:6060/debug/pprof/$(NC)"
	@ENABLE_PPROF=true $(MAKE) run-api

##@ Environment

.PHONY: env-validate
env-validate: ## Validate environment configuration
	@echo "$(BLUE)✅ Validating environment...$(NC)"
	@if [ ! -f .env ]; then echo "$(RED)❌ .env file not found$(NC)"; exit 1; fi
	@echo "$(GREEN)✅ Environment validated$(NC)"

.PHONY: env-example
env-example: ## Update .env.example from current .env
	@echo "$(BLUE)📝 Updating .env.example...$(NC)"
	@cp .env .env.example
	@sed -i.bak 's/=.*/=/' .env.example && rm .env.example.bak
	@echo "$(GREEN)✅ .env.example updated$(NC)"

##@ Git

.PHONY: git-hooks
git-hooks: ## Install git hooks
	@echo "$(BLUE)🪝 Installing git hooks...$(NC)"
	@cp scripts/git-hooks/* .git/hooks/
	@chmod +x .git/hooks/*
	@echo "$(GREEN)✅ Git hooks installed$(NC)"

##@ Release

.PHONY: tag
tag: ## Create and push git tag (usage: make tag VERSION=v1.0.0)
	@if [ -z "$(VERSION)" ]; then echo "$(RED)❌ VERSION is required. Usage: make tag VERSION=v1.0.0$(NC)"; exit 1; fi
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@git push origin $(VERSION)
	@echo "$(GREEN)✅ Tagged and pushed $(VERSION)$(NC)"

.PHONY: release
release: ## Create release (requires VERSION)
	@if [ -z "$(VERSION)" ]; then echo "$(RED)❌ VERSION is required. Usage: make release VERSION=v1.0.0$(NC)"; exit 1; fi
	@$(MAKE) test-all
	@$(MAKE) docker-build-prod
	@$(MAKE) docker-push
	@$(MAKE) tag VERSION=$(VERSION)
	@echo "$(GREEN)✅ Release $(VERSION) completed$(NC)"

# Help target implementation
.PHONY: help-advanced
help-advanced: ## Show advanced help
	@echo "$(BLUE)🎯 Advanced Usage Examples:$(NC)"
	@echo ""
	@echo "$(YELLOW)Development Workflow:$(NC)"
	@echo "  make setup-dev      # One-time setup"
	@echo "  make dev           # Start development"
	@echo "  make test          # Run tests"
	@echo "  make lint          # Check code quality"
	@echo ""
	@echo "$(YELLOW)Database Operations:$(NC)"
	@echo "  make migrate-create NAME=add_users_table"
	@echo "  make migrate-up"
	@echo "  make seed"
	@echo ""
	@echo "$(YELLOW)Docker Deployment:$(NC)"
	@echo "  make docker-build-prod"
	@echo "  make deploy-prod"
	@echo ""
	@echo "$(YELLOW)Kubernetes Deployment:$(NC)"
	@echo "  make deploy-k8s"
	@echo "  make k8s-scale REPLICAS=5"
	@echo ""
	@echo "$(YELLOW)Monitoring:$(NC)"
	@echo "  make health-check"
	@echo "  make metrics"
	@echo "  make logs"

# Check if required tools are installed
check-tools:
	@command -v go >/dev/null 2>&1 || (echo "$(RED)❌ Go is required$(NC)"; exit 1)
	@command -v docker >/dev/null 2>&1 || (echo "$(RED)❌ Docker is required$(NC)"; exit 1)
	@command -v docker-compose >/dev/null 2>&1 || (echo "$(RED)❌ Docker Compose is required$(NC)"; exit 1)

# Include version information in help
version:
	@echo "$(APP_NAME) $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(GO_VERSION)"
