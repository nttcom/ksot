package install

import (
	"encoding/json"
	"strings"
)

resources: {
	namespace: {
		var: namespace: string
		out: [#Namespace & {
			metadata: name: var.namespace
		}]
	}

	gitRepository: {
		var: {
			namespace:        string
			configRepo:       string
			usePrivateRepo:   bool
			gitRepoSecretRef: string | *""
		}

		let _splited = strings.Split(var.configRepo, "/")
		let _repoName = _splited[len(_splited)-1]

		out: [#GitRepository & {
			metadata: name:      _repoName
			metadata: namespace: var.namespace
			spec: {
				url: var.configRepo
				ref: branch: "main"
				gitImplementation: "go-git"
				interval:          "1m0s"
				timeout:           "60s"
				if var.usePrivateRepo {
					secretRef: name: var.gitRepoSecretRef
				}
			}
		}]
	}

	deviceOcDemo: {
		var: {
			name:       string
			namespace:  string
			configRepo: string
		}

		let _splited = strings.Split(var.configRepo, "/")
		let _repoName = _splited[len(_splited)-1]

		out: [
			#OcDemo & {
				metadata: name:      var.name
				metadata: namespace: var.namespace
				spec: {
					rolloutRef: _repoName
					address:    "gnmi-fake-\(var.name).\(var.namespace)"
					port:       9339
					tls: {
						secretName: "\(var.name)-cert"
						skipVerify: true
					}
				}
			},
			#Certificate & {
				metadata: name:      "\(var.name)-cert"
				metadata: namespace: var.namespace
				spec: {
					commonName: "\(var.name).example.com"
					issuerRef: {
						kind: "ClusterIssuer"
						name: "kuesta-ca-issuer"
					}
					secretName: "\(var.name)-cert"
				}
			},
		]
	}

	gnmiFake: {
		var: {
			name:      string
			namespace: string
			image:     string
		}

		let _name = "gnmi-fake-\(var.name)"

		out: [
			#Service & {
				metadata: {
					name:      _name
					namespace: var.namespace
				}
				spec: {
					ports: [{
						port:       9339
						protocol:   "TCP"
						targetPort: 9339
					}]
					selector: app: _name
					type: "ClusterIP"
				}
			},
			#PersistentVolumeClaim & {
				metadata: {
					name:      _name
					namespace: var.namespace
				}
				spec: {
					accessModes: [
						"ReadWriteOnce",
					]
					resources: requests: storage: "100Mi"
					storageClassName: "standard"
				}
			},
			#Deployment & {
				metadata: {
					labels: app: "gnmi-fake"
					name:      _name
					namespace: var.namespace
				}
				spec: {
					replicas: 1
					selector: matchLabels: app: _name
					template: {
						metadata: labels: app: _name
						spec: {
							containers: [{
								args: [
									"-bind_address",
									":9339",
									"-insecure",
									"-key",
									"/tmp/cert/tls.key",
									"-cert",
									"/tmp/cert/tls.crt",
									"-ca",
									"/tmp/cert/ca.crt",
								]
								image:           var.image
								imagePullPolicy: "IfNotPresent"
								name:            "gnmi-fake"
								ports: [{
									containerPort: 9339
								}]
								volumeMounts: [{
									mountPath: "/src/store"
									name:      "gnmi-fake-pvc"
								}, {
									mountPath: "/src/fixture"
									name:      "gnmi-fake-configmap"
								}, {
									mountPath: "/tmp/cert"
									name:      "cert"
									readOnly:  true
								}]
							}]
							volumes: [{
								name: "gnmi-fake-pvc"
								persistentVolumeClaim: claimName: _name
							}, {
								configMap: name: _name
								name: "gnmi-fake-configmap"
							}, {
								name: "cert"
								secret: {
									defaultMode: 420
									secretName:  "\(_name)-cert"
								}
							}]
						}
					}
				}
			},
			#Certificate & {
				metadata: name:      "\(_name)-cert"
				metadata: namespace: var.namespace
				spec: {
					commonName: "\(_name).example.com"
					issuerRef: {
						kind: "ClusterIssuer"
						name: "kuesta-ca-issuer"
					}
					secretName: "\(_name)-cert"
				}
			},
			#ConfigMap & {
				metadata: name:       _name
				metadata: namespace:  var.namespace
				data: "default.json": json.Marshal(_initialConfig)
			},
		]

		_initialConfig: {
			"openconfig-interfaces:interfaces": interface: [{
				name: "admin"
				config: name: "admin"
			}, {
				name: "Ethernet1"
				config: name: "Ethernet1"
			}, {
				name: "Ethernet2"
				config: name: "Ethernet2"
			}, {
				name: "Ethernet3"
				config: name: "Ethernet3"
			}]
		}
	}
}
