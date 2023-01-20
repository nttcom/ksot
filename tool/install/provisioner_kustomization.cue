package install

import (
	"strings"
	appsv1 "k8s.io/api/apps/v1"
)

kustomizations: {
	provisioner: {
		// input
		var: {
			debug:           bool | *false
		}

		// path
		baseDir:       "../../provisioner"
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
					name:      "provisioner-controller-manager"
					namespace: "provisioner-system"
				}
				spec: template: spec: containers: [{
					name: "manager"
					if var.debug {
						args: ["--leader-elect"]
					}
				}]
			}
		}
	}
}
