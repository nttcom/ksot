package oc_interface

import (
	ocdemo "github.com/nttcom/kuesta/testdata/types/pkg/ocdemo"
)

#Input: {
	// kuesta:"key=1"
	device: string
	// kuesta:"key=2"
	port:   uint16
	noShut: bool
	desc:   string | *""
	mtu:    uint16 | *9000
}

#Template: {
	input: #Input

	let _portName = "Ethernet\(input.port)"

	output: devices: "\(input.device)": config: {
		ocdemo.#Device
		Interface: "\(_portName)": {
			Name:        _portName
			Description: input.desc
			Enabled:     input.noShut
			AdminStatus: 1
			OperStatus:  1
			Type:        ocdemo.#IETFInterfaces_InterfaceType_ethernetCsmacd
			Mtu:         input.mtu
			...
		}
		...
	}
}
