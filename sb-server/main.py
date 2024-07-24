from flask import Flask, request, jsonify, Response
from ncclient import manager
from pygnmi.client import gNMIclient
import xmltodict
import xml.etree.ElementTree as ET
import os
import io
import json
from flask_cors import CORS
import logging
logging.basicConfig(level=logging.DEBUG)

app = Flask(__name__)
CORS(app)

# gnmi get
def getGnmiDevice(deviceCfg, password):
    host = (deviceCfg["ip"], deviceCfg["port"])
    username = deviceCfg["username"]
    skip_verify=deviceCfg["skipVerify"]
    insecure=deviceCfg["insecure"]
    encoding=deviceCfg["encoding"]
    PATH = []
    if request.args.get("path") == None:
        if "rootpath" in deviceCfg:
             root = {}
             for path in deviceCfg["rootpath"]:
                 with gNMIclient(target=host, username=username, password = password, skip_verify=skip_verify, insecure=insecure) as gc:
                     result = gc.get(path=["openconfig:" + path],encoding=encoding)
                 root[path] = result["notification"][0]["update"][0]["val"]
             return json.dumps(root)
    else:
        PATH.append("openconfig:" + request.args['path'])
    with gNMIclient(target=host, username=username, password = password, skip_verify=skip_verify, insecure=insecure) as gc:
        result = gc.get(path=PATH,encoding=encoding)
    return json.dumps(result["notification"][0]["update"][0]["val"])

# netconf by xml
def getNetconfDevice(deviceCfg, password):
    host = deviceCfg["ip"]
    port = deviceCfg["port"]
    user = deviceCfg["username"]
    hostkeyVerify = deviceCfg["hostKeyVerify"]
    with manager.connect(host=host, port=port, username=user, password=password, hostkey_verify=hostkeyVerify, device_params={'name':'default'}) as m:
        c = m.get_config(source='running').data_xml
    dict_data = xmltodict.parse(c)
    if 'data' in dict_data:
        dict_data = dict_data["data"]
    if 'config' in dict_data:
        dict_data = dict_data["config"]
    # xmltodictで扱うxmlはroot要素が必要
    root_dict = {"root": dict_data}
    confread = xmltodict.unparse(root_dict, full_document=False)
    result = confread[confread.find(">")+1:confread.rfind("<")]
    return result

# gnmi set
def setGnmiDevice(deviceCfg, password, req_json):
    host = (deviceCfg["ip"], deviceCfg["port"])
    username = deviceCfg["username"]
    skip_verify=deviceCfg["skipVerify"]
    insecure=deviceCfg["insecure"]
    encoding=deviceCfg["encoding"]
    u = []
    for k, v in req_json.items():
        u.append(("openconfig:" + k, v))
    with gNMIclient(target=host, username=username, password = password, skip_verify=skip_verify, insecure=insecure) as gc:
        result = gc.set(update=u, encoding=encoding)
    return result

# device get
@app.route("/devices",  methods=['GET'])
def getDevices():
    connected_file = open('connect.json', 'r')
    connected_map = json.load(connected_file)
    devices = []
    for k, v in connected_map.items():
        device_info = {}
        device_info["name"] = k
        device_info["if"] = v["if"]
        devices.append(device_info)
    result = {}
    result["devices"] = devices
    result = Response(json.dumps(result), mimetype='application/json')
    return result

@app.route("/devices/<devicename>", methods=['GET'])
def getDevice(devicename):
    connected_file = open('connect.json', 'r')
    connected_map = json.load(connected_file)
    device_cfg = connected_map[devicename]
    password = os.environ.get(devicename, 'password')
    if device_cfg["if"] == "gnmi":
        return getGnmiDevice(device_cfg, password)
    elif device_cfg["if"] == "netconf":
        return getNetconfDevice(device_cfg, password)
    else:
        return jsonify({'message': 'noexpected if'}), 500


# device set
@app.route("/devices/<devicename>",  methods=['POST'])
def setDevice(devicename):
    connected_file = open('connect.json', 'r')
    connected_map = json.load(connected_file)
    device_cfg = connected_map[devicename]
    password = os.environ.get(devicename, 'password')
    req_json = request.get_json()
    if device_cfg["if"] == "gnmi":
        return setGnmiDevice(device_cfg, password, req_json)
    elif device_cfg["if"] == "netconf":
        return setNetconfDevice(device_cfg, password, req_json)
    else:
        return jsonify({'message': 'noexpected if'}), 500

# netconf by xml
@app.route("/netconf/get/<devicename>", methods=['GET'])
def getNetconfByXml(devicename):
    connected_file = open('connect.json', 'r')
    connected_map = json.load(connected_file)
    host = connected_map[devicename]["ip"]
    port = connected_map[devicename]["port"]
    user = connected_map[devicename]["username"]
    password = os.environ.get(devicename, 'password')
    with manager.connect(host=host, port=port, username=user, password=password, hostkey_verify=connected_map[devicename]["hostKeyVerify"], device_params={'name':'default'}) as m:
        c = m.get_config(source='running').data_xml
    return c

@app.route("/devices/netconf/<devicename>",  methods=['POST'])
def setNetconfDevice(devicename):
    connected_file = open('connect.json', 'r')
    connected_map = json.load(connected_file)
    host = connected_map[devicename]["ip"]
    port = connected_map[devicename]["port"]
    user = connected_map[devicename]["username"]
    password = os.environ.get(devicename, 'password')
    conf = request.files['set']
    confstream = io.TextIOWrapper(conf.stream,encoding='utf-8')
    confread = '<config xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0">' + confstream.read() + "</config>"
    with manager.connect(host=host,port=port,username=user,password=password,hostkey_verify=connected_map[devicename]["hostKeyVerify"],device_params={'name':'default'}) as m:
        c = m.edit_config(target='running',config=confread,default_operation="replace")
    return  confread

# メイン関数
if __name__ == "__main__":
    app.run("0.0.0.0", debug=True)