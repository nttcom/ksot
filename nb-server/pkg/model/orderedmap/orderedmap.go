package orderedmap

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iancoleman/orderedmap"
)

type OrderedmapInterfaces interface {
	// create and update
	RecursiveSet(keys []string, value interface{}) error
	// delete
	RecursiveDelete(keys []string) error
	// convertByte
	MakeByte() ([]byte, error)
	// return orderedmap
	GetValue() *orderedmap.OrderedMap
}

type Orderedmap struct {
	Value *orderedmap.OrderedMap
}

var _ OrderedmapInterfaces = (*Orderedmap)(nil)

func New(b []byte) (*Orderedmap, error) {
	result := orderedmap.New()
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, fmt.Errorf("GetDevice: %w", err)
	}
	return &Orderedmap{Value: result}, nil
}

func returnEndString(s string) string {
	rarr := []rune(s)
	if len(rarr) == 0 {
		return ""
	}
	return string(rarr[len(rarr)-1])
}

func convertPathToKey(path string) (string, string, string, error) {
	// expected X or X[xxx=xxx]
	mapKey, listKey, listValue := "", "", ""
	if returnEndString(path) == "]" {
		equalSplits := strings.Split(path, "=")
		if len(equalSplits) != 2 || equalSplits[0] == "" || equalSplits[1] == "" {
			return mapKey, listKey, listValue, fmt.Errorf("convertPathToKeys: noexpected path %v", path)
		}

		rightAboutSplits := strings.Split(equalSplits[1], "]")
		if len(rightAboutSplits) != 2 || rightAboutSplits[0] == "" || rightAboutSplits[1] != "" {
			return mapKey, listKey, listValue, fmt.Errorf("convertPathToKeys: noexpected path %v", path)
		}
		listValue = rightAboutSplits[0]

		leftAboutSplits := strings.Split(equalSplits[0], "[")

		if len(leftAboutSplits) != 2 || leftAboutSplits[0] == "" || leftAboutSplits[1] == "" {
			return mapKey, listKey, listValue, fmt.Errorf("convertPathToKeys: noexpected path %v", path)
		}
		mapKey, listKey = leftAboutSplits[0], leftAboutSplits[1]
	} else {
		mapKey = path
	}

	return mapKey, listKey, listValue, nil
}

func recursiveSetParentElement(keys []string, omap *orderedmap.OrderedMap, value interface{}) (*orderedmap.OrderedMap, error) {
	mapKey, listKey, listValue, err := convertPathToKey(keys[0])
	if err != nil {
		return nil, fmt.Errorf("recursiveSetParentElement: %w", err)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("recursiveSetParentElement: keys is empty: %v", keys[0])
	}
	if len(keys) == 1 {
		if listKey != "" || listValue != "" {
			return nil, fmt.Errorf("recursiveSetParentElement: list is not key: %v", keys[0])
		}
		omap.Set(mapKey, value)
		return omap, nil
	}

	v, ok := omap.Get(mapKey)
	if ok {
		// list case
		if listKey != "" && listValue != "" {
			vlist, ok := v.([]interface{})
			if !ok {
				return nil, fmt.Errorf("recursiveSetParentElement: not list value: %v", v)
			}
			for i, v := range vlist {
				vmap := v.(orderedmap.OrderedMap)
				if vmap.Values()[listKey] == listValue {
					newVlistV, err := recursiveSetParentElement(keys[1:], &vmap, value)
					if err != nil {
						return nil, fmt.Errorf("recursiveSetParentElement: %w", err)
					}
					vlist[i] = *newVlistV
					omap.Set(mapKey, vlist)
					return omap, nil
				}
			}
			newVlistV := orderedmap.New()
			newVlistV.Set(listKey, listValue)
			newVlistV, err = recursiveSetParentElement(keys[1:], newVlistV, value)
			if err != nil {
				return nil, fmt.Errorf("recursiveSetParentElement: %w", err)
			}
			vlist = append(vlist, *newVlistV)
			omap.Set(mapKey, vlist)
			return omap, nil
		}
		// dafault
		vmap, ok := v.(orderedmap.OrderedMap)
		if !ok {
			return nil, fmt.Errorf("recursiveSetParentElement: not map value3: %v", vmap)
		}
		nvmap, err := recursiveSetParentElement(keys[1:], &vmap, value)
		if err != nil {
			return nil, fmt.Errorf("recursiveSetParentElement: not map value2: %v", v)
		}
		omap.Set(keys[0], *nvmap)
		return omap, nil
	}

	// list case
	if listKey != "" && listValue != "" {
		vlist := make([]interface{}, 0)
		vlistV := orderedmap.New()
		vlistV.Set(listKey, listValue)
		newVlistV, err := recursiveSetParentElement(keys[1:], vlistV, value)
		if err != nil {
			return nil, fmt.Errorf("recursiveSetParentElement: %w", err)
		}
		vlist = append(vlist, *newVlistV)
		omap.Set(mapKey, vlist)
		return omap, nil
	}
	// map case
	vmap := orderedmap.New()
	nvmap, err := recursiveSetParentElement(keys[1:], vmap, value)
	if err != nil {
		return nil, fmt.Errorf("recursiveSetParentElement: not map value1: %v", v)
	}
	omap.Set(keys[0], *nvmap)
	return omap, nil
}

