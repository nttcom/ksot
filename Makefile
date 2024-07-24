.PHONY: kind-load
kind-load:
	cd nb-server && $(MAKE) kind-load
	cd sb-server && $(MAKE) kind-load
	cd github-server && $(MAKE) kind-load

.PHONY: deploy
deploy:
	cd nb-server && $(MAKE) deploy
	cd sb-server && $(MAKE) deploy
	cd github-server && $(MAKE) deploy

.PHONY: undeploy
undeploy:
	cd nb-server && $(MAKE) undeploy
	cd sb-server && $(MAKE) undeploy
	cd github-server && $(MAKE) undeploy

.PHONY: getting-started
getting-started: kind-load deploy
	kubectl get pods