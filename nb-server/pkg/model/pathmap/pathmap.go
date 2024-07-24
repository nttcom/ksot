package pathmap

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/exp/maps"
)

type PathMapInterface interface {
	SetValue(path string, value any, opt map[string]string) error
	GetKeys() []string
	GetValue(path string) (any, bool)
	GetMapInterface() map[string]interface{}
	DeleteValue(path string) bool
	Diff(other PathMapInterface) (*DiffResult, error)
	Composite([]PathMapInterface) error
}

type PathMap map[string]*PathMapValue

var _ PathMapInterface = (PathMap)(nil)

func NewPathMap(jmap map[string]interface{}) (PathMap, error) {
	newPathMap := make(PathMap)
	for k, v := range jmap {
		err := newPathMap.SetValue(k, v, make(map[string]string))
		if err != nil {
			return newPathMap, fmt.Errorf("NewPathMap: %w", err)
		}
	}
	return newPathMap, nil
}

func (pm PathMap) SetValue(path string, value any, opt map[string]string) error {
	pathList := strings.Split(filepath.Clean(path), "/")[1:]
	pathValue, err := NewPathMapValue(pathList, value, opt)
	if err != nil {
		return fmt.Errorf("SetValue: %w", err)
	}
	pm[filepath.Clean(path)] = pathValue
	return nil
}

func (pm PathMap) GetKeys() []string {
	return maps.Keys(pm)
}

func (pm PathMap) GetValue(path string) (any, bool) {
	v, ok := pm[filepath.Clean(path)]
	if ok {
		return v.value, ok
	}
	return nil, ok
}

func (pm PathMap) GetMapInterface() map[string]interface{} {
	result := make(map[string]interface{})
	for path, pathMapValue := range pm {
		result[path] = pathMapValue.value
	}
	return result
}

func (pm PathMap) GetPath(path string) ([]string, bool) {
	v, ok := pm[filepath.Clean(path)]
	return v.path, ok
}

func (pm PathMap) GetOption(path string) (map[string]string, bool) {
	v, ok := pm[filepath.Clean(path)]
	return v.option, ok
}

func (pathMap PathMap) DeleteValue(path string) bool {
	if _, ok := pathMap[filepath.Clean(path)]; ok {
		delete(pathMap, filepath.Clean(path))
		return true
	}
	return false
}

func (pm PathMap) Diff(other PathMapInterface) (*DiffResult, error) {
	result := NewDiffResult()
	stackKeys := make(map[string]bool)
	for _, path := range other.GetKeys() {
		newValue, _ := other.GetValue(path)
		if oldValue, ok := pm.GetValue(path); ok {
			if reflect.DeepEqual(newValue, oldValue) {
				stackKeys[path] = true
				continue
			}
			err := result.Update.SetValue(path, newValue, make(map[string]string))
			if err != nil {
				return nil, fmt.Errorf("DiffPathMap: %w", err)
			}
			stackKeys[path] = true
			continue
		}
		err := result.Create.SetValue(path, newValue, make(map[string]string))
		if err != nil {
			return nil, fmt.Errorf("DiffPathMap: %w", err)
		}
		stackKeys[path] = true
		continue
	}
	for _, path := range pm.GetKeys() {
		oldValue, _ := pm.GetValue(path)
		if _, ok := stackKeys[path]; ok {
			continue
		}
		err := result.Delete.SetValue(path, oldValue, make(map[string]string))
		if err != nil {
			return nil, fmt.Errorf("DiffPathMap: %w", err)
		}
	}
	return result, nil
}

func (pm PathMap) Composite(pmList []PathMapInterface) error {
	for _, v := range pmList {
		err := mergePathMapValue(pm, v)
		if err != nil {
			return fmt.Errorf("CompositePathMap: %w", err)
		}
	}
	return nil
}

type PathMapValue struct {
	path   []string
	value  any
	option map[string]string
}

func NewPathMapValue(path []string, value any, option map[string]string) (*PathMapValue, error) {
	switch v := value.(type) {
	case bool, int, float64, uint, string, []bool, []int, []float64, []uint, []string:
		return &PathMapValue{path: path, value: v, option: option}, nil
	default:
		return nil, fmt.Errorf("checkTypeForPathMapValue: noexpected pahtmap value %v type: %t", v, v)
	}
}

func NewPathMapValueSafe(path []string, value any, option map[string]string) *PathMapValue {
	switch v := value.(type) {
	case bool, int, float64, uint, string, []bool, []int, []float64, []uint, []string:
		return &PathMapValue{path: path, value: v, option: option}
	default:
		return nil
	}
}

