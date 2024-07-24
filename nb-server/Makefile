include ../.env
NB_IMG ?= ksot-nb-server:latest
NB_KUSTOMIZE_ROOT ?= overlays/test

.PHONY: docker-build
docker-build:
	docker build -t ${NB_IMG} .

.PHONY: kind-load
kind-load: docker-build
	kind load docker-image ${NB_IMG} --name ${KIND_NAME}

.PHONY: deploy
deploy:
	kubectl apply -k config/${NB_KUSTOMIZE_ROOT}

.PHONY: undeploy
undeploy:
	kubectl delete -k config/${NB_KUSTOMIZE_ROOT}

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test ./...