.PHONY: run docs build

docs:
	swag init -g cmd/monitor/main.go

run: docs
	go run cmd/monitor/main.go

build: docs
	go build -o site-monitor cmd/monitor/main.go
