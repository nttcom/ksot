package api

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/nttcom/ksot/nb-server/pkg/model"
	"github.com/nttcom/ksot/nb-server/pkg/model/orderedmap"
	"github.com/nttcom/ksot/nb-server/pkg/model/pathmap"
)

type GithubApiInterface interface {
	API
	PostFilesForBytes(byteMap map[string][]byte) error
	UpdateFilesForBytes(byteMap map[string][]byte) error
	InitializeFilesForBytes(byteMap map[string][]byte) error
	GetDeviceConfigs([]string) (map[string]orderedmap.OrderedmapInterfaces, error)
	GetServices(services []string) (map[string]orderedmap.OrderedmapInterfaces, error)
	GetServiceRefs([]string) (map[string]map[string]pathmap.PathMapInterface, error)
	GetDevices(services []string) (map[string]orderedmap.OrderedmapInterfaces, error)
	GetDeviceRefs([]string) (map[string]map[string]pathmap.PathMapInterface, error)
	DeleteServices(serviceNames []string) error
	MakePathForDeviceRef(string) string
	MakePathForDeviceActual(string) string
	MakePathForDeviceSet(string) string
	MakePathForServiceInput(string) string
	MakePathForServiceOutput(string) string
}

type githubAPI struct {
	*api
}

var _ GithubApiInterface = (*githubAPI)(nil)

func NewGithubApi(url string) *githubAPI {
	return &githubAPI{
		&api{
			baseURL: url,
		},
	}
}

/*
func (ga *githubAPI) GetService(serviceName string) (map[string]interface{}, error) {
	result := make(map[string]orderedmap.OrderedmapInterfaces)
	url := fmt.Sprintf("/file?path=%v", ga.MakePathForServiceInput(serviceName))
	res, err := ga.GetRequest(url, 300)
	if err != nil {
		return nil, err
	}
	var resBody model.ServiceAllResFromGitServer
	if err := json.Unmarshal(res, &resBody); err != nil {
		return nil, err
	}
	allService := make(map[string]interface{}, 0)
	if err := json.Unmarshal([]byte(resBody.StringData), &allService); err != nil {
		return nil, err
	}
	return allService, nil
}

func (ga *githubAPI) GetService(serviceName string) (map[string]orderedmap.OrderedmapInterfaces, error) {
	result := make(map[string]orderedmap.OrderedmapInterfaces)
	for _, v := range devices {
		url := fmt.Sprintf("/file?path=%v", ga.MakePathForDeviceSet(v))
		res, err := ga.GetRequest(url, 300)
		if err != nil {
			return nil, err
		}
		var resBody model.ServiceAllResFromGitServer
		if err := json.Unmarshal(res, &resBody); err != nil {
			return nil, err
		}
		mapValue, err := orderedmap.New([]byte(resBody.StringData))
		if err != nil {
			return nil, err
		}
		result[v] = mapValue
	}
	return result, nil
}
*/

