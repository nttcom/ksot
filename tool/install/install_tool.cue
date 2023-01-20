package install

import (
	"encoding/yaml"
	"list"
	"strings"
	"tool/file"
	"tool/exec"
	"tool/cli"
)

command: install: {

	args: {
		// TODO replace to nttcom
		imageRegistry: string | *"ghcr.io/nttcom/kuesta" @tag(imageRegistry)
		version:       string | *"latest"                   @tag(version)
		_debug:        string | *"false"                    @tag(debug)
		debug:         _debug != "false"
	}

	$usage: "cue install"
	$short: "Install kuesta to Kubernetes cluster with kubectl/kustomize."

	configRepo: cli.Ask & {
		prompt:   "Github repository for config:"
		response: string
	}

	statusRepo: cli.Ask & {
		$dep:     configRepo
		prompt:   "Github repository for status:"
		response: string
	}

	usePrivateRepo: cli.Ask & {
		$dep:     statusRepo
		prompt:   "Are these repositories private? (yes|no):"
		response: bool | *false
	}

	gitUsername: {
		if usePrivateRepo.response {
			cli.Ask & {
				prompt:   "Github username:"
				response: string
			}
		}
	}

	gitToken: {
		$dep: gitUsername.$done
		if usePrivateRepo.response {
			cli.Ask & {
				prompt:   "Github private access token:"
				response: string
			}
		}
	}

	wantEmulator: cli.Ask & {
		$dep: [usePrivateRepo.$done, gitToken.$done]
		prompt:   "Do you need sample driver and emulator for trial?: (yes|no)"
		response: bool
	}

	printInputs: cli.Print & {
		$dep: wantEmulator.$done
		text: strings.Join([
			"",
			"---",
			"Github Config Repository: \(configRepo.response)",
			"Github Status Repository: \(statusRepo.response)",
			"Use Private Repo: \(usePrivateRepo.response)",
			if usePrivateRepo.response {
				"Github Username: \(gitUsername.response)"
			},
			if usePrivateRepo.response {
				"Github Access Token: ***"
			},
			"",
			"Image Registry: \(args.imageRegistry)",
			"Version: \(args.version)",
			"Deploy sample driver and emulator: \(wantEmulator.response)",
			"---",
			"",
		], "\n")
	}

	confirm: cli.Ask & {
		$dep:     printInputs.$done
		prompt:   "Continue? (yes|no)"
		response: bool | *false
	}

	printConfirmResult: cli.Print & {
		$dep: confirm.$done
		if confirm.response {
			text: "\nApplying kustomize manifests...\n"
		}
		if !confirm.response {
			text: "\nCancelled.\n"
		}
	}

	apply: {
		if confirm.response {
			vendor: deployVendor & {
				$dep: confirm.$done
			}

			kuesta: deployKuesta & {
				$dep: vendor.$done
				var: {
					"configRepo":     configRepo.response
					"statusRepo":     statusRepo.response
					"usePrivateRepo": usePrivateRepo.response
					if usePrivateRepo.response {
						"gitToken": gitToken.response
					}
					image:   "\(args.imageRegistry)/kuesta"
					version: args.version
					debug:   args.debug
				}
			}
			provisioner: deployProvisioner & {
				$dep: kuesta.$done
				var: {
					image:   "\(args.imageRegistry)/provisioner"
					version: args.version
					debug:   args.debug
				}
			}
			deviceOperator: {
				$dep: provisioner.$done
				if wantEmulator.response {
					deployDeviceOperator & {
						var: {
							"statusRepo":    statusRepo.response
							image:           "\(args.imageRegistry)/device-operator"
							subscriberImage: "\(args.imageRegistry)/device-subscriber"
							version:         args.version
							debug:           args.debug
						}
					}
				}
			}
			gettingStartedResources: {
				$dep: deviceOperator.$done
				if wantEmulator.response {
					deployGettingStartedResources & {
						var: {
							"configRepo":     configRepo.response
							"usePrivateRepo": usePrivateRepo.response
							if usePrivateRepo.response {
								"gitUsername": gitUsername.response
								"gitToken":    gitToken.response
							}
							gnmiFakeImage: "\(args.imageRegistry)/gnmi-fake"
						}
					}
				}
			}
		}
	}
}

