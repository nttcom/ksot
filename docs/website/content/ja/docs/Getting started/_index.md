---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 1
description: >
  K-SOTを動かしてみよう
---

このチュートリアルでは、以下のステップでK-SOTを構築し、動作確認を行います。

1. ローカルKubernetesクラスタ作成
2. 検証用のGithubレポジトリの作成
3. K-SOTをインストール、各種ファイル設定
4. 装置エミュレータを立ち上げる
5. K-SOTのデプロイ
6. 動作確認

## 事前準備 

1. [golang (v1.21.0+))](https://go.dev/doc/install) をインストールしてください。装置設定導出ロジックを書くのに必要です。
2. [docker (v20.10.24+)](https://docs.docker.com/engine/install) をインストールしてください。kindの実行に必要です。
3. [kind (v0.12.0+)](https://kind.sigs.k8s.io/docs/user/quick-start/) をインストールしてください。ローカルのKubernetesクラスタの構築に使用します。
4. [kubectl (v1.25.4+)](https://kubernetes.io/docs/tasks/tools/) をインストールしてください。Kubernetesクラスタに対してコマンドを実行する際に使用します。

## 1. ローカルKubernetesクラスタ作成

以下のコマンドを実行して、ローカルのKubernetesクラスタを立ち上げます。

```bash
kind create cluster --name ksot-started
```

実行が完了したら、以下のコマンドを実行して、Kubernetesクラスタが正常に作成されたことを確認します。

```bash
kubectl cluster-info
```

以下のような出力が表示されたら、ローカルクラスタは正常に動作しています。

```bash
Kubernetes control plane is running at https://127.0.0.1:52096
CoreDNS is running at https://127.0.0.1:52096/api/v1/namespaces/kube-system/services/kube-dns:dns/proxy

To further debug and diagnose cluster problems, use 'kubectl cluster-info dump'.
```


## 2. 検証用のGithubレポジトリの作成

K-SOTをGithubを用いてネットワーク装置の設定を管理するには、GitHubレポジトリを作成する必要があります。
以下の手順でGitHubレポジトリを作成してください。

1. GitHubにログインし、`ksot-example` という名前で新規レポジトリを作成してください。レポジトリの公開・非公開については、好きな方を選択してください。

2. 作成したレポジトリのルートに、以下の2つのフォルダを作成してください。
   - `Devices`
   - `Services`


## 3. K-SOTをインストール、各種ファイル設定
### K-SOTのレポジトリをclone、submoduleの初期化
レポジトリのclone
```bash
git clone https://github.com/nttcom/ksot.git
```
submoduleの初期化
```bash
git submodule update --init --recursive
```

### envファイルの設定
上でcloneしたK-SOTレポジトリに対して、以下envファイルを配置してください。

/.env
```bash
KIND_NAME=ksot-started
GITHUB_REPO_URL=https://{Githubユーザー名}:{Githubユーザートークン}@github.com/{Githubユーザー名}/ksot-example
GITHUB_USER_NAME={Githubユーザー名}
GITHUB_USER_MAIL={Github登録メールアドレス}
```

/github-server/config/overlays/test/.env
```bash
GITHUB_REPO_URL=https://{Githubユーザー名}:{Githubユーザートークン}@github.com/{Githubユーザー名}/ksot-example
```

### yangファイルの設定
#### 装置向けyangの設定
本記事で管理対象の装置とする[ODTN-emulator](https://github.com/opennetworkinglab/ODTN-emulator)の[YANGフォルダ]( https://github.com/opennetworkinglab/ODTN-emulator/tree/master/emulator-oc-cassini/yang/openconfig-odtn)内のyangを全て取得。```nb-server/pkg/tf/yang/devices```配下にフォルダ(名前: cassini)を作成して先ほど取得したyangファイルを配置してください。
#### サービス向けyangの設定
```nb-server/pkg/tf/yang/services```配下にフォルダ(名前: transceivers)を作成して以下内容のyangファイルをその中に配置します。

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

### 装置設定導出パッケージの設定
```nb-server/pkg/tf/transceivers.go```を以下内容で作成します。
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
```nb-server/pkg/tf/tf.go```を以下の内容に変更します。
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

## 4. 装置エミュレータを立ち上げる
事前にエミュレーターの接続情報を記載したenvファイルを追加
sb-server/config/overlays/test/.env
```bash
cassini=root
```

以下コマンドで[OTDN-emulator](https://github.com/opennetworkinglab/ODTN-emulator)を立ち上げます。
```bash
docker pull onosproject/oc-cassini:0.21
docker run -it -d --name odtn-emulator_openconfig_cassini_1 -p 11002:830 onosproject/oc-cassini:0.21
```

以下のようにコンテナが立ち上がっているか確認してください。
```bash
% docker ps
CONTAINER ID  IMAGE             COMMAND          CREATED    STATUS    PORTS                   NAMES
52da5c9f1c79  onosproject/oc-cassini:0.21  "sh /root/script/pus…"  4 hours ago  Up 4 hours  22/tcp, 8080/tcp, 0.0.0.0:11002->830/tcp  odtn-emulator_openconfig_cassini_1
```

コントローラーから[OTDN-emulator](https://github.com/opennetworkinglab/ODTN-emulator)への接続情報追加するため、K-SOTレポジトリに```sb-server/connect.json```を作成し以下の内容で保存します。
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

## 5. K-SOTのデプロイ
以下コマンドをK-SOTレポジトリのルートで実行してください。
```bash
make getting-started
```
実行後```kubectl get pod```で起動しているpodを確認すると、以下3つのpodが作成されています。

```bash
NAMESPACE            NAME                                                 READY   STATUS    RESTARTS   AGE
default              ksot-github-deployment-766b6ff499-72zkr              1/1     Running   0          24s
default              ksot-nb-deployment-967757888-dqzw7                   1/1     Running   0          24s
default              ksot-sb-deployment-6d89c6f85c-zksl5                  1/1     Running   0          24s
...
```

## 6. 動作確認
### ポートフォワードの実施
まずNorthboundとなるk8s serviceにport-forwardを実施しておきます。

```bash
kubectl port-forward svc/ksot-nb-service 8080:8080
```

### 装置情報の取得
```PUT: http://localhost:8080/sync/devices```を実施し、Githubレポジトリに現在の装置設定を反映させます。
検証用Githubレポジトリの```Devices/cassini```に以下のファイルが作成されていれば成功です。
- actual.json: 実機から取得した設定
- ref.json: 各サービスから設定されているパラメータ(初期値は空のJSON)
- set.json: コントローラーが投入した設定(初期値は実機から取得したものと同じ)


### サービスの作成
以下JSONをbodyとして```POST: http://localhost:8080/services```を実施、bodyに指定されたサービスを作成し装置の設定を変更します。
今回のサービスではtransceiversのud/downを実施するサービスを作成します。なお作成するサービスについては事前に装置設定導出パッケージを追加する必要があります(前の手順)。
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
検証用Githubレポジトリの```Services/transceivers```に以下のファイルが作成されていることを確認してください。
- input.json: 投入したサービスの値
- output.json: 設定を変更した装置のxpathと対応する値

### 装置情報の取得(サービス作成後)
```PUT: http://localhost:8080/sync/devices```を実施し、Githubレポジトリに現在の装置設定を反映します。
検証用Githubレポジトリの```Devices/cassini/actual.json```が以下のように更新され、[OTDN-emulator](https://github.com/opennetworkinglab/ODTN-emulator)の指定した箇所のtransceiversがdownしていることがわかります。
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

### サービスの更新
以下JSONをbodyとして```PUT: http://localhost:8080/services```を実施、bodyに指定されたサービスを更新し装置の設定を変更します。
今回はoe1のtransceiverについてUPするよう更新します。
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

### 装置情報の取得(サービス更新後)
```PUT: http://localhost:8080/sync/devices```を実施し、Githubレポジトリに現在の装置設定を反映します。
検証用Githubレポジトリの```Devices/cassini/actual.json```が以下のように更新され、[OTDN-emulator](https://github.com/opennetworkinglab/ODTN-emulator)の指定した箇所のtransceiversがUPしていることがわかります。
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

### サービスの削除
```DELETE: http://localhost:8080/services?name=transceivers```を実施。クエリパラメーターに指定した名前のサービスを削除し、装置の設定を変更します。
検証用Githubレポジトリの```Services/transceivers```フォルダが消えるので確認してください。

### 装置情報の取得(サービス削除後)
```PUT: http://localhost:8080/sync/devices```を実施し、Githubレポジトリに現在の装置設定を反映します。
検証用Githubレポジトリの```Devices/cassini/actual.json```が以下のように更新され、サービスから投入していたコンフィグ箇所が消えていることがわかります。

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

### 本項で使用している外部ツール
- [OTDN-emulator](https://github.com/opennetworkinglab/ODTN-emulator)