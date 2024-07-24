package libyang

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type LibyangInterface interface {
	// TODO パスマップのsync機能)
	ValidateAndConvertXMLToJSON(deviceName string, xml []byte) (bool, []byte, error)
	ValidateAndConvertJSONToXML(deviceName string, jsonFile []byte) (bool, []byte, error)
	ValidateJsonForYang(deviceName string, jsonFile []byte) (bool, error)
}

type libyang struct {
	yangFolderPath        string
	temporaryXmlFilePath  string
	temporaryJsonFilePath string
}

var _ LibyangInterface = (*libyang)(nil)

func New(yangFolderPath string, temporaryXmlFilePath string, temporaryJsonFilePath string) *libyang {
	return &libyang{yangFolderPath: yangFolderPath, temporaryXmlFilePath: temporaryXmlFilePath, temporaryJsonFilePath: temporaryJsonFilePath}
}

func searchYangFiles(yangFolderPath string, kind string, deviceName string) ([]string, error) {
	out, err := exec.Command("find", filepath.Join(yangFolderPath, filepath.Clean(fmt.Sprintf("%v/%v", kind, deviceName))), "-name", "*.yang").CombinedOutput()
	if err != nil {
		return []string{}, fmt.Errorf("inputYangFile: find yang file error: %w: %v", err, string(out))
	}
	result := strings.Split(string(out), "\n")
	sort.Slice(result[:len(result)-1], func(i, j int) bool {
		return result[i] < result[j]
	})
	return result[:len(result)-1], nil
}

func (l *libyang) ValidateAndConvertXMLToJSON(deviceName string, xml []byte) (bool, []byte, error) {
	f, err := os.Create(l.temporaryXmlFilePath)
	defer os.Remove(l.temporaryXmlFilePath)
	if err != nil {
		return false, []byte{}, fmt.Errorf("ValidateAndConvertXMLToJSON: %w", err)
	}
	_, err = f.Write(xml)
	if err != nil {
		return false, []byte{}, fmt.Errorf("ValidateAndConvertXMLToJSON: %w", err)
	}
	yangFiles, err := searchYangFiles(l.yangFolderPath, "devices", deviceName)
	if err != nil {
		return false, []byte{}, fmt.Errorf("ValidateAndConvertXMLToJSON: %w", err)
	}
	command := append([]string{"-t", "getconfig", "--quiet", "--format", "json"}, append(yangFiles, l.temporaryXmlFilePath)...)
	jsonByte, err := exec.Command("yanglint", command...).CombinedOutput()
	if err != nil {
		return false, []byte{}, fmt.Errorf("ValidateJsonForYang: yanglint error: %v", string(jsonByte))
	}
	return true, jsonByte, nil
}

func (l *libyang) ValidateAndConvertJSONToXML(deviceName string, jsonFile []byte) (bool, []byte, error) {
	f, err := os.Create(l.temporaryJsonFilePath)
	defer os.Remove(l.temporaryJsonFilePath)
	if err != nil {
		return false, []byte{}, fmt.Errorf("ValidateAndConvertXMLToJSON: %w", err)
	}
	_, err = f.Write(jsonFile)
	if err != nil {
		return false, []byte{}, fmt.Errorf("ValidateAndConvertXMLToJSON: %w", err)
	}
	yangFiles, err := searchYangFiles(l.yangFolderPath, "devices", deviceName)
	if err != nil {
		return false, []byte{}, fmt.Errorf("ValidateAndConvertXMLToJSON: %w", err)
	}
	command := append([]string{"-t", "getconfig", "--quiet", "--format", "xml"}, append(yangFiles, l.temporaryJsonFilePath)...)
	xmlByte, err := exec.Command("yanglint", command...).CombinedOutput()
	if err != nil {
		return false, []byte{}, fmt.Errorf("ValidateJsonForYang: yanglint error: %v", string(xmlByte))
	}
	return true, xmlByte, nil
}

func (l *libyang) ValidateJsonForYang(deviceName string, jsonFile []byte) (bool, error) {
	f, err := os.Create(l.temporaryJsonFilePath)
	defer os.Remove(l.temporaryJsonFilePath)
	if err != nil {
		return false, fmt.Errorf("ValidateAndConvertXMLToJSON: %w", err)
	}
	_, err = f.Write(jsonFile)
	if err != nil {
		return false, fmt.Errorf("ValidateAndConvertXMLToJSON: %w", err)
	}
	yangFiles, err := searchYangFiles(l.yangFolderPath, "services", deviceName)
	if err != nil {
		return false, fmt.Errorf("ValidateAndConvertXMLToJSON: %w", err)
	}
	command := append([]string{"-t", "config", "--quiet"}, append(yangFiles, l.temporaryJsonFilePath)...)
	out, err := exec.Command("yanglint", command...).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("ValidateJsonForYang: yanglint error: %v", string(out))
	}
	return true, nil
}
