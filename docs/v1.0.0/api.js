const spec = {
  "openapi": "3.0.0",
  "info": {
    "version": "1.0.0",
    "title": "K-SOT"
  },
  "paths": {
    "/services/{service}": {
      "get": {
        "tags": [
          "Northbound Server"
        ],
        "summary": "サービスを取得",
        "description": "登録されているサービスのinput.jsonを取得します。",
        "parameters": [{
          "name": "service",
          "in": "path",
          "description": "取得したいサービス名",
          "required": true,
          "type": "string",
          "example": "/transceivers"
        }],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": {
                    "transceivers": [
                        {
                            "down": [
                                "oe4",
                                "oe5"
                            ],
                            "name": "cassini1",
                            "nos": "netconf",
                            "up": [
                                "oe1",
                                "oe2",
                                "oe3"
                            ]
                        },
                        {
                            "down": [
                                "oe3"
                            ],
                            "name": "cassini2",
                            "nos": "netconf",
                            "up": [
                                "oe1",
                                "oe2"
                            ]
                        }
                    ]
                }                
                }
              }
            }
          }
        }
      },
    },
    "/services": {
      "post": {
        "tags": [
          "Northbound Server"
        ],
        "summary": "サービスの追加",
        "description": "リクエスト内のサービスをinput.jsonとして追加します。また、PathMapを作成しoutput.jsonとして追加し出力します。そして、作成したPathMapを元に実機のコンフィグを更新しset.jsonを更新します。",
        "requestBody": {
          "description": "追加したいサービスを送信する。",
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "example": {
                    "transceivers": {
                        "transceivers:transceivers":  [
                            {
                                "name": "cassini1",
                                "up": [],
                                "down": [
                                    "oe1",
                                    "oe2",
                                    "oe3",
                                    "oe4",
                                    "oe5"
                                ]
                            },
                            {
                                "name": "cassini2",
                                "up": [],
                                "down": [
                                    "oe1",
                                    "oe2",
                                    "oe3",
                                    "oe4",
                                    "oe5"
                                ]
                            }
                        ]
                    }
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": "[\"cassini1\", \"cassini2\"]"
                }
              }
            }
          }
        }
      },
      "put": {
        "tags": [
          "Northbound Server"
        ],
        "summary": "サービスの更新",
        "description": "リクエスト内のサービスをinput.jsonとして更新します。また、PathMapを作成しoutput.jsonとして追加し出力します。そして、作成したPathMapを元に実機のコンフィグを更新しset.jsonを更新します。",
        "requestBody": {
          "description": "追加したいサービスを送信する。",
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "example": {
                    "transceivers": {
                        "transceivers:transceivers":  [
                            {
                                "name": "cassini1",
                                "up": [],
                                "down": [
                                    "oe1",
                                    "oe2",
                                    "oe3",
                                    "oe4",
                                    "oe5"
                                ]
                            },
                            {
                                "name": "cassini2",
                                "up": [],
                                "down": [
                                    "oe1",
                                    "oe2",
                                    "oe3",
                                    "oe4",
                                    "oe5"
                                ]
                            }
                        ]
                    }
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": "[\"cassini1\", \"cassini2\"]"
                }
              }
            }
          }
        }
      },
      "delete": {
        "tags": [
          "Northbound Server"
        ],
        "summary": "指定したサービスモデルを削除",
        "description": "登録されているサービスモデルについて指定したサービスモデルを削除します。また、PathMapの更新を実行します。",
        "parameters": [{
          "name": "name",
          "in": "query",
          "description": "削除したいサービスモデル",
          "required": true,
          "type": "string",
          "example": "interfaces"
        }],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": "[\"cassini1\", \"cassini2\"]"
                }
              }
            }
          },
          "500": {
            "description": "INTERNAL SERVER ERROR",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": {
                    "message": "DeleteServices: Get \"http://github-server:8080/file?path=/ServiceModels/all.json\": dial tcp: lookup github-server on 127.0.0.11:53: no such host"
                  }
                }
              }
            }
          }
        }
      }
    },
    "/devices/{device}": {
      "get": {
        "tags": [
          "Northbound Server"
        ],
        "summary": "デバイスを取得",
        "description": "登録されているデバイスのset.jsonを取得します。",
        "parameters": [{
          "name": "device",
          "in": "path",
          "description": "取得したい装置名",
          "required": true,
          "type": "string",
          "example": "/cassini1"
        }],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": {
                    "data": {
                        "@xmlns": "urn:ietf:params:xml:ns:netconf:base:1.0",
                        "@xmlns:nc": "urn:ietf:params:xml:ns:netconf:base:1.0",
                        "components": {
                            "@xmlns": "http://openconfig.net/yang/platform",
                            "component": [{
                                    "config": {
                                        "name": "oe1"
                                    },
                                    "name": "oe1",
                                    "port": null,
                                    "state": {
                                        "empty": "false",
                                        "type": {
                                            "#text": "oc-platform-types:TRANSCEIVER",
                                            "@xmlns:oc-platform-types": "http://openconfig.net/yang/platform-types"
                                        }
                                    },
                                    "transceiver": {
                                        "@xmlns": "http://openconfig.net/yang/platform/transceiver",
                                        "config": {
                                            "enabled": "true",
                                            "form-factor-preconf": {
                                                "#text": "oc-opt-types:CFP2_ACO",
                                                "@xmlns:oc-opt-types": "http://openconfig.net/yang/transport-types"
                                            }
                                        }
                                    }
                            }]
                        }
                    }
                  }
                }
              }
            }
          }
        }
      }
    },
    "/sync/devices": {
      "put": {
        "tags": [
          "Northbound Server"
        ],
        "summary": "実機のコンフィグから装置のset.jsonとactual.jsonを更新する",
        "description": "実機のコンフィグからPathMapを導出し、Github上に/Pathmap/actual_all.jsonを追加します。",
        "parameters": [],
        "requestBody": {},
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": "[\"cassini1\", \"cassini2\"]"
                }
              }
            }
          },
          "400": {
            "description": "Bad Request",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": {
                    "message": "Diff: getPathMap: GetRequest: endpoint=http://github-server:8080/file?path=/PathMap/all_actual.json,  statusCode=500"
                  }
                }
              }
            }
          }
        }
      }
    },
    "/file": {
      "get": {
        "tags": [
          "Github Server"
        ],
        "summary": "Github上の指定したパスのデータを取得",
        "description": "Githubに登録されているデータについて指定したパスのデータをGithub上から取得します。",
        "parameters": [{
          "name": "path",
          "in": "query",
          "description": "取得したいデータのパス",
          "required": true,
          "type": "string",
          "example": "/ServiceModels/test3.json"
        }],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": {
                    "string_data": "{\n\t\"interfaces\": [\n\t\t{\n\t\t\t\"down\": [\n\t\t\t\t\"oe5\",\n\t\t\t\t\"oe2\",\n\t\t\t\t\"oe3\"\n\t\t\t],\n\t\t\t\"name\": \"cassini1\",\n\t\t\t\"nos\": \"netconf\",\n\t\t\t\"up\": [\n\t\t\t\t\"oe1\",\n\t\t\t\t\"oe5\"\n\t\t\t]\n\t\t},\n\t\t{\n\t\t\t\"down\": [],\n\t\t\t\"name\": \"cassini2\",\n\t\t\t\"nos\": \"netconf\",\n\t\t\t\"up\": [\n\t\t\t\t\"oe1\",\n\t\t\t\t\"oe2\",\n\t\t\t\t\"oe3\"\n\t\t\t]\n\t\t}\n\t]\n}"
                  }
                }
              }
            }
          },
          "400": {
            "description": "Bad Request",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": {
                    "message": "query parameter for file path does not exist:"
                  }
                }
              }
            }
          },
          "500": {
            "description": "Internal Server Error",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": {
                    "message": "GetFileData: open /work/github-server/gitrepo/ksot-testrepo/ServiceModels/test1.json: no such file or directory"
                  }
                }
              }
            }
          }
        }
      },
      "post": {
        "tags": [
          "Github Server"
        ],
        "summary": "Github上の指定したパスのデータを追加",
        "description": "リクエスト内のデータについてGithub上に追加します。",
        "parameters": [{
          "name": "X-POST-OPTION",
          "in": "header",
          "description": "POST処理についてのオプションを指定する",
          "required": true,
          "type": "string",
          "example": "new|new_safe|update"
        }],
        "requestBody": {
          "description": "追加したいパスとデータ",
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "example": {
                  "path": "/ServiceModels/D/test1.json",
                  "string_data": "{\n\t\"test1\": [\n\t\t{\n\t\t\t\"down\": [\n\t\t\t\t\"oe5\",\n\t\t\t\t\"oe2\",\n\t\t\t\t\"oe3\"\n\t\t\t],\n\t\t\t\"name\": \"cassini1\",\n\t\t\t\"nos\": \"netconf\",\n\t\t\t\"up\": [\n\t\t\t\t\"oe1\",\n\t\t\t\t\"oe5\"\n\t\t\t]\n\t\t},\n\t\t{\n\t\t\t\"down\": [],\n\t\t\t\"name\": \"cassini2\",\n\t\t\t\"nos\": \"netconf\",\n\t\t\t\"up\": [\n\t\t\t\t\"oe1\",\n\t\t\t\t\"oe2\",\n\t\t\t\t\"oe3\"\n\t\t\t]\n\t\t}\n\t]\n}"
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example":
                    "Post /ServiceModels/D/test1.json"
                }
              }
            }
          },
          "400": {
            "description": "Bad Request",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": {
                    "message": "PostFileData: unexpected end of JSON input"
                  }
                }
              }
            }
          },
          "500": {
            "description": "Internal Server Error",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": {
                    "message": "PostFileData: updateGithubWithLocal: open /work/github-server/gitrepo/ksot-testrepo/ServiceModel/interfaces.json: no such file or directory"
                  }
                }
              }
            }
          }
        }
      },
      "put": {
        "tags": [
          "Github Server"
        ],
        "summary": "Github上の指定したパスのデータをアップデート",
        "description": "Gtihubに登録されているデータとリクエスト内のデータについてマージしてGithub上に更新します。",
        "requestBody": {
          "description": "更新したいパスとデータ",
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "example":{
                  "path": "/ServiceModels/D/test3.json",
                  "string_data": "{\n\t\"test3\": [\n\t\t{\n\t\t\t\"down\": [\n\t\t\t\t\"oe5\",\n\t\t\t\t\"oe2\",\n\t\t\t\t\"oe3\"\n\t\t\t],\n\t\t\t\"name\": \"cassini1\",\n\t\t\t\"nos\": \"netconf\",\n\t\t\t\"up\": [\n\t\t\t\t\"oe1\",\n\t\t\t\t\"oe5\"\n\t\t\t]\n\t\t},\n\t\t{\n\t\t\t\"down\": [],\n\t\t\t\"name\": \"cassini2\",\n\t\t\t\"nos\": \"netconf\",\n\t\t\t\"up\": [\n\t\t\t\t\"oe1\",\n\t\t\t\t\"oe2\",\n\t\t\t\t\"oe3\"\n\t\t\t]\n\t\t}\n\t]\n}"
                }
              }
            }
          }
        },
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example":
                    "Put /ServiceModels/D/test3.json"
                }
              }
            }
          }
        }
      },
      "delete": {
        "tags": [
          "Github Server"
        ],
        "summary": "Github上の指定したパスのデータを削除",
        "description": "Githubに登録されているデータについて指定したパスのデータをGithub上から削除します。",
        "parameters": [{
          "name": "path",
          "in": "query",
          "description": "削除したいデータのパス",
          "required": true,
          "type": "string",
          "example": "/ServiceModels/D"
        }],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": "Delete: [/ServiceModels/D]"
                }
              }
            }
          },
          "400": {
            "description": "Bad Request",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": {
                    "message": "DelFileData: updateGithubWithLocal: remove /work/github-server/gitrepo/ksot-testrepo/ServiceModel/test3.json: no such file or directory"
                  }
                }
              }
            }
          }
        }
      }
    },
    "/devices ": {
      "get": {
        "tags": [
          "Southhbound Server"
        ],
        "summary": "全てのデバイス名一覧を取得",
        "description": "登録されている全てのデバイス名を取得します。",
        "parameters": [],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "	application/json": {
                "schema": {
                  "type": "object",
                  "example": { "devices": ["emulator1", "emulator2", "emulator3", "emulator4"] }
                }
              }
            }
          }
        }
      }
    },
    "/devices/{devicename}": {
      "get": {
        "tags": [
          "Southhbound Server"
        ],
        "summary": "指定したデバイスのデータを取得",
        "description": "指定したデバイスに接続し、データを取得します。",
        "parameters": [{
          "name": "devicename",
          "in": "path",
          "description": "データを取得したい装置名",
          "required": true,
          "type": "string",
          "example": "/emulator2"
        }],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "application/json": {
                "schema": {
                  "type": "object",
                  "example": {
                    "data": { "@xmlns": "urn:ietf:params:xml:ns:netconf:base:1.0", "@xmlns:nc": "urn:ietf:params:xml:ns:netconf:base:1.0", "components": { "@xmlns": "http://openconfig.net/yang/platform", "component": [{ "name": "oe1", "config": { "name": "oe1" }, "state": { "type": { "@xmlns:oc-platform-types": "http://openconfig.net/yang/platform-types", "#text": "oc-platform-types:TRANSCEIVER" }, "empty": "false" }, "port": null, "transceiver": { "@xmlns": "http://openconfig.net/yang/platform/transceiver", "config": { "enabled": "true", "form-factor-preconf": { "@xmlns:oc-opt-types": "http://openconfig.net/yang/transport-types", "#text": "oc-opt-types:CFP2_ACO" } } } }] } }
                  }
                }
              }
            }
          },
          " 500": {
            "description": "INTERNAL SERVER ERROR",
            "content": {
              "text/html": {
              }
            }
          }
        }
      },
      "post": {
        "tags": [
          "Southhbound Server"
        ],
        "summary": "指定したデバイスにデータを送信",
        "description": "指定したデバイスに接続し、データを送信します。",
        "parameters": [{
          "name": "devicename",
          "in": "path",
          "description": "データを取得したい装置名",
          "required": true,
          "type": "string",
          "example": "/emulator2"
        }],
        "responses": {
          "200": {
            "description": "OK",
            "content": {
              "text/html": {
                "schema": {
                  "type": "object",
                  "example": '<?xml version="1.0" encoding="utf-8"?><config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0"></config>'
                }
              }
            }
          },
          " 500": {
            "description": "INTERNAL SERVER ERROR",
            "content": {
              "text/html": {
              }
            }
          }
        }
      }
    }
  }
}