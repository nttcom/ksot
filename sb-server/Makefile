include ../.env

SB_IMG ?= ksot-sb-server:latest
SB_KUSTOMIZE_ROOT ?= overlays/test

.PHONY: docker-build
docker-build:
	docker build -t ${SB_IMG} .

.PHONY: kind-load
kind-load: docker-build
	kind load docker-image ${SB_IMG} --name ${KIND_NAME}

.PHONY: deploy
deploy:
	kubectl apply -k config/${SB_KUSTOMIZE_ROOT}

.PHONY: undeploy
undeploy:
	kubectl delete -k config/${SB_KUSTOMIZE_ROOT}