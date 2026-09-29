# Makefile for heat24 project

.PHONY: web
web:
	{ printf 'window.HEAT_DATA = '; go run ./cmd/heat24; } > web/data.js
	xdg-open web/index.html

.PHONY: test
test:
	go test ./...
