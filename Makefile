APP_NAME=tsukuyo
OUT_DIR=./out
BIN_PATH=$(OUT_DIR)/$(APP_NAME)

.PHONY: all build run install clean test fmt lint vet help

all: build

build:
	@mkdir -p $(OUT_DIR)
	go build -o $(BIN_PATH) main.go

run: build
	$(BIN_PATH)

install: build
ifeq ($(OS),Windows_NT)
	@echo "Installing on Windows..."
	@mkdir -p $(USERPROFILE)\bin
	@cp $(BIN_PATH).exe $(USERPROFILE)\bin\$(APP_NAME).exe
	@echo "Add %USERPROFILE%\\bin to your PATH if not already present."
else
	@echo "Installing on Unix..."
	@mkdir -p $$HOME/bin
	cp $(BIN_PATH) $$HOME/bin/$(APP_NAME)
	@echo "Make sure $$HOME/bin is in your PATH."
endif

clean:
	rm -rf $(OUT_DIR)

fmt:
	go fmt ./...

lint:
	golangci-lint run || true

vet:
	go vet ./...

test:
	go test ./...

help:
	@echo "Common make targets:"
	@echo "  build   - Build the binary into $(OUT_DIR)"
	@echo "  run     - Build and run the CLI from $(OUT_DIR)"
	@echo "  install - Install the binary to your user bin directory (adds to PATH)"
	@echo "  clean   - Remove build artifacts in $(OUT_DIR)"
	@echo "  test    - Run tests"
	@echo "  fmt     - Format code"
	@echo "  lint    - Lint code (requires golangci-lint)"
	@echo "  vet     - Run go vet"
