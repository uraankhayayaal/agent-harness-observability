.PHONY: infra-up infra-down backend producer frontend build test env e2e-install e2e

env:
	@test -f .env || cp .env.example .env

-include .env
export

infra-up: env
	docker compose up -d --wait

infra-down: env
	docker compose down -v

backend: env
	go -C backend run ./cmd/server

producer: env
	go -C backend run ./cmd/producer

frontend: env
	npm --prefix frontend run dev

build:
	go -C backend build ./...
	npm --prefix frontend run build

test:
	go -C backend test ./...

e2e-install:
	cd tests && npm install && npx playwright install chromium

# Полный e2e-прогон: Kafka → backend → API → frontend (UI).
# Требует запущенных: make infra-up, make backend, make frontend
# (playwright сам запустит backend/frontend, если они не подняты)
e2e: env
	cd tests && npm test
