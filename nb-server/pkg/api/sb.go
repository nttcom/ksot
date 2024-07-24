package api

import (
	"encoding/json"
	"fmt"

	"github.com/nttcom/ksot/nb-server/pkg/model/orderedmap"
)

type SbApiInterface interface {
	API
	GetDevice(string) (orderedmap.OrderedmapInterfaces, error)
	SetDevice(string, interface{}) ([]byte, error)
	GetDeviceInfos() (map[string]string, error)
}

type sbAPI struct {
	*api
}

var _ SbApiInterface = (*sbAPI)(nil)

func NewSbApi(url string) *sbAPI {
	return &sbAPI{
		&api{
			baseURL: url,
		},
	}
}

func (sb *sbAPI) GetDevice(name string) (orderedmap.OrderedmapInterfaces, error) {
	res, err := sb.GetRequest("/devices/"+name, 300)
	if err != nil {
		return nil, fmt.Errorf("GetDevice: %w", err)
	}
	result, err := orderedmap.New(res)
	if err != nil {
		return nil, fmt.Errorf("GetDevice: %w", err)
	}
	return result, nil
}

func (sb *sbAPI) SetDevice(name string, setConfig interface{}) ([]byte, error) {
	return sb.PostRequest("/devices/"+name, "application/json", setConfig, 300)
}

func (sb *sbAPI) GetDeviceInfos() (map[string]string, error) {
	result := make(map[string]string)
	res, err := sb.GetRequest("/devices", 300)
	if err != nil {
		return nil, fmt.Errorf("GetDeviceNames: %w", err)
	}
	var resBody ResGetDevices
	if err := json.Unmarshal(res, &resBody); err != nil {
		return nil, fmt.Errorf("GetDeviceNames: %w", err)
	}
	for _, v := range resBody.Devices {
		result[v.Name] = v.If
	}
	return result, nil
}
