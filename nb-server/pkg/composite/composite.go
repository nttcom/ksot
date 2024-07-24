package composite

import (
	"fmt"

	"github.com/nttcom/ksot/nb-server/pkg/model/pathmap"
)

type CompositeInterface interface {
	CompositePathmaps(map[string]map[string]pathmap.PathMapInterface) (map[string]pathmap.PathMapInterface, error)
	CompositeAndUpdateAndReplaceKeyPathmaps(map[string]map[string]pathmap.PathMapInterface, map[string]map[string]pathmap.PathMapInterface) (map[string]pathmap.PathMapInterface, error)
	UpdateDeviceRefForComposite(serviceDeviceToPathmap map[string]map[string]pathmap.PathMapInterface, oldDeviceServiceToPathmap map[string]map[string]pathmap.PathMapInterface) (map[string]map[string]pathmap.PathMapInterface, error)
}

type Composite struct{}

var _ CompositeInterface = (*Composite)(nil)

func NewCompositeInterface() CompositeInterface {
	return &Composite{}
}

func (c *Composite) UpdateDeviceRefForComposite(serviceDeviceToPathmap map[string]map[string]pathmap.PathMapInterface, oldDeviceServiceToPathmap map[string]map[string]pathmap.PathMapInterface) (map[string]map[string]pathmap.PathMapInterface, error) {
	deviceServiceToPathmap := make(map[string]map[string]pathmap.PathMapInterface)
	for serviceName, deviceToPathmap := range serviceDeviceToPathmap {
		for deviceName, v := range deviceToPathmap {
			if _, ok := deviceServiceToPathmap[deviceName]; !ok {
				oldPathmapValue, checkSync := oldDeviceServiceToPathmap[deviceName]
				if !checkSync {
					return nil, fmt.Errorf("updateAndReplaceKey: not found device %v ref, run sync", deviceName)
				}
				deviceServiceToPathmap[deviceName] = oldPathmapValue
			}
			deviceServiceToPathmap[deviceName][serviceName] = v
		}
	}
	return deviceServiceToPathmap, nil
}

func (c *Composite) CompositePathmaps(deviceServiceToPathmap map[string]map[string]pathmap.PathMapInterface) (map[string]pathmap.PathMapInterface, error) {
	deviceToPathmap := make(map[string]pathmap.PathMapInterface)
	for deviceName, serviceToPathmap := range deviceServiceToPathmap {
		setPathmap, err := pathmap.NewPathMap(make(map[string]any))
		if err != nil {
			return nil, fmt.Errorf("CompositePathmaps: %w", err)
		}
		deviceToPathmap[deviceName] = setPathmap
		for _, vmap := range serviceToPathmap {
			err := deviceToPathmap[deviceName].Composite([]pathmap.PathMapInterface{vmap})
			if err != nil {
				return nil, fmt.Errorf("CompositePathmaps: %w", err)
			}
		}
	}
	return deviceToPathmap, nil
}

func (c *Composite) CompositeAndUpdateAndReplaceKeyPathmaps(serviceDeviceToPathmap map[string]map[string]pathmap.PathMapInterface, oldDeviceServiceToPathmap map[string]map[string]pathmap.PathMapInterface) (map[string]pathmap.PathMapInterface, error) {
	deviceServiceToPathmap, err := c.UpdateDeviceRefForComposite(serviceDeviceToPathmap, oldDeviceServiceToPathmap)
	if err != nil {
		return nil, fmt.Errorf("CompositePathmaps: %w", err)
	}
	deviceToPathmap, err := c.CompositePathmaps(deviceServiceToPathmap)
	if err != nil {
		return nil, fmt.Errorf("CompositePathmaps: %w", err)
	}
	return deviceToPathmap, nil
}
