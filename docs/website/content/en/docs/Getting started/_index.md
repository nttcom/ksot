---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 1
description: >
  Let's move K-SOT!
---

In this tutorial, you will build K-SOT and check its operation by following the steps below.

1. Create a local Kubernetes cluster
2. Create a Github repository for verification
3. Install K-SOT and configure various files
4. Start up the device emulator
5. Deploy K-SOT
6. Operation check

## Advance preparation

1. Install [golang (v1.21.0+))](https://go.dev/doc/install). You will need it to write the device configuration derivation logic.
2. Install [docker (v20.10.24+)](https://docs.docker.com/engine/install). Required to run KIND.
3. Install [kind (v0.12.0+)](https://kind.sigs.k8s.io/docs/user/quick-start/). Used to build local Kubernetes clusters.
4. Install [kubectl (v1.25.4+)](https://kubernetes.io/docs/tasks/tools/). Used to execute commands against a Kubernetes cluster.

## 1. Create a local Kubernetes cluster

Run the following command to start a local Kubernetes cluster.

```bash
kind create cluster --name ksot-started
```

When execution is complete, confirm that the Kubernetes cluster has been successfully created by running the following command.

```bash
kubectl cluster-info
```

If you see output similar to the following, the local cluster is working properly.

```bash
Kubernetes control plane is running at https://127.0.0.1:52096
CoreDNS is running at https://127.0.0.1:52096/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
```


## 2. Create a Github repository for verification

In order for K-SOT to manage the configuration of network devices using Github, a GitHub repository must be created.
Follow the steps below to create the GitHub repository.

1. Log in to GitHub and create a new repository named `ksot-example`. You can choose to make the repository public or private as you like.

2. Create the following two folders at the root of the created repository:
   - `Devices`
   - `Services`


## 3. Install K-SOT and configure various files
### Clone the K-SOT repository and initialize submodules.
Repository clone
```bash
git clone https://github.com/nttcom/ksot.git
```
Submodule initialization
```bash
git submodule update --init --recursive
```

### Configuration of env file
Place the following env file in the K-SOT repository cloned above.

/.env
```bash
KIND_NAME=ksot-started
GITHUB_REPO_URL=https://{GithubUserName}:{GithubUserToken}@github.com/{GithubUserName}/ksot-example
GITHUB_USER_NAME={GithubUserName}
GITHUB_USER_MAIL={GithubMailAddress}
```

/github-server/config/overlays/test/.env
```bash
GITHUB_REPO_URL=https://{GithubUserName}:{GithubUserToken}@github.com/{GithubUserName}/ksot-example
```

### Configuration of the YANG
#### YANG settings for equipment
Acquire all YANG in [YANG Folder]( https://github.com/opennetworkinglab/ODTN-emulator/tree/master/emulator-oc-cassini/yang/openconfig-odtn) of [ODTN-emulator](https://github.com/opennetworkinglab/ODTN-emulator), which is the device to be managed in this article. Create a folder (name: cassini) under ```nb-server/pkg/tf/yang/devices`` and place the YANG files you just acquired.
#### Setting up YANG for services
Create a folder (name: transceivers) under ``nb-server/pkg/tf/yang/devices`` and place the yang files with the following contents in it.

```yang
module transceivers {
    namespace "tf-ksot/transceivers";
	prefix ts;
	grouping transceivers-top {
		list transceivers {
			key "name";
			leaf name {
				type string;
			}
			leaf-list  up {
				type string;
			}
			leaf-list down {
				type string;
			}
		}
	}
    uses transceivers-top;
}
```

### Configuration of equipment configuration derivation package
```nb-server/pkg/tf/transceivers.go```with the following contents.
```go
package tf

import (
	"encoding/json"
	"fmt"

	"github.com/nttcom-ic/ksot/nb-server/pkg/model/pathmap"
)

type Transceiver struct {
	Name string   `json:"name"`
	Nos  string   `json:"nos"`
	Up   []string `json:"up"`
	Down []string `json:"down"`
}
type Transceivers map[string][]Transceiver

func TfTransceivers(ts interface{}) (map[string]pathmap.PathMapInterface, error) {
	fmt.Println("hello")
	mapByte, err := json.Marshal(ts)
	if err != nil {
		return nil, err
	}
	var transceivers Transceivers
	if err := json.Unmarshal(mapByte, &transceivers); err != nil {
		fmt.Println(err)
		return nil, err
	}
	results := make(map[string]pathmap.PathMapInterface)
	fmt.Println("tflogic: ", transceivers)
	for _, devices := range transceivers {
		for _, device := range devices {
			result, err := pathmap.NewPathMap(map[string]interface{}{})
			if err != nil {
				return nil, err
			}
			for _, v := range device.Up {
				path := fmt.Sprintf("/openconfig-platform:components/component[name=%v]/openconfig-platform-transceiver:transceiver/config/enabled", v)
				err = result.SetValue(path, true, map[string]string{})
				if err != nil {
					return nil, err
				}
			}
			for _, v := range device.Down {
				path := fmt.Sprintf("/openconfig-platform:components/component[name=%v]/openconfig-platform-transceiver:transceiver/config/enabled", v)
				err = result.SetValue(path, false, map[string]string{})
				if err != nil {
					return nil, err
				}
			}
			results[device.Name] = result
		}
	}
	fmt.Println("check: ", results)
	return results, nil
}
```
```nb-server/pkg/tf/tf.go``` to the following content.
```go
package tf

import (
	"github.com/nttcom-ic/ksot/nb-server/pkg/model"
)

var TfLogic = model.NewPathMapLogic()

func init() {
	// User needs to create a function to generate a pathmap and add it to the MAP.
	// Example.
	// TfLogic["serviceName"] = serviceName
	TfLogic["transceivers"] = TfTransceivers
}
```

## 4. Start up the device emulator
Add an env file with emulator connection information.
sb-server/config/overlays/test/.env
```bash
cassini=root
```

Launch [OTDN-emulator](https://github.com/opennetworkinglab/ODTN-emulator) with the following command.
```bash
docker pull onosproject/oc-cassini:0.21
docker run -it -d --name odtn-emulator_openconfig_cassini_1 -p 11002:830 onosproject/oc-cassini:0.21
```

Make sure the container is up and running as follows.
```bash
% docker ps
CONTAINER ID  IMAGE             COMMAND          CREATED    STATUS    PORTS                   NAMES
52da5c9f1c79  onosproject/oc-cassini:0.21  "sh /root/script/pus…"  4 hours ago  Up 4 hours  22/tcp, 8080/tcp, 0.0.0.0:11002->830/tcp  odtn-emulator_openconfig_cassini_1
```

Create ```sb-server/connect.json``` in the K-SOT repository to add connection information from the controller to [OTDN-emulator](https://github.com/opennetworkinglab/ODTN-emulator) and save it with the following contents.
```json
{
    "cassini": {
        "ip": "host.docker.internal",
        "port": 11002,
        "username": "root",
        "hostKeyVerify": false,
        "if": "netconf"
    }
}
```

## 5. Deploy K-SOT
Execute the following command in the root of the K-SOT repository.
```bash
make getting-started
```
After execution, check the pods running with ```kubectl get pod`` and you will see the following three pods have been created.

```bash
NAMESPACE            NAME                                                 READY   STATUS    RESTARTS   AGE
default              ksot-github-deployment-766b6ff499-72zkr              1/1     Running   0          24s
default              ksot-nb-deployment-967757888-dqzw7                   1/1     Running   0          24s
default              ksot-sb-deployment-6d89c6f85c-zksl5                  1/1     Running   0          24s
...
```

## 6. Operation check
### Implementing port-forwarding
First, port-forwarding should be performed on the k8s service that will be the Northbound.

```bash
kubectl port-forward svc/ksot-nb-service 8080:8080
```

### Get equipment information.
Perform ```PUT: http://localhost:8080/sync/devices``` to reflect the current device configuration in the Github repository.
It is successful if the following files are created in ``Devices/cassini`` of the Github repository for verification.
- actual.json: configuration obtained from the actual device
- ref.json: parameters set from each service (default value is empty JSON)
- set.json: settings submitted by the controller (initial values are the same as those obtained from the actual device)


### Service Creation
```POST: http://localhost:8080/services``` is executed with the following JSON as the body, the service specified in the body is created, and the device settings are changed.
```JSON
{
    "transceivers": {
        "transceivers:transceivers":  [
            {
                "name": "cassini",
                "up": [],
                "down": [
                    "oe1",
                    "oe2",
                ]
            }
        ]
    }
}
```
Make sure the following files are created in ``Services/transceivers`` in the verification Github repository.
- input.json: the value of the submitted service
- output.json: xpath and corresponding values for the device whose configuration you have changed

### Get equipment information (after service creation)
Perform ```PUT: http://localhost:8080/sync/devices``` to reflect the current device configuration in the Github repository.
The ``Devices/cassini/actual.json`` in the Github repository for verification is updated as follows, and the transceivers in the specified location of [OTDN-emulator](https://github.com/opennetworkinglab/ODTN-emulator) are down. The ``Devices/cassini/actual.json`` is updated as follows.
```diff
{
...
				"name": "oe1",
				"config": {
					"name": "oe1"
				},
				"state": {
					"type": "openconfig-platform-types:TRANSCEIVER",
					"empty": false
				},
				"openconfig-platform-transceiver:transceiver": {
					"config": {
						- "enabled": true,
						+  "enabled": false,
						"form-factor-preconf": "openconfig-transport-types:CFP2_ACO"
					}
				}
...
				"name": "oe2",
				"config": {
					"name": "oe2"
				},
				"state": {
					"type": "openconfig-platform-types:TRANSCEIVER",
					"empty": false
				},
				"openconfig-platform-transceiver:transceiver": {
					"config": {
						- "enabled": true,
						+  "enabled": false
						"form-factor-preconf": "openconfig-transport-types:CFP2_ACO"
					}
				}
...
}
```

### Updating services
Execute ``PUT: http://localhost:8080/services`` with the following JSON as the body, update the service specified in the body, and change the device settings.
In this case, we will update the oe1 transceiver.
```JSON
{
    "transceivers": {
        "transceivers:transceivers":  [
            {
                "name": "cassini",
                "up": [
					"oe1",
				],
                "down": [
                    "oe2",
                ]
            }
        ]
    }
}
```

### Get equipment information (after service update)
Perform ```PUT: http://localhost:8080/sync/devices``` to reflect the current equipment configuration in the Github repository.
The  ```Devices/cassini/actual.json``` in the Github repository for verification is updated as follows, and the transceivers in the specified location of [OTDN-emulator](https://github.com/opennetworkinglab/ODTN-emulator) are updated. The  ```Devices/cassini/actual.json``` is updated as follows
```diff
{
...
				"name": "oe1",
				"config": {
					"name": "oe1"
				},
				"state": {
					"type": "openconfig-platform-types:TRANSCEIVER",
					"empty": false
				},
				"openconfig-platform-transceiver:transceiver": {
					"config": {
						+  "enabled": true,
						- "enabled": false,
						"form-factor-preconf": "openconfig-transport-types:CFP2_ACO"
					}
				}
...
}
```

### delete service.
```DELETE: http://localhost:8080/services?name=transceivers```. Delete the service with the name specified in the query parameters and change the configuration of the device.
Check the ``Services/transceivers`` folder in the verification Github repository as it will disappear.

### Get device information (after deleting the service)
Perform ```PUT: http://localhost:8080/sync/devices``` to reflect the current equipment configuration in the Github repository.
The  ```Devices/cassini/actual.json``` in the Github repository for verification will be updated as follows, and you will see that the configuration points that were submitted from the service will disappear.
```diff
{
...
				"name": "oe1",
				"config": {
					"name": "oe1"
				},
				"state": {
					"type": "openconfig-platform-types:TRANSCEIVER",
					"empty": false
				},
				"openconfig-platform-transceiver:transceiver": {
					"config": {
						- "enabled": true,
						"form-factor-preconf": "openconfig-transport-types:CFP2_ACO"
					}
				}
...
				"name": "oe2",
				"config": {
					"name": "oe2"
				},
				"state": {
					"type": "openconfig-platform-types:TRANSCEIVER",
					"empty": false
				},
				"openconfig-platform-transceiver:transceiver": {
					"config": {
						-  "enabled": false
						"form-factor-preconf": "openconfig-transport-types:CFP2_ACO"
					}
				}
...
}
```

### External tools used in this section
- [OTDN-emulator](https://github.com/opennetworkinglab/ODTN-emulator)