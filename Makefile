.PHONY: all build build-linux run test clean docs deps package

# Variables
APP_NAME=flows-service
MAIN_PATH=cmd/server/main.go
BUILD_DIR=bin

all: build

# Compila para el sistema actual
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)

# Compila específicamente para Linux AMD64 (común para servidores)
build-linux:
	@echo "Building for Linux AMD64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP_NAME)-linux $(MAIN_PATH)

# Empaqueta el binario y el .env para despliegue
package: build-linux
	@echo "Packaging for deployment..."
	@mkdir -p dist
	@cp $(BUILD_DIR)/$(APP_NAME)-linux dist/$(APP_NAME)
	@cp .env dist/.env
	@tar -czf dist/deployment.tar.gz -C dist $(APP_NAME) .env
	@echo "Package created at dist/deployment.tar.gz"

run:
	@echo "Running $(APP_NAME)..."
	go run $(MAIN_PATH)

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) dist
	@go clean

swagger:
	@echo "Generating Swagger documentation..."
	swag init -g $(MAIN_PATH) -o docs

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
