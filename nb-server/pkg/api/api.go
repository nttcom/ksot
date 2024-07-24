package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/nttcom/ksot/nb-server/pkg/config"
)

var (
	Github = &githubAPI{
		&api{
			baseURL: config.Cfg.GithubServerURL,
		},
	}
	Sb = &sbAPI{
		&api{
			baseURL: config.Cfg.SbServerURL,
		},
	}
)

type API interface {
	GetRequest(path string, timeout int) ([]byte, error)
	PostRequest(path string, contentType string, reqBody interface{}, timeout int) ([]byte, error)
	PutRequest(path string, contentType string, reqBody interface{}, timeout int) ([]byte, error)
	DeleteRequest(path string, timeout int) ([]byte, error)
	PostRequestAddOption(path string, contentType string, option string, reqBody interface{}, timeout int) ([]byte, error)
	PostFileRequest(path string, fileData []byte, timeout int) error
}

type api struct {
	baseURL string
}

var _ API = (*api)(nil)

func newhttpClient(timeout int) *http.Client {
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	return client
}

func (ap *api) GetRequest(path string, timeout int) ([]byte, error) {
	client := newhttpClient(timeout)
	url := ap.baseURL + path
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("GetRequest: endpoint=%v,  statusCode=%v", url, response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (ap *api) DeleteRequest(path string, timeout int) ([]byte, error) {
	client := newhttpClient(timeout)
	url := ap.baseURL + path
	req, err := http.NewRequestWithContext(context.Background(), http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("GetRequest: endpoint=%v,  statusCode=%v", url, response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (ap *api) PostRequest(path string, contentType string, reqBody interface{}, timeout int) ([]byte, error) {
	client := newhttpClient(timeout)
	var req *http.Request
	var err error
	url := ap.baseURL + path
	switch reqBody := reqBody.(type) {
	case *bytes.Buffer:
		req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, url, reqBody)
	case []byte:
		req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(reqBody))
	default:
		return nil, errors.New("noexpected type of requestbody")
	}
	req.Header.Set("Content-Type", contentType)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("PostRequest: endpoint=%v,  statusCode=%v", url, response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (ap *api) PostRequestAddOption(path string, contentType string, option string, reqBody interface{}, timeout int) ([]byte, error) {
	client := newhttpClient(timeout)
	var req *http.Request
	var err error
	url := ap.baseURL + path
	switch reqBody := reqBody.(type) {
	case *bytes.Buffer:
		req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, url, reqBody)
	case []byte:
		req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewBuffer(reqBody))
	default:
		return nil, errors.New("noexpected type of requestbody")
	}
	req.Header.Set("Content-Type", contentType)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-POST-OPTION", option)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("PostRequest: endpoint=%v,  statusCode=%v", url, response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (ap *api) PutRequest(path string, contentType string, reqBody interface{}, timeout int) ([]byte, error) {
	client := newhttpClient(timeout)
	var req *http.Request
	var err error
	url := ap.baseURL + path
	switch reqBody := reqBody.(type) {
	case *bytes.Buffer:
		req, err = http.NewRequestWithContext(context.Background(), http.MethodPut, url, reqBody)
	case []byte:
		req, err = http.NewRequestWithContext(context.Background(), http.MethodPut, url, bytes.NewBuffer(reqBody))
	default:
		return nil, errors.New("noexpected type of requestbody")
	}
	req.Header.Set("Content-Type", contentType)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return nil, fmt.Errorf("PutRequest: endpoint=%v,  statusCode=%v", url, response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (ap *api) PostFileRequest(path string, fileData []byte, timeout int) error {
	client := newhttpClient(timeout)
	var req *http.Request
	var err error
	url := ap.baseURL + path
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("set", "set")
	if err != nil {
		return err
	}
	_, err = io.Copy(part, bytes.NewReader(fileData))
	if err != nil {
		return err
	}
	writer.Close()

	req, err = http.NewRequestWithContext(context.Background(), http.MethodPost, url, body)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())

	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("PostFileRequest: response=%v", response)
	}
	defer response.Body.Close()
	return nil
}
