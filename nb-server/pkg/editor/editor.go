package editor

import (
	"fmt"

	"github.com/nttcom/ksot/nb-server/pkg/model/orderedmap"
	"github.com/nttcom/ksot/nb-server/pkg/model/pathmap"
)

type EditorInterface interface {
	EditConfigByPathmapDiff(map[string]orderedmap.OrderedmapInterfaces, map[string]*pathmap.DiffResult) error
}

type Editor struct{}

var _ EditorInterface = (*Editor)(nil)

func NewEditorInterface() EditorInterface {
	return &Editor{}
}

func (d *Editor) EditConfigByPathmapDiff(deviceNameToOrderedmap map[string]orderedmap.OrderedmapInterfaces, deviceNameTodiff map[string]*pathmap.DiffResult) error {
	for deviceName, diffValue := range deviceNameTodiff {
		createPaths := diffValue.Create.GetKeys()
		updatePaths := diffValue.Update.GetKeys()
		deletePaths := diffValue.Delete.GetKeys()
		for _, v := range createPaths {
			setPath, _ := diffValue.Create.GetPath(v)
			setValue, _ := diffValue.Create.GetValue(v)
			err := deviceNameToOrderedmap[deviceName].RecursiveSet(setPath, setValue)
			if err != nil {
				return fmt.Errorf("EditConfigByPathmapDiff: %w", err)
			}
		}
		for _, v := range updatePaths {
			setPath, _ := diffValue.Update.GetPath(v)
			setValue, _ := diffValue.Update.GetValue(v)
			err := deviceNameToOrderedmap[deviceName].RecursiveSet(setPath, setValue)
			if err != nil {
				return fmt.Errorf("EditConfigByPathmapDiff: %w", err)
			}
		}
		for _, v := range deletePaths {
			setPath, _ := diffValue.Delete.GetPath(v)
			err := deviceNameToOrderedmap[deviceName].RecursiveDelete(setPath)
			if err != nil {
				return fmt.Errorf("EditConfigByPathmapDiff: %w", err)
			}
		}
	}
	return nil
}
