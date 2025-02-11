.PHONY: build run docker-build docker-up

build:
	go build -o bot .

run:
	./bot

docker-build:
	docker compose build

docker-up: docker-build
	docker compose up