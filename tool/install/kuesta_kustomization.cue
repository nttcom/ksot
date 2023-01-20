package install

import (
	"strings"
	appsv1 "k8s.io/api/apps/v1"
)

kustomizations: {
	kuesta: {
		// input
		var: {
			configRepo:        string
			statusRepo:        string
			usePrivateRepo:    bool
			secretEnvFileName: string | *""
			secretKeyGitToken: string | *""
			debug:             bool | *false
		}

		// variables
		_secretName: "kuesta-secrets"

		// path
		baseDir:       "../.."
		configDir:     "config"
		kustomizeRoot: "overlays/getting-started"
		path:          strings.Join([baseDir, configDir, kustomizeRoot], "/")

		// kustomization
		kustomization: {
			apiVersion: "kustomize.config.k8s.io/v1beta1"
			kind:       "Kustomization"

			resources: ["../../default"]
			namespace: "kuesta-system"

			if var.usePrivateRepo {
				secretGenerator: [{
					envs: [var.secretEnvFileName]
					name: _secretName
				}]
			}

			patches: ["patch.yaml"]
		}

		// patches
		patches: {
			kuestaDeployment: appsv1.#Deployment & {
				apiVersion: "apps/v1"
				kind:       "Deployment"
				metadata: name: "kuesta-server"
				spec: template: spec: containers: [{
					name: "kuesta"
					env: [
						if var.debug {
							{
								name:  "KUESTA_DEVEL"
								value: "true"
							}
						},
						if var.debug {
							{
								name:  "KUESTA_VERBOSE"
								value: "2"
							}
						},
						{
							name:  "KUESTA_CONFIG_REPO_URL"
							value: var.configRepo
						},
						{
							name:  "KUESTA_STATUS_REPO_URL"
							value: var.statusRepo
						},
						if var.usePrivateRepo {
							{
								name: "KUESTA_GIT_TOKEN"
								valueFrom: secretKeyRef: {
									name: _secretName
									key:  var.secretKeyGitToken
								}
							}
						},
					]
				}]
			}

			kuestaAggregatorDeployment: appsv1.#Deployment & {
				apiVersion: "apps/v1"
				kind:       "Deployment"
				metadata: name: "kuesta-aggregator"
				spec: template: spec: containers: [{
					name: "kuesta"
					env: [
						if var.debug {
							{
								name:  "KUESTA_DEVEL"
								value: "true"
							}
						},
						if var.debug {
							{
								name:  "KUESTA_VERBOSE"
								value: "2"
							}
						},
						{
							name:  "KUESTA_STATUS_REPO_URL"
							value: var.statusRepo
						},
						if var.usePrivateRepo {
							{
								name: "KUESTA_GIT_TOKEN"
								valueFrom: secretKeyRef: {
									name: _secretName
									key:  var.secretKeyGitToken
								}
							}
						},
					]
				}]
			}
		}
	}
}
