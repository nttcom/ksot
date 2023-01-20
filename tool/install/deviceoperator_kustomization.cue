package install

import (
	"strings"
	appsv1 "k8s.io/api/apps/v1"
)

kustomizations: {
	deviceoperator: {
		// input
		var: {
			statusRepo:      string
			version:         string
			subscriberImage: string
			debug:           bool | *false
		}

		// path
		baseDir:       "../../device-operator"
		configDir:     "config"
		kustomizeRoot: "overlays/getting-started"
		path:          strings.Join([baseDir, configDir, kustomizeRoot], "/")

		// kustomization
		kustomization: {
			apiVersion: "kustomize.config.k8s.io/v1beta1"
			kind:       "Kustomization"

			resources: ["../../default"]
			patches: ["patch.yaml"]
		}

		// patches
		patches: {
			deviceOperatorDeployment: appsv1.#Deployment & {
				apiVersion: "apps/v1"
				kind:       "Deployment"
				metadata: {
					name:      "device-operator-controller-manager"
					namespace: "device-operator-system"
				}
				spec: template: spec: containers: [{
					name: "manager"
					if var.debug {
						args: ["--leader-elect"]
					}
					env: [{
						name:  "KUESTA_AGGREGATOR_URL"
						value: "https://kuesta-aggregator.kuesta-system:8000"
					}, {
						name:  "KUESTA_SUBSCRIBER_IMAGE"
						value: var.subscriberImage
					}, {
						name:  "KUESTA_SUBSCRIBER_IMAGE_VERSION"
						value: var.version
					}]
				}]
			}
		}
	}
}
