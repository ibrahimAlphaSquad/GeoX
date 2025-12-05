SHELL := /bin/bash

.PHONY: docs

# Install swag CLI and generate swagger docs into ./docs
docs:
	@echo "Installing swag CLI (if not present)..."
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Generating OpenAPI docs (swag init)..."
	@swag init -g main.go -o docs
	@echo "Done. Generated files are in ./docs"
