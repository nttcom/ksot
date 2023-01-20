package model

// TestDevice represents the /device YANG schema element.
type TestDevice struct {
	Interface map[string]*Interface `path:"interfaces/interface" module:"openconfig-interfaces/openconfig-interfaces"`
	Vlan      map[uint16]*Vlan      `path:"vlans/vlan" module:"openconfig-vlan/openconfig-vlan"`
}
