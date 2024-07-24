package configurator

import (
	"github.com/nttcom/ksot/nb-server/pkg/api"
)

func netconfLogic(deviceName string, setconfig []byte, roolbackConfig []byte, sb api.SbApiInterface) (func() error, error) {
	roolbackFunc := func() error {
		return sb.PostFileRequest("/devices/netconf/"+deviceName, roolbackConfig, 120)
	}
	err := sb.PostFileRequest("/devices/netconf/"+deviceName, setconfig, 120)
	return roolbackFunc, err
}
