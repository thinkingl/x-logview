.PHONY: all build build-server build-client dev clean

all: build

build: build-server build-client

build-server:
	cd . && go build -o bin/x-logview-server ./cmd/server

build-client:
	cd client && npm run build

dev:
	@echo "Starting development environment..."
	@echo "1. Start Go server: make dev-server"
	@echo "2. Start React dev server: make dev-client"

dev-server:
	go run ./cmd/server

dev-client:
	cd client && npm run dev

clean:
	rm -rf bin/
	rm -rf client/dist/
	rm -rf client/node_modules/

install-deps:
	cd client && npm install

lint:
	cd client && npm run lint

typecheck:
	cd client && npm run typecheck
