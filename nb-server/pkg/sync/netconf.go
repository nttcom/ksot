package sync

import (
	"fmt"

	"github.com/nttcom/ksot/nb-server/pkg/api"
	"github.com/nttcom/ksot/nb-server/pkg/util/libyang"
)

type SyncNetconf struct {
	*syncBase
}

var _ SyncInterface = (*SyncNetconf)(nil)

func (syncNetconf *SyncNetconf) SyncDevice(sb api.SbApiInterface, lb libyang.LibyangInterface, deviceName string) ([]byte, error) {
	path := fmt.Sprintf("/devices/%v", deviceName)
	xml, err := sb.GetRequest(path, 120)
	if err != nil {
		return []byte{}, fmt.Errorf("SyncDevice %v (netconf): %w", deviceName, err)
	}
	validation, jsonByte, err := lb.ValidateAndConvertXMLToJSON(deviceName, xml)
	if !validation || err != nil {
		return []byte{}, fmt.Errorf("SyncDevice %v (netconf): %w", deviceName, err)
	}
	return jsonByte, nil
}