deployVendor: {
	_dep="$dep": _

	start: cli.Print & {
		$dep: _dep
		text: """
			\n\n
			==============================
			Deploy vendor dependencies\n
			"""
	}

	deployVendor: exec.Run & {
		$dep: start.$done
		dir:  "../.."
		cmd: ["bash", "-c", """
			kubectl apply -f ./config/vendor
			"""]
	}

	wait: exec.Run & {
		$dep: deployVendor.$done
		cmd: ["bash", "-c", """
				echo
				echo 'Waiting for cert-manager-webhook ready...'
				kubectl -n cert-manager wait deploy/cert-manager-webhook --for=condition=Available --timeout=120s
				echo
			"""]
	}

	deployPrivateCA: exec.Run & {
		$dep: wait.$done
		dir:  "../.."
		cmd: ["bash", "-c", """
			kubectl apply -f ./config/privateCA
			"""]
	}

	$done: deployPrivateCA.$done
}

deployKuesta: {
	_dep="$dep": _

	// inputs
	var: {
		configRepo:     string
		statusRepo:     string
		usePrivateRepo: bool
		gitToken:       string | *""
		image:          string
		version:        string | *"latest"
		debug:          bool
	}

	// private variables
	let _secretEnvFileName = ".env.secret"
	let _secretKeyGitToken = "gitToken"
	let _k = kustomizations.kuesta & {
		"var": {
			configRepo:        var.configRepo
			statusRepo:        var.statusRepo
			usePrivateRepo:    var.usePrivateRepo
			secretEnvFileName: _secretEnvFileName
			secretKeyGitToken: _secretKeyGitToken
			debug:             var.debug
		}
	}
	let _kustomizationFile = strings.Join([_k.path, "kustomization.yaml"], "/")
	let _patchFile = strings.Join([_k.path, "patch.yaml"], "/")
	let _secretEnvFile = strings.Join([_k.path, _secretEnvFileName], "/")

	// tasks
	start: cli.Print & {
		$dep: _dep
		text: """
			\n\n
			==============================
			Deploy kuesta\n
			"""
	}

	mkdir: file.MkdirAll & {
		$dep: start.$done
		path: _k.path
	}

	writeKustomization: file.Create & {
		$dep:     mkdir.$done
		filename: _kustomizationFile
		contents: yaml.Marshal(_k.kustomization)
	}

	writePatch: file.Create & {
		$dep:     mkdir.$done
		filename: _patchFile
		contents: yaml.MarshalStream([ for _, v in _k.patches {v}])
	}

	writeSecret: {
		$dep: writePatch.$done
		if var.usePrivateRepo {
			file.Create & {
				filename: _secretEnvFile
				contents: "\(_secretKeyGitToken)=\(var.gitToken)"
			}
		}
	}

	deploy: exec.Run & {
		$dep: [writeKustomization.$done, writePatch.$done, writeSecret.$done]
		dir: _k.baseDir
		cmd: ["bash", "-c", """
			export IMG='\(var.image):\(var.version)'
			export KUSTOMIZE_ROOT='\(_k.kustomizeRoot)'
			make deploy
			"""]
	}

	deleteSecret: {
		$dep: deploy.$done
		if var.usePrivateRepo {
			file.RemoveAll & {
				path: _secretEnvFile
			}
		}
	}

	$done: deploy.$done
}

deployProvisioner: {
	_dep="$dep": _

	// input
	var: {
		image:   string
		version: string | *"latest"
		debug:   bool
	}

	// private variables
	let _k = kustomizations.provisioner & {
		"var": {
			debug: var.debug
		}
	}
	let _kustomizationFile = strings.Join([_k.path, "kustomization.yaml"], "/")
	let _patchFile = strings.Join([_k.path, "patch.yaml"], "/")

	// tasks
	start: cli.Print & {
		$dep: _dep
		text: """
			\n\n
			==============================
			Deploy kuesta-provisioner\n
			"""
	}

	mkdir: file.MkdirAll & {
		$dep: start.$done
		path: _k.path
	}

	writeKustomization: file.Create & {
		$dep:     mkdir.$done
		filename: _kustomizationFile
		contents: yaml.Marshal(_k.kustomization)
	}

	writePatch: file.Create & {
		$dep:     mkdir.$done
		filename: _patchFile
		contents: yaml.MarshalStream([ for _, v in _k.patches {v}])
	}

	installCRD: exec.Run & {
		$dep: start.$done
		dir:  _k.baseDir
		cmd: ["bash", "-c", "make install"]
	}

	deploy: exec.Run & {
		$dep: installCRD.$done
		dir:  _k.baseDir
		cmd: ["bash", "-c", """
			export IMG='\(var.image):\(var.version)'
			export KUSTOMIZE_ROOT='\(_k.kustomizeRoot)'
			make deploy
			"""]
	}

	$done: deploy.$done
}

