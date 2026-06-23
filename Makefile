.PHONY: dev run build test lint fmt migrate-up migrate-down migrate-status migrate-create docker-up docker-down seed seed-demo clean


run:
	go run ./cmd/api


migrate-up:
	go run ./cmd/api migrate up

migrate-down:
	go run ./cmd/api migrate down 1

migrate-status:
	go run ./cmd/api migrate status