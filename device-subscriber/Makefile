IMG ?= device-subscriber:latest

.PHONY: docker-build
docker-build: test
	docker build -f ../build/Dockerfile.subscriber -t ${IMG} ..

.PHONY: docker-push
docker-push:
	docker push ${IMG}

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test ./... -coverprofile cover.out
