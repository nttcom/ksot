package diff

import (
	"fmt"

	"github.com/nttcom/ksot/nb-server/pkg/model/pathmap"
)

type DiffInterface interface {
	DiffPathmaps(map[string]pathmap.PathMapInterface, map[string]pathmap.PathMapInterface) (map[string]*pathmap.DiffResult, error)
}

type Diff struct{}

var _ DiffInterface = (*Diff)(nil)

func NewDiffInterface() DiffInterface {
	return &Diff{}
}

func (d *Diff) DiffPathmaps(oldDeviceToPathmap map[string]pathmap.PathMapInterface, newDeviceToPathmap map[string]pathmap.PathMapInterface) (map[string]*pathmap.DiffResult, error) {
	result := make(map[string]*pathmap.DiffResult)
	for deviceName, pathmapValue := range oldDeviceToPathmap {
		diffValue, err := pathmapValue.Diff(newDeviceToPathmap[deviceName])
		if err != nil {
			return nil, fmt.Errorf("DiffPathmaps: %w", err)
		}
		result[deviceName] = diffValue
	}
	return result, nil
}