deployDeviceOperator: {
	_dep="$dep": _

	// inputs
	var: {
		statusRepo:      string
		image:           string
		subscriberImage: string
		version:         string | *"latest"
		debug:           bool
	}

	// private variables
	let _k = kustomizations.deviceoperator & {
		"var": {
			statusRepo:      var.statusRepo
			version:         var.version
			subscriberImage: var.subscriberImage
			debug:           var.debug
		}
	}
	let _kustomizationFile = strings.Join([_k.path, "kustomization.yaml"], "/")
	let _patchFile = strings.Join([_k.path, "patch.yaml"], "/")

	// tasks
	start: cli.Print & {
		$dep: _dep
		text: """
			\n\n
			==============================
			Deploy device-operator\n
			"""
	}

	mkdir: file.MkdirAll & {
		$dep: start.$done
		path: _k.path
	}

	writeKustomization: file.Create & {
		$dep:     mkdir.$done
		filename: _kustomizationFile
		contents: yaml.Marshal(_k.kustomization)
	}

	writePatch: file.Create & {
		$dep:     mkdir.$done
		filename: _patchFile
		contents: yaml.MarshalStream([ for _, v in _k.patches {v}])
	}

	installCRD: exec.Run & {
		$dep: [writeKustomization.$done, writePatch.$done]
		dir: _k.baseDir
		cmd: ["bash", "-c", "make install"]
	}

	deploy: exec.Run & {
		$dep: installCRD.$done
		dir:  _k.baseDir
		cmd: ["bash", "-c", """
			export IMG='\(var.image):\(var.version)'
			export KUSTOMIZE_ROOT='\(_k.kustomizeRoot)'
			make deploy
			"""]
	}

	$done: deploy.$done
}

deployGettingStartedResources: {
	_dep="$dep": _

	// inputs
	var: {
		namespace:       string | *"kuesta-getting-started"
		configRepo:      string
		usePrivateRepo:  bool
		gitUsername:     string | *""
		gitToken:        string | *""
		gnmiFakeImage:   string
		gnmiFakeVersion: string | *"latest"
	}

	// private variables
	let _manifestFile = "getting-started.yaml"
	let _splited = strings.Split(var.configRepo, "/")
	let _repoName = _splited[len(_splited)-1]

	let _gitTokenSecretName = "\(_repoName)-secret"
	let _resources = [
		resources.namespace & {
			"var": namespace: var.namespace
		},
		resources.gitRepository & {
			"var": {
				namespace:      var.namespace
				configRepo:     var.configRepo
				usePrivateRepo: var.usePrivateRepo
				if var.usePrivateRepo {
					gitRepoSecretRef: _gitTokenSecretName
				}
			}
		},
		resources.deviceOcDemo & {
			"var": {
				name:       "oc01"
				namespace:  var.namespace
				configRepo: var.configRepo
			}
		},
		resources.deviceOcDemo & {
			"var": {
				name:       "oc02"
				namespace:  var.namespace
				configRepo: var.configRepo
			}
		},
		resources.gnmiFake & {
			"var": {
				name:      "oc01"
				namespace: var.namespace
				image:     "\(var.gnmiFakeImage):\(var.gnmiFakeVersion)"
			}
		},
		resources.gnmiFake & {
			"var": {
				name:      "oc02"
				namespace: var.namespace
				image:     "\(var.gnmiFakeImage):\(var.gnmiFakeVersion)"
			}
		},
	]

	// tasks
	start: cli.Print & {
		$dep: _dep
		text: """
			\n\n
			==============================
			Deploy getting-started resources\n
			"""
	}

	createSecret: {
		$dep: start.$done
		if var.usePrivateRepo {
			exec.Run & {
				cmd: ["bash", "-c", """
				kubectl create ns \(var.namespace)
				kubectl create secret generic \(_gitTokenSecretName) -n \(var.namespace) \\
				--from-literal=username=\(var.gitUsername) --from-literal=password=\(var.gitToken)
				"""]
			}
		}
	}

	writeManifest: file.Create & {
		$dep:     start.$done
		filename: _manifestFile
		contents: yaml.MarshalStream(list.Concat([ for _, v in _resources {v.out}]))
	}

	deploy: exec.Run & {
		$dep: [writeManifest.$done, createSecret.$done]
		cmd: ["bash", "-c", "kubectl apply -f \(_manifestFile)"]
	}

	deleteManifest: file.RemoveAll & {
		$dep: deploy.$done
		path: _manifestFile
	}

	$done: deploy.$done
}
