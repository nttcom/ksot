package model

type ServiceReqToGitServer struct {
	Path       string `json:"path"`
	StringData string `json:"string_data"`
}

type ServiceAllResFromGitServer struct {
	StringData string `json:"string_data"`
}
