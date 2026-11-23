.PHONY: all run-origin run-edge-india run-edge-europe test test-edge bench build docker-up docker-down fmt clean

all: build

run-origin:
	go run .

run-edge-india:
	go run ./edge --config ./edge/config/india.json

run-edge-europe:
	go run ./edge --config ./edge/config/europe.json

test:
	go test -v -race ./...

test-edge:
	go test -v -race ./edge

bench:
	go test -run='^$$' -bench='.' -benchmem ./edge

build:
	go build -v -o bin/origin .
	go build -v -o bin/edge ./edge

docker-up:
	docker compose up --build

docker-down:
	docker compose down

fmt:
	gofmt -w .

clean:
	rm -rf bin/ edge.exe
