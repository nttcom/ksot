package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	iomap "github.com/iancoleman/orderedmap"
	"github.com/labstack/echo"
	"github.com/nttcom/ksot/nb-server/pkg/api"
	"github.com/nttcom/ksot/nb-server/pkg/composite"
	"github.com/nttcom/ksot/nb-server/pkg/config"
	"github.com/nttcom/ksot/nb-server/pkg/configurator"
	"github.com/nttcom/ksot/nb-server/pkg/diff"
	"github.com/nttcom/ksot/nb-server/pkg/editor"
	"github.com/nttcom/ksot/nb-server/pkg/model"
	"github.com/nttcom/ksot/nb-server/pkg/model/orderedmap"
	"github.com/nttcom/ksot/nb-server/pkg/model/pathmap"
	"github.com/nttcom/ksot/nb-server/pkg/sync"
	"github.com/nttcom/ksot/nb-server/pkg/tf"
	"github.com/nttcom/ksot/nb-server/pkg/util/libyang"
	"golang.org/x/exp/maps"
)

const (
	SERVICE_REPO     = "/Services/"
	PATHMAOP_REPO    = "/PathMap/"
	DEVICES_REPO     = "/Devices/"
	SERVICE_REPO_ALL = "/Services/all.json"
	PATHMAP_REPO_ALL = "/Devices/deviceToPathmap.json"
	JSON_EXTENSION   = ".json"
)

type handler struct {
	githubAPI api.GithubApiInterface
	sbAPI     api.SbApiInterface
	libyang   libyang.LibyangInterface
	tfLogic   model.PathMapLogic
}

func NewHandler(cfg config.Config) *handler {
	return &handler{
		githubAPI: api.NewGithubApi(config.Cfg.GithubServerURL),
		sbAPI:     api.NewSbApi(config.Cfg.SbServerURL),
		libyang:   libyang.New(cfg.YangFolderPath, cfg.TemporaryFilePathForLibyang+".xml", cfg.TemporaryFilePathForLibyang+".json"),
		tfLogic:   tf.TfLogic,
	}
}

func (h *handler) GetService(c echo.Context) error {
	filePath := c.Param("service")
	res, err := h.githubAPI.GetServices([]string{filePath})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("GetServices: %v", err))
	}
	return c.JSON(http.StatusOK, res[filePath].GetValue().Values())
}

func (h *handler) GetDevice(c echo.Context) error {
	filePath := c.Param("device")
	res, err := h.githubAPI.GetDevices([]string{filePath})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("GetDevices: %v", err))
	}
	return c.JSON(http.StatusOK, res[filePath].GetValue().Values())
}

func initializeServiceDatas(ga api.GithubApiInterface, serviceNames []string) error {
	postFiles := make(map[string][]byte)
	for _, v := range serviceNames {
		postFiles[api.Github.MakePathForServiceInput(v)] = []byte("{}")
		postFiles[api.Github.MakePathForServiceOutput(v)] = []byte("{}")
	}
	if err := ga.PostFilesForBytes(postFiles); err != nil {
		return fmt.Errorf("initializeServiceDatas: %w", err)
	}
	return nil
}

func (h *handler) runTfLogic(serviceMap *orderedmap.Orderedmap, serviceDevicePathmap map[string]map[string]pathmap.PathMapInterface, updateDevices map[string]bool, updateFiles map[string][]byte) error {
	fmt.Println("runTfLogic check: ", maps.Keys(h.tfLogic), serviceMap)
	for serviceName, serviceValue := range serviceMap.Value.Values() {
		serviceValueMap, ok := serviceValue.(iomap.OrderedMap)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runTfLogic: Noexpected service model format: %v", serviceValue))
		}
		serviceValueMapByte, err := json.Marshal(serviceValueMap)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runTfLogic: %v", err))
		}
		chekcServiceValidate, err := h.libyang.ValidateJsonForYang(serviceName, serviceValueMapByte)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runTfLogic: %v", err))
		}
		if !chekcServiceValidate {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runTfLogic: failed validate service %v", string(serviceValueMapByte)))
		}

		updateFiles[h.githubAPI.MakePathForServiceInput(serviceName)] = serviceValueMapByte
		deviceToPathmap, err := h.tfLogic[serviceName](serviceValue)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runTfLogic: %v", err))
		}
		for deviceName := range deviceToPathmap {
			updateDevices[deviceName] = true
		}
		serviceDevicePathmap[serviceName] = deviceToPathmap
	}

	for serviceName, deviceToPathmap := range serviceDevicePathmap {
		outputValue := make(map[string]any)
		for deviceName, pathmapValue := range deviceToPathmap {
			outputValue[deviceName] = pathmapValue.GetMapInterface()
		}
		refValueByte, err := json.Marshal(outputValue)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runTfLogic: %v", err))
		}
		updateFiles[h.githubAPI.MakePathForServiceOutput(serviceName)] = refValueByte
	}

	oldServiceRfs, err := h.githubAPI.GetServiceRefs(serviceMap.Value.Keys())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runTfLogic: %v", err))
	}
	for serviceName, deviceToPathmap := range oldServiceRfs {
		for deviceName := range deviceToPathmap {
			_, ok := serviceDevicePathmap[serviceName][deviceName]
			if !ok {
				deleteValue, err := pathmap.NewPathMap(make(map[string]interface{}))
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runTfLogic: %v", err))
				}
				serviceDevicePathmap[serviceName][deviceName] = deleteValue
				updateDevices[deviceName] = true
			}
		}
	}
	return nil
}

