.PHONY: help build test test-integration test-coverage lint run clean docker-build docker-up docker-down migrate-up migrate-down generate-mocks

# Variables
BINARY_NAME=flowengine
BINARY_PATH=bin/$(BINARY_NAME)
MAIN_PATH=cmd/api/main.go
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GORUN=$(GOCMD) run

# Docker
DOCKER_COMPOSE=docker-compose

help: ## Mostrar esta ayuda
	@echo "Comandos disponibles:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Compilar el binario
	@echo "Compilando $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "Binario generado en $(BINARY_PATH)"

test: ## Ejecutar tests unitarios
	@echo "Ejecutando tests unitarios..."
	$(GOTEST) -v -race -short ./...

test-integration: ## Ejecutar tests de integración
	@echo "Ejecutando tests de integración..."
	$(GOTEST) -v -race -tags=integration ./test/...

test-coverage: ## Generar reporte de cobertura
	@echo "Generando reporte de cobertura..."
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Reporte generado en $(COVERAGE_HTML)"
	$(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total

lint: ## Ejecutar linter (golangci-lint)
	@echo "Ejecutando linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint no instalado. Instalar con: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin" && exit 1)
	golangci-lint run ./...

run: ## Ejecutar API server
	@echo "Iniciando API server..."
	$(GORUN) $(MAIN_PATH)

clean: ## Limpiar archivos generados
	@echo "Limpiando archivos generados..."
	@rm -rf bin/
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@echo "Limpieza completada"

docker-build: ## Construir imagen Docker
	@echo "Construyendo imagen Docker..."
	docker build -t $(BINARY_NAME):latest .

docker-up: ## Levantar stack completo con Docker Compose
	@echo "Levantando stack Docker Compose..."
	$(DOCKER_COMPOSE) up -d
	@echo "Stack levantado. Servicios disponibles:"
	$(DOCKER_COMPOSE) ps

docker-down: ## Detener stack Docker Compose
	@echo "Deteniendo stack Docker Compose..."
	$(DOCKER_COMPOSE) down

docker-logs: ## Ver logs de Docker Compose
	$(DOCKER_COMPOSE) logs -f

migrate-up: ## Aplicar migraciones de base de datos
	@echo "Aplicando migraciones..."
	@which migrate > /dev/null || (echo "golang-migrate no instalado. Instalar con: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" && exit 1)
	migrate -path migrations -database "postgresql://flowengine:flowengine@localhost:5432/flowengine?sslmode=disable" up

migrate-down: ## Revertir última migración
	@echo "Revirtiendo última migración..."
	migrate -path migrations -database "postgresql://flowengine:flowengine@localhost:5432/flowengine?sslmode=disable" down 1

migrate-create: ## Crear nueva migración (uso: make migrate-create NAME=nombre_migracion)
	@if [ -z "$(NAME)" ]; then echo "Error: NAME requerido. Uso: make migrate-create NAME=nombre_migracion"; exit 1; fi
	@echo "Creando migración $(NAME)..."
	migrate create -ext sql -dir migrations -seq $(NAME)

generate-mocks: ## Generar mocks con mockery
	@echo "Generando mocks..."
	@which mockery > /dev/null || (echo "mockery no instalado. Instalar con: go install github.com/vektra/mockery/v2@latest" && exit 1)
	./scripts/generate-mocks.sh

deps: ## Descargar dependencias
	@echo "Descargando dependencias..."
	$(GOMOD) download
	$(GOMOD) tidy

fmt: ## Formatear código
	@echo "Formateando código..."
	$(GOCMD) fmt ./...

vet: ## Ejecutar go vet
	@echo "Ejecutando go vet..."
	$(GOCMD) vet ./...

mod-verify: ## Verificar módulos
	@echo "Verificando módulos..."
	$(GOMOD) verify

all: clean deps fmt vet lint test build ## Ejecutar pipeline completo

# Desarrollo
dev: ## Ejecutar en modo desarrollo con hot-reload
	@which air > /dev/null || (echo "air no instalado. Instalar con: go install github.com/air-verse/air@latest" && exit 1)
	air

.DEFAULT_GOAL := help
