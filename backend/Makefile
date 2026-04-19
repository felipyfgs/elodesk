APP_NAME = backend
BUILD_DIR = bin

.PHONY: dev build test lint docs tidy clean seed

dev: ## Roda o backend localmente (go run)
	go run cmd/backend/main.go

build: ## Compila o binário Go
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(BUILD_DIR)/$(APP_NAME) cmd/backend/main.go

test: ## Roda testes com race detector
	go test -race ./...

lint: ## Roda linter
	golangci-lint run ./...

docs: ## Gera documentação Swagger
	swag init -g main.go -o docs --parseInternal --useStructName \
		-d cmd/backend,internal

tidy: ## go mod tidy
	go mod tidy

clean: ## Remove artefatos de build
	@rm -rf $(BUILD_DIR)

seed: ## Executa seed de dados iniciais
	go run cmd/seed/main.go 2>/dev/null || echo "seed command not implemented yet"

install-tools: ## Instala ferramentas de dev
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin
	go install github.com/swaggo/swag/cmd/swag@latest
