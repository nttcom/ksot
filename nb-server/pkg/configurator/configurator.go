package configurator

import (
	"fmt"

	"github.com/nttcom/ksot/nb-server/pkg/api"
	"golang.org/x/exp/maps"
)

var (
	configureLogicMap = make(ConfigureLogicMap)
)

type ConfigureLogicMap map[string]func(string, []byte, []byte, api.SbApiInterface) (func() error, error)

const (
	NETCONF = "netconf"
	GNMI    = "gnmi"
)

type ConfiguratorInterface interface {
	Configure(map[string]string, map[string][]byte, map[string][]byte) error
}

type Configurator struct {
	sb api.SbApiInterface
}

var _ ConfiguratorInterface = (*Configurator)(nil)

func NewConfiguratorInterface(sb api.SbApiInterface) ConfiguratorInterface {
	return &Configurator{sb: sb}
}

func init() {
	configureLogicMap[NETCONF] = netconfLogic
}

func (c *Configurator) Configure(deviceNameToIfMap map[string]string, deviceNameToConfigMap map[string][]byte, oldDeviceNameToConfigMap map[string][]byte) error {
	roolbackFuns := make(map[string]func() error)
	for deviceName, iface := range deviceNameToIfMap {
		rfunc, err := configureLogicMap[iface](deviceName, deviceNameToConfigMap[deviceName], oldDeviceNameToConfigMap[deviceName], c.sb)
		if err != nil {
			successDevices := maps.Keys(roolbackFuns)
			for i, deviceName := range successDevices {
				err := roolbackFuns[deviceName]()
				if err != nil {
					return fmt.Errorf("Configure: failed roolback devices %v", successDevices[i:])
				}
			}
			return fmt.Errorf("Configure: success roolback %v", successDevices)
		}
		roolbackFuns[deviceName] = rfunc
	}
	return nil
}
