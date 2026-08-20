.PHONY: frontend backend build dev

frontend:
	pnpm install
	pnpm build

backend:
	go build -o compose-dash .

build: frontend backend

dev:
	go run . &
	pnpm dev