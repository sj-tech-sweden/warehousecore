# StorageCore Makefile

.PHONY: help build run test docker-build docker-push clean

help:
	@echo "StorageCore - Available commands:"
	@echo "  make build         - Build the Go binary"
	@echo "  make run           - Run the server locally"
	@echo "  make test          - Run tests"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-push   - Push Docker image to registry"
	@echo "  make clean         - Clean build artifacts"

build:
	@echo "Building StorageCore..."
	go build -o storagecore ./cmd/server

run:
	@echo "Running StorageCore..."
	go run ./cmd/server/main.go

test:
	@echo "Running tests..."
	go test -v ./...

docker-build:
	@echo "Building Docker image..."
	docker build -t nobentie/storagecore:1.0 .
	docker tag nobentie/storagecore:1.0 nobentie/storagecore:latest

docker-push:
	@echo "Pushing Docker image..."
	docker push nobentie/storagecore:1.0
	docker push nobentie/storagecore:latest

clean:
	@echo "Cleaning..."
	rm -f storagecore
	go clean