type DiffResult struct {
	Create PathMap
	Update PathMap
	Delete PathMap
}

func NewDiffResult() *DiffResult {
	c, _ := NewPathMap(make(map[string]interface{}))
	u, _ := NewPathMap(make(map[string]interface{}))
	d, _ := NewPathMap(make(map[string]interface{}))
	return &DiffResult{
		Create: c,
		Update: u,
		Delete: d,
	}
}

type pathMapValueType interface {
	bool | int | float64 | uint | string
}

func mergeList[T pathMapValueType](x, y []T) []T {
	x = append(x, y...)

	result := make([]T, 0)
	duplicateCheckMap := make(map[T]bool)

	for _, v := range x {
		if _, ok := duplicateCheckMap[v]; ok {
			continue
		}
		result = append(result, v)
		duplicateCheckMap[v] = true
	}

	return result
}

func checkConflictOrMergeListForPathMapValue(x, y any) (any, error) {
	switch vx := x.(type) {
	case bool, int, float64, uint, string:
		if reflect.DeepEqual(x, y) {
			return vx, nil
		}
		return nil, fmt.Errorf("checkConflictOrMergeListForPathMapValue: conflict error %v : %v", x, y)
	case []bool:
		vy, ok := y.([]bool)
		if !ok {
			return nil, fmt.Errorf("checkConflictOrMergeListForPathMapValue: conflict error %v : %v", x, y)
		}
		if reflect.DeepEqual(vx, vy) {
			return vx, nil
		}
		return mergeList(vx, vy), nil
	case []int:
		vy, ok := y.([]int)
		if !ok {
			return nil, fmt.Errorf("checkConflictOrMergeListForPathMapValue: conflict error %v : %v", x, y)
		}
		if reflect.DeepEqual(vx, vy) {
			return vx, nil
		}
		return mergeList(vx, vy), nil
	case []float64:
		vy, ok := y.([]float64)
		if !ok {
			return nil, fmt.Errorf("chackTypeAndConflictForPathMapValue: conflict error %v : %v", x, y)
		}
		if reflect.DeepEqual(vx, vy) {
			return vx, nil
		}
		return mergeList(vx, vy), nil
	case []uint:
		vy, ok := y.([]uint)
		if !ok {
			return nil, fmt.Errorf("checkConflictOrMergeListForPathMapValue: conflict error %v : %v", x, y)
		}
		if reflect.DeepEqual(vx, vy) {
			return vx, nil
		}
		return mergeList(vx, vy), nil
	case []string:
		vy, ok := y.([]string)
		if !ok {
			return nil, fmt.Errorf("checkConflictOrMergeListForPathMapValue: conflict error %v : %v", x, y)
		}
		if reflect.DeepEqual(vx, vy) {
			return vx, nil
		}
		return mergeList(vx, vy), nil
	default:
		return nil, fmt.Errorf("checkConflictOrMergeListForPathMapValue: noexpected pahtmap value %v type: %t", vx, vx)
	}
}

func mergePathMapValue(x PathMapInterface, y PathMapInterface) error {
	for _, path := range x.GetKeys() {
		xv, _ := x.GetValue(path)
		if yv, ok := y.GetValue(path); ok {
			resultValue, err := checkConflictOrMergeListForPathMapValue(xv, yv)
			if err != nil {
				return fmt.Errorf("mergePathMapValue: %w", err)
			}
			y.DeleteValue(path)
			err = x.SetValue(path, resultValue, make(map[string]string))
			if err != nil {
				return fmt.Errorf("mergePathMapValue: %w", err)
			}
			continue
		}
		resultValue, err := checkConflictOrMergeListForPathMapValue(xv, xv)
		if err != nil {
			return fmt.Errorf("mergePathMapValue: %w", err)
		}
		err = x.SetValue(path, resultValue, make(map[string]string))
		if err != nil {
			return fmt.Errorf("mergePathMapValue: %w", err)
		}
	}
	for _, path := range y.GetKeys() {
		yv, _ := y.GetValue(path)
		resultValue, err := checkConflictOrMergeListForPathMapValue(yv, yv)
		if err != nil {
			return fmt.Errorf("mergePathMapValue: %w", err)
		}
		err = x.SetValue(path, resultValue, make(map[string]string))
		if err != nil {
			return fmt.Errorf("mergePathMapValue: %w", err)
		}
	}
	return nil
}
