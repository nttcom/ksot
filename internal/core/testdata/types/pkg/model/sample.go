package model

type TestDevice struct {
	Interface map[string]*Interface `path:"interfaces/interface" module:"openconfig-interfaces/openconfig-interfaces"`
	Vlan      map[string]*Vlan      `path:"vlans/vlan" module:"openconfig-vlan/openconfig-vlan"`
}