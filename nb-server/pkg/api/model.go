package api

type DeviceInfo struct {
	Name string `json:"name"`
	If   string `json:"if"`
}

type ResGetDevices struct {
	Devices []DeviceInfo `json:"devices"`
}
