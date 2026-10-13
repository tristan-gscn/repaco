NAME        = repaco
BIN_DIR     = bin
TARGET      = $(BIN_DIR)/$(NAME)
SRC         = ./cmd/repaco
GO          = go
INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: all build install uninstall run test clean lint help

all: build

build:
	@mkdir -p $(BIN_DIR)
	@echo "==> Compilation de $(NAME)..."
	$(GO) build -o $(TARGET) $(SRC)

install: build
	@mkdir -p $(INSTALL_DIR)
	@echo "==> Installation de $(NAME) dans $(INSTALL_DIR)..."
	install -m 755 $(TARGET) $(INSTALL_DIR)/$(NAME)
	@echo "==> Terminé ! Vous pouvez désormais taper 'repaco' n'importe où dans votre terminal."

uninstall:
	@echo "==> Désinstallation de $(NAME)..."
	rm -f $(INSTALL_DIR)/$(NAME)
	@echo "==> Désinstallé."

run:
	$(GO) run $(SRC) $(ARGS)

test:
	@echo "==> Tests unitaires..."
	$(GO) test -v ./...

lint:
	@echo "==> go vet..."
	$(GO) vet ./...

clean:
	@echo "==> Nettoyage..."
	rm -rf $(BIN_DIR) .repaco

help:
	@echo "Commandes disponibles :"
	@echo "  make build     - Compile le binaire dans bin/repaco"
	@echo "  make install   - Installe le binaire dans $(INSTALL_DIR) pour accès global"
	@echo "  make uninstall - Supprime le binaire installé"
	@echo "  make test      - Lance les tests unitaires"
	@echo "  make clean     - Supprime les binaires compilés"
