.ONESHELL:
.DEFAULT_GOAL := help

.PHONY: local-buf
local-buf:
	docker build -f buf.Dockerfile -t local-buf:latest .

.PHONY: generate
generate: local-buf ## generate protobuf outputs via Buf
	docker run --rm -v `pwd`:/workspace -w /workspace local-buf:latest generate

.PHONY: lint
lint: local-buf ## lint proto and code
	docker run --rm -v `pwd`:/workspace -w /workspace local-buf:latest lint
	golangci-lint run ./...

.PHONY: test
test:
	docker build -t technicallyjosh/protoc-gen-openapi .
	docker run --rm technicallyjosh/protoc-gen-openapi
