include ../.env

GITHUB_IMG=ksot-github-server:latest
GITHUB_KUSTOMIZE_ROOT ?= overlays/test

.PHONY: docker-build
docker-build: test
	docker build --no-cache -t ${GITHUB_IMG} . \
	--build-arg GITHUB_REPO_URL=${GITHUB_REPO_URL} \
	--build-arg GITHUB_USER_NAME=${GITHUB_USER_NAME} \
	--build-arg GITHUB_USER_MAIL=${GITHUB_USER_MAIL}

.PHONY: kind-load
kind-load: docker-build
	kind load docker-image ${GITHUB_IMG} --name ${KIND_NAME}

.PHONY: deploy
deploy:
	kubectl apply -k config/${GITHUB_KUSTOMIZE_ROOT}

.PHONY: undeploy
undeploy:
	kubectl delete -k config/${GITHUB_KUSTOMIZE_ROOT}

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test ./...