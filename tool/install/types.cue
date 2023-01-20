package install

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

#Namespace: corev1.#Namespace & {
	apiVersion: "v1"
	kind:       "Namespace"
}

#Service: corev1.#Service & {
	apiVersion: "v1"
	kind:       "Service"
}

#ConfigMap: corev1.#ConfigMap & {
	apiVersion: "v1"
	kind:       "ConfigMap"
}

#PersistentVolumeClaim: corev1.#PersistentVolumeClaim & {
	apiVersion: "v1"
	kind:       "PersistentVolumeClaim"
}

#Deployment: appsv1.#Deployment & {
	apiVersion: "apps/v1"
	kind:       "Deployment"
}

// TODO cue-get type def and conjunct
#GitRepository: {
	apiVersion: "source.toolkit.fluxcd.io/v1beta2"
	kind:       "GitRepository"
	...
}

#Certificate: {
	apiVersion: "cert-manager.io/v1"
	kind:       "Certificate"
	...
}

#OcDemo: {
	apiVersion: "kuesta.hrk091.dev/v1alpha1"
	kind:       "OcDemo"
	...
}