func (h *handler) runConfigurator(deviceNames []string, serviceDevicePathmap map[string]map[string]pathmap.PathMapInterface, updateFiles map[string][]byte) error {
	deviceIfs, err := h.sbAPI.GetDeviceInfos()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runConfigurator: %v", err))
	}
	deviceConfigs, err := h.githubAPI.GetDeviceConfigs(deviceNames)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("GetDeviceConfigs: %v", err))
	}
	rollbackConfigs, err := h.githubAPI.GetDeviceConfigs(deviceNames)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("GetDeviceConfigs: %v", err))
	}
	oldDeviceRfs, err := h.githubAPI.GetDeviceRefs(deviceNames)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("GetDeviceRefs: %v", err))
	}
	compositeInterface := composite.NewCompositeInterface()
	oldPathmaps, err := compositeInterface.CompositePathmaps(oldDeviceRfs)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("CompositePathmaps: %v", err))
	}
	newDeviceRef, err := compositeInterface.UpdateDeviceRefForComposite(serviceDevicePathmap, oldDeviceRfs)
	for deviceName, serviceToPathmap := range newDeviceRef {
		refValue := make(map[string]any)
		for serviceName, pathmapValue := range serviceToPathmap {
			if len(pathmapValue.GetMapInterface()) != 0 {
				refValue[serviceName] = pathmapValue.GetMapInterface()
			}
		}
		refValueByte, err := json.Marshal(refValue)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runConfigurator: %v", err))
		}
		updateFiles[h.githubAPI.MakePathForDeviceRef(deviceName)] = refValueByte
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runConfigurator: %v", err))
	}
	newPathmaps, err := compositeInterface.CompositePathmaps(newDeviceRef)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateServices: TfLogic: %v", err))
	}
	diffInterface := diff.NewDiffInterface()
	diffResult, err := diffInterface.DiffPathmaps(oldPathmaps, newPathmaps)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateServices: TfLogic: %v", err))
	}
	editorInterface := editor.NewEditorInterface()
	err = editorInterface.EditConfigByPathmapDiff(deviceConfigs, diffResult)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateServices: TfLogic: %v", err))
	}
	setBytes := make(map[string][]byte)
	rollbackBytes := make(map[string][]byte)
	for k, v := range deviceConfigs {
		setByte, err := v.MakeByte()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateServices: TfLogic: %v", err))
		}
		updateFiles[h.githubAPI.MakePathForDeviceSet(k)] = setByte
		chekcJson, setxmlbyte, err := h.libyang.ValidateAndConvertJSONToXML(k, setByte)
		if !chekcJson || err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateServices: no sync device %v, %v, %v", k, err, chekcJson))
		}
		setBytes[k] = setxmlbyte
		rollbackConfig, ok := rollbackConfigs[k]
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateServices: no sync device %v", k))
		}
		rollbackByte, err := rollbackConfig.MakeByte()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateServices: TfLogic: %v", err))
		}
		chekcJson, roolBackxmlbyte, err := h.libyang.ValidateAndConvertJSONToXML(k, rollbackByte)
		if !chekcJson || err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateServices: no sync device %v, %v, %v", k, err, chekcJson))
		}
		rollbackBytes[k] = roolBackxmlbyte
	}
	configuratorInterface := configurator.NewConfiguratorInterface(h.sbAPI)
	if err := configuratorInterface.Configure(deviceIfs, setBytes, rollbackBytes); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateServices: TfLogic: %v", err))
	}
	return nil
}

