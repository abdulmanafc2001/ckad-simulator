.PHONY: run run-backend run-frontend stop build install clean help

# Default ports (override with e.g. `make run PORT=9090 VITE_PORT=5174`)
PORT ?= 8080
VITE_PORT ?= 5173

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

install: ## Install frontend dependencies
	cd frontend && npm install

build: ## Build backend binary and frontend production bundle
	cd backend && go build -o /tmp/ckad-server ./cmd/server
	cd frontend && npm run build

run: ## Run backend and frontend together (Ctrl+C stops both)
	@echo "Starting CKAD Simulator..."
	@echo "  Backend  -> http://localhost:$(PORT)  (PORT=$(PORT))"
	@echo "  Frontend -> http://localhost:$(VITE_PORT)  (VITE_PORT=$(VITE_PORT))"
	@echo ""
	@trap 'kill 0' INT TERM; \
	PORT=$(PORT) go run ./backend/cmd/server & \
	BACKEND_PID=$$!; \
	cd frontend && VITE_PORT=$(VITE_PORT) npm run dev -- --port $(VITE_PORT) --host 0.0.0.0 & \
	FRONTEND_PID=$$!; \
	wait $$BACKEND_PID $$FRONTEND_PID

run-backend: ## Run only the Go backend (http://localhost:8080)
	PORT=$(PORT) go run ./backend/cmd/server

run-frontend: ## Run only the Vite frontend (http://localhost:5173)
	cd frontend && npm run dev -- --port $(VITE_PORT) --host 0.0.0.0

stop: ## Stop running backend and frontend (kills processes on PORT and VITE_PORT)
	@echo "Stopping CKAD Simulator (ports $(PORT) and $(VITE_PORT))..."
	@lsof -ti :$(PORT) 2>/dev/null | xargs -r kill -9 2>/dev/null && echo "  Stopped backend  (port $(PORT))" || echo "  Backend not running (port $(PORT))"
	@lsof -ti :$(VITE_PORT) 2>/dev/null | xargs -r kill -9 2>/dev/null && echo "  Stopped frontend (port $(VITE_PORT))" || echo "  Frontend not running (port $(VITE_PORT))"
	@pkill -f "go run ./backend/cmd/server" 2>/dev/null && echo "  Stopped Go backend process" || true
	@pkill -f "vite.*--port $(VITE_PORT)" 2>/dev/null && echo "  Stopped Vite process" || true
	@echo "Done."

clean: ## Remove build artifacts
	rm -f /tmp/ckad-server
	rm -rf frontend/dist frontend/node_modules/.vite

lint: ## Vet Go code and lint frontend
	cd backend && go vet ./...
	cd frontend && npm run lint