func (ga *githubAPI) UpdateFilesForBytes(byteMap map[string][]byte) error {
	for fp, bv := range byteMap {
		reqToGitServer := model.ServiceReqToGitServer{
			Path:       fp,
			StringData: string(bv),
		}
		reqToGitServerByte, err := json.Marshal(reqToGitServer)
		if err != nil {
			return err
		}
		_, err = ga.PostRequestAddOption("/file", "application/json", "update", reqToGitServerByte, 300)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ga *githubAPI) InitializeFilesForBytes(byteMap map[string][]byte) error {
	for fp, bv := range byteMap {
		reqToGitServer := model.ServiceReqToGitServer{
			Path:       fp,
			StringData: string(bv),
		}
		reqToGitServerByte, err := json.Marshal(reqToGitServer)
		if err != nil {
			return err
		}
		_, err = ga.PostRequestAddOption("/file", "application/json", "new_safe", reqToGitServerByte, 300)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ga *githubAPI) PostFilesForBytes(byteMap map[string][]byte) error {
	for fp, bv := range byteMap {
		reqToGitServer := model.ServiceReqToGitServer{
			Path:       fp,
			StringData: string(bv),
		}
		reqToGitServerByte, err := json.Marshal(reqToGitServer)
		if err != nil {
			return err
		}
		_, err = ga.PostRequestAddOption("/file", "application/json", "new", reqToGitServerByte, 300)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ga *githubAPI) GetDeviceConfigs(devices []string) (map[string]orderedmap.OrderedmapInterfaces, error) {
	result := make(map[string]orderedmap.OrderedmapInterfaces)
	for _, v := range devices {
		url := fmt.Sprintf("/file?path=%v", ga.MakePathForDeviceSet(v))
		res, err := ga.GetRequest(url, 300)
		if err != nil {
			return nil, err
		}
		var resBody model.ServiceAllResFromGitServer
		if err := json.Unmarshal(res, &resBody); err != nil {
			return nil, err
		}
		mapValue, err := orderedmap.New([]byte(resBody.StringData))
		if err != nil {
			return nil, err
		}
		result[v] = mapValue
	}
	return result, nil
}

func (ga *githubAPI) GetDeviceRefs(devices []string) (map[string]map[string]pathmap.PathMapInterface, error) {
	result := make(map[string]map[string]pathmap.PathMapInterface)
	for _, deviceName := range devices {
		result[deviceName] = make(map[string]pathmap.PathMapInterface)
		url := fmt.Sprintf("/file?path=%v", ga.MakePathForDeviceRef(deviceName))
		res, err := ga.GetRequest(url, 300)
		if err != nil {
			return nil, err
		}
		var resBody model.ServiceAllResFromGitServer
		if err := json.Unmarshal(res, &resBody); err != nil {
			return nil, err
		}
		mapValue := make(map[string]any)
		err = json.Unmarshal([]byte(resBody.StringData), &mapValue)
		if err != nil {
			return nil, err
		}
		for serviceName, pmv := range mapValue {
			mpi, ok := pmv.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("noexpected testdata")
			}
			vpathmap, err := pathmap.NewPathMap(mpi)
			if err != nil {
				return nil, err
			}
			result[deviceName][serviceName] = vpathmap
		}
	}
	return result, nil
}

func (ga *githubAPI) GetServices(services []string) (map[string]orderedmap.OrderedmapInterfaces, error) {
	result := make(map[string]orderedmap.OrderedmapInterfaces)
	for _, v := range services {
		url := fmt.Sprintf("/file?path=%v", ga.MakePathForServiceInput(v))
		res, err := ga.GetRequest(url, 300)
		if err != nil {
			return nil, err
		}
		var resBody model.ServiceAllResFromGitServer
		if err := json.Unmarshal(res, &resBody); err != nil {
			return nil, err
		}
		mapValue, err := orderedmap.New([]byte(resBody.StringData))
		if err != nil {
			return nil, err
		}
		result[v] = mapValue
	}
	return result, nil
}

func (ga *githubAPI) GetServiceRefs(services []string) (map[string]map[string]pathmap.PathMapInterface, error) {
	result := make(map[string]map[string]pathmap.PathMapInterface)
	for _, serviceName := range services {
		result[serviceName] = make(map[string]pathmap.PathMapInterface)
		url := fmt.Sprintf("/file?path=%v", ga.MakePathForServiceOutput(serviceName))
		res, err := ga.GetRequest(url, 300)
		if err != nil {
			return nil, err
		}
		var resBody model.ServiceAllResFromGitServer
		if err := json.Unmarshal(res, &resBody); err != nil {
			return nil, err
		}
		mapValue := make(map[string]any)
		err = json.Unmarshal([]byte(resBody.StringData), &mapValue)
		if err != nil {
			return nil, err
		}
		for deviceName, pmv := range mapValue {
			mpi, ok := pmv.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("noexpected testdata")
			}
			vpathmap, err := pathmap.NewPathMap(mpi)
			if err != nil {
				return nil, err
			}
			result[serviceName][deviceName] = vpathmap
		}
	}
	return result, nil
}

func (ga *githubAPI) GetDevices(devices []string) (map[string]orderedmap.OrderedmapInterfaces, error) {
	result := make(map[string]orderedmap.OrderedmapInterfaces)
	for _, v := range devices {
		url := fmt.Sprintf("/file?path=%v", ga.MakePathForDeviceSet(v))
		res, err := ga.GetRequest(url, 300)
		if err != nil {
			return nil, err
		}
		var resBody model.ServiceAllResFromGitServer
		if err := json.Unmarshal(res, &resBody); err != nil {
			return nil, err
		}
		mapValue, err := orderedmap.New([]byte(resBody.StringData))
		if err != nil {
			return nil, err
		}
		result[v] = mapValue
	}
	return result, nil
}

func (ga *githubAPI) DeleteServices(serviceNames []string) error {
	deleteQuery := ""
	if len(serviceNames) == 1 {
		deleteQuery += "?path=" + fmt.Sprintf("/Services/%v", serviceNames[0])
	} else {
		for i, v := range serviceNames {
			deleteQuery += "?path=" + fmt.Sprintf("/Services/%v", v)
			if i != len(serviceNames)-1 {
				deleteQuery += "&"
			}
		}
	}
	_, err := ga.DeleteRequest("/file"+deleteQuery, 300)
	if err != nil {
		return err
	}
	return nil
}

func (ga *githubAPI) MakePathForDeviceRef(name string) string {
	return filepath.Clean(fmt.Sprintf("/Devices/%v/ref.json", name))
}
func (ga *githubAPI) MakePathForDeviceSet(name string) string {
	return filepath.Clean(fmt.Sprintf("/Devices/%v/set.json", name))
}
func (ga *githubAPI) MakePathForServiceInput(name string) string {
	return filepath.Clean(fmt.Sprintf("/Services/%v/input.json", name))
}
func (ga *githubAPI) MakePathForServiceOutput(name string) string {
	return filepath.Clean(fmt.Sprintf("/Services/%v/output.json", name))
}
func (ga *githubAPI) MakePathForDeviceActual(name string) string {
	return filepath.Clean(fmt.Sprintf("/Devices/%v/actual.json", name))
}
