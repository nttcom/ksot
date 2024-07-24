package model

import (
	"github.com/nttcom/ksot/nb-server/pkg/model/pathmap"
)

type JsonMap map[string]interface{}
type ServiceMap map[string]map[string]interface{}

func NewJsonMap() JsonMap {
	return make(map[string]interface{}, 0)
}

type PathMapLogic map[string]func(interface{}) (map[string]pathmap.PathMapInterface, error)

func NewPathMapLogic() PathMapLogic {
	return make(PathMapLogic, 0)
}

func NewServiceMap() ServiceMap {
	return make(map[string]map[string]interface{}, 0)
}

func (serviceMap ServiceMap) AddServiceMap(key1 string, key2 string, val interface{}) {
	serviceMap[key1] = make(map[string]interface{}, 0)
	serviceMap[key1][key2] = val
}