func recursiveDeleteParentElement(keys []string, omap *orderedmap.OrderedMap) (*orderedmap.OrderedMap, error) {
	mapKey, listKey, listValue, err := convertPathToKey(keys[0])
	if err != nil {
		return nil, fmt.Errorf("recursiveDeleteParentElement: %w", err)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("recursiveDeleteParentElement: keys is empty: %v", keys[0])
	}
	if len(keys) == 1 {
		if listKey != "" || listValue != "" {
			return nil, fmt.Errorf("recursiveDeleteParentElement: list is not key: %v", keys[0])
		}
		omap.Delete(mapKey)
		return omap, nil
	}
	v, ok := omap.Get(mapKey)
	if ok {
		// list case
		if listKey != "" && listValue != "" {
			vlist, ok := v.([]interface{})
			if !ok {
				return nil, fmt.Errorf("recursiveDeleteParentElement: not list value: %v", v)
			}
			for i, v := range vlist {
				vmap, ok := v.(orderedmap.OrderedMap)
				if !ok {
					return nil, fmt.Errorf("recursiveDeleteParentElement: not map value: %v", v)
				}
				if vmap.Values()[listKey] == listValue {
					newVlistV, err := recursiveDeleteParentElement(keys[1:], &vmap)
					if err != nil {
						return nil, fmt.Errorf("recursiveDeleteParentElement: %w", err)
					}
					// if list value map is empty, delete list value

					if len(newVlistV.Keys()) == 0 {
						vlist = append(vlist[:i], vlist[i+1:]...)
					} else {
						vlist[i] = *newVlistV
					}

					// if list is empty, delete list
					if len(vlist) == 0 {
						omap.Delete(mapKey)
					} else {
						omap.Set(mapKey, vlist)
					}
					return omap, nil
				}
			}
			return omap, nil
		}
		// dafault
		vmap, ok := v.(orderedmap.OrderedMap)
		if !ok {
			return nil, fmt.Errorf("recursiveDeleteParentElement: not map value3: %v", vmap)
		}
		nvmap, err := recursiveDeleteParentElement(keys[1:], &vmap)
		if err != nil {
			return nil, fmt.Errorf("recursiveDeleteParentElement: not map value2: %v", v)
		}
		if len(nvmap.Keys()) == 0 {
			omap.Delete(mapKey)
		} else {
			omap.Set(mapKey, *nvmap)
		}
		return omap, nil
	}
	return omap, nil
}

func (o *Orderedmap) RecursiveSet(keys []string, value interface{}) error {
	newValue, err := recursiveSetParentElement(keys, o.Value, value)
	if err != nil {
		return fmt.Errorf("RecursiveSet: %w", err)
	}
	o.Value = newValue
	return nil
}

func (o *Orderedmap) RecursiveDelete(keys []string) error {
	newValue, err := recursiveDeleteParentElement(keys, o.Value)
	if err != nil {
		return fmt.Errorf("RecursiveDelete: %w", err)
	}
	o.Value = newValue
	return nil
}

func (o *Orderedmap) MakeByte() ([]byte, error) {
	return json.Marshal(o.Value)
}

func (o *Orderedmap) GetValue() *orderedmap.OrderedMap {
	return o.Value
}
