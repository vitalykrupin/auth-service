SVC=auth-service

.PHONY: build run docker-build up down migrate-up migrate-down

build:
	go build -o $(SVC) ./cmd/auth

run:
	HTTP_ADDR=0.0.0.0:8081 go run cmd/auth/main.go

docker-build:
	docker build -t $(SVC):local .

up:
	docker compose up --build -d

down:
	docker compose down


