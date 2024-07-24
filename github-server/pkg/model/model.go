package model

type StringData struct {
	StringData string `json:"string_data"`
}

type ReqStringData struct {
	Path       string `json:"path"`
	StringData string `json:"string_data"`
}
