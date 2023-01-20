IMG ?= kuesta:latest
KUSTOMIZE_ROOT ?= overlays/example

.PHONY: docker-build
docker-build: test
	docker build -f build/Dockerfile.kuesta -t ${IMG} .

.PHONY: docker-push
docker-push:
	docker push ${IMG}

.PHONY: deploy-preview
deploy-preview: kustomize
	cd config/bases/kuesta && $(KUSTOMIZE) edit set image kuesta=${IMG}
	kubectl kustomize config/${KUSTOMIZE_ROOT}

.PHONY: deploy
deploy: kustomize
	cd config/bases/kuesta && $(KUSTOMIZE) edit set image kuesta=${IMG}
	kubectl apply -k config/${KUSTOMIZE_ROOT}

.PHONY: undeploy
undeploy: kustomize
	cd config/bases/kuesta && $(KUSTOMIZE) edit set image kuesta=${IMG}
	kubectl delete -k config/${KUSTOMIZE_ROOT}

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

.PHONY: test-all
test-all:
	@echo "*** kuesta ***"
	make test
	@echo
	@echo "*** kuesta-provisioner ***"
	cd provisioner && make test
	@echo
	@echo "*** device-operator ***"
	cd device-operator && make test
	@echo
	@echo "*** device-subscriber ***"
	cd device-subscriber && make test

.PHONY: lint-all
lint-all:
	@echo "*** kuesta ***"
	golangci-lint run -v
	@echo
	@echo "*** kuesta-provisioner ***"
	cd provisioner && golangci-lint run -v
	@echo
	@echo "*** device-operator ***"
	cd device-operator && golangci-lint run -v
	@echo
	@echo "*** device-subscriber ***"
	cd device-subscriber && golangci-lint run -v

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"

## Tool Versions
KUSTOMIZE_VERSION ?= v4.5.7

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }
