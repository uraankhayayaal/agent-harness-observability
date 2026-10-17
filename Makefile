.PHONY: infra-up infra-down backend producer frontend build test env e2e-install e2e up down up-dev up-producer producer-docker

env:
	@test -f .env || cp .env.example .env

-include .env
export

infra-up: env
	docker compose up -d --wait

infra-down: env
	docker compose down -v

# Полный стек в Docker: infra + backend + frontend (nginx)
up: env
	docker compose up -d --build --wait

down: env
	docker compose down

# Dev-режим: infra + backend (go run, mount исходников) + frontend (vite, hot-reload)
up-dev: env
	docker compose --profile dev up -d --build --wait

# Демо-producer в Kafka (одноразовый генератор сообщений)
up-producer: env
	docker compose --profile producer up --build producer

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
