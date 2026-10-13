NAME       = repaco
BIN_DIR    = bin
TARGET     = $(BIN_DIR)/$(NAME)
SRC        = ./cmd/repaco
GO         = go

.PHONY: all build run test clean lint help

all: build

build:
	@mkdir -p $(BIN_DIR)
	@echo "==> Building $(NAME)..."
	$(GO) build -o $(TARGET) $(SRC)

run:
	$(GO) run $(SRC) $(ARGS)

test:
	@echo "==> Running tests..."
	$(GO) test -v ./...

lint:
	@echo "==> Running go vet..."
	$(GO) vet ./...

clean:
	@echo "==> Cleaning..."
	rm -rf $(BIN_DIR) .repaco

help:
	@echo "Available commands:"
	@echo "  make build   - Compile repaco binary into bin/repaco"
	@echo "  make run     - Run repaco (use ARGS=\"...\" to pass arguments)"
	@echo "  make test    - Run Go unit tests"
	@echo "  make lint    - Run go vet"
	@echo "  make clean   - Remove binaries and build artifacts"
