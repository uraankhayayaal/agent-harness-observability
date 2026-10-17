.PHONY: infra-up infra-down backend producer frontend build test

infra-up:
	docker compose up -d --wait

infra-down:
	docker compose down -v

backend:
	go -C backend run ./cmd/server

producer:
	go -C backend run ./cmd/producer

frontend:
	npm --prefix frontend run dev

build:
	go -C backend build ./...
	npm --prefix frontend run build

test:
	go -C backend test ./...