func (h *handler) CreateServices(c echo.Context) error {
	reqByte, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("CreateServices: %v", err))
	}
	reqServices, err := orderedmap.New(reqByte)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("CreateServices: %v", err))
	}
	if err := initializeServiceDatas(h.githubAPI, reqServices.Value.Keys()); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("initializeServiceDatas: %v", err))
	}
	updateFiles := make(map[string][]byte)
	updateDevices := make(map[string]bool, 0)
	tfLogicResult := make(map[string]map[string]pathmap.PathMapInterface)
	if err := h.runTfLogic(reqServices, tfLogicResult, updateDevices, updateFiles); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runTfLogicsByServiceMap: %v", err))
	}
	if err := h.runConfigurator(maps.Keys(updateDevices), tfLogicResult, updateFiles); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runConfigurator: TfLogic: %v", err))
	}
	if err := h.githubAPI.UpdateFilesForBytes(updateFiles); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateFilesForBytes: TfLogic: %v", err))
	}
	response := maps.Keys(updateDevices)
	sort.Slice(response, func(i, j int) bool {
		return response[i] < response[j]
	})
	return c.JSON(http.StatusOK, response)
}

func (h *handler) UpdateServices(c echo.Context) error {
	reqByte, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateServices: %v", err))
	}
	reqServices, err := orderedmap.New(reqByte)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateServices: %v", err))
	}
	updateFiles := make(map[string][]byte)
	updateDevices := make(map[string]bool, 0)
	tfLogicResult := make(map[string]map[string]pathmap.PathMapInterface)
	if err := h.runTfLogic(reqServices, tfLogicResult, updateDevices, updateFiles); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runTfLogic: %v", err))
	}
	if err := h.runConfigurator(maps.Keys(updateDevices), tfLogicResult, updateFiles); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runConfigurator: %v", err))
	}
	if err := h.githubAPI.UpdateFilesForBytes(updateFiles); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateFilesForBytes: %v", err))
	}
	response := maps.Keys(updateDevices)
	sort.Slice(response, func(i, j int) bool {
		return response[i] < response[j]
	})
	return c.JSON(http.StatusOK, response)
}

func (h *handler) DeleteServices(c echo.Context) error {
	values := c.QueryParams()
	deleteServiceNames := values["name"]
	deleteServicesReq, _ := orderedmap.New([]byte("{}"))
	for _, v := range deleteServiceNames {
		deleteServicesReq.Value.Set(v, *iomap.New())
	}
	updateFiles := make(map[string][]byte)
	updateDevices := make(map[string]bool, 0)
	tfLogicResult := make(map[string]map[string]pathmap.PathMapInterface)
	if err := h.runTfLogic(deleteServicesReq, tfLogicResult, updateDevices, updateFiles); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runTfLogic: %v", err))
	}
	if err := h.runConfigurator(maps.Keys(updateDevices), tfLogicResult, updateFiles); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("runConfigurator: %v", err))
	}
	if err := h.githubAPI.UpdateFilesForBytes(updateFiles); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateFilesForBytes: %v", err))
	}
	if err := h.githubAPI.DeleteServices(deleteServiceNames); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("UpdateFilesForBytes: %v", err))
	}
	response := maps.Keys(updateDevices)
	sort.Slice(response, func(i, j int) bool {
		return response[i] < response[j]
	})
	return c.JSON(http.StatusOK, response)
}

func (h *handler) SyncDevices(c echo.Context) error {
	updateFiles := make(map[string][]byte)
	initializeFiles := make(map[string][]byte)
	deviceInfos, err := h.sbAPI.GetDeviceInfos()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("SyncDevices: %v", err))
	}
	successList := make([]string, 0)
	for deviceName, iface := range deviceInfos {
		jsonByte, err := sync.SyncInterfaceMap[iface].SyncDevice(h.sbAPI, h.libyang, deviceName)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("SyncDevices: fail sync device %v: %v: success devices: %v", deviceName, err, successList))
		}
		updateFiles[h.githubAPI.MakePathForDeviceActual(deviceName)] = jsonByte
		// TODO diffを実装する場合、ここで差分を確認したい
		updateFiles[h.githubAPI.MakePathForDeviceSet(deviceName)] = jsonByte
		initializeFiles[h.githubAPI.MakePathForDeviceRef(deviceName)] = []byte("{}")
		successList = append(successList, deviceName)
	}

	err = h.githubAPI.UpdateFilesForBytes(updateFiles)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("SyncDevices: UpdateFilesForBytes: %v", err))
	}
	err = h.githubAPI.InitializeFilesForBytes(initializeFiles)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("SyncDevices: InitializeFilesForBytes %v", err))
	}

	sort.Slice(successList, func(i, j int) bool {
		return successList[i] < successList[j]
	})
	return c.JSON(http.StatusOK, successList)
}
