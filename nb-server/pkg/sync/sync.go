package sync

import (
	"fmt"

	"github.com/nttcom/ksot/nb-server/pkg/api"
	"github.com/nttcom/ksot/nb-server/pkg/util/libyang"
)

type SyncInterface interface {
	// TODO パスマップのsync機能
	SyncPathMap(configMap map[string]interface{}, pathMap map[string]interface{}) (map[string]interface{}, error)
	SyncDevice(sb api.SbApiInterface, lb libyang.LibyangInterface, deviceName string) ([]byte, error)
}

var SyncInterfaceMap = map[string]SyncInterface{}

type syncBase struct{}

var _ SyncInterface = (*syncBase)(nil)

func (sync *syncBase) SyncPathMap(configMap map[string]interface{}, pathMap map[string]interface{}) (map[string]interface{}, error) {
	return nil, fmt.Errorf("SyncPathMap: undefined interface function")
}

func (syncNetconf *syncBase) SyncDevice(sb api.SbApiInterface, lb libyang.LibyangInterface, deviceName string) ([]byte, error) {
	return []byte{}, fmt.Errorf("SyncPathMap: undefined interface function")
}

func init() {
	SyncInterfaceMap["netconf"] = &SyncNetconf{}
}
