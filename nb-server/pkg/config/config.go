package config

import "os"

type Config struct {
	GithubServerURL             string
	SbServerURL                 string
	YangFolderPath              string
	TemporaryFilePathForLibyang string
}

var Cfg Config

func init() {
	if githubServerURL, ok := os.LookupEnv("GITHUB_SERVER_URL"); !ok {
		Cfg.GithubServerURL = "https://github.com"
	} else {
		Cfg.GithubServerURL = githubServerURL
	}

	if sbServerURL, ok := os.LookupEnv("SB_SERVER_URL"); !ok {
		Cfg.SbServerURL = "https://sb.com"
	} else {
		Cfg.SbServerURL = sbServerURL
	}

	if yangFolderPath, ok := os.LookupEnv("YANG_FOLDER_PATH"); !ok {
		Cfg.YangFolderPath = "/work/yangFolder"
	} else {
		Cfg.YangFolderPath = yangFolderPath
	}

	if temporaryFilePathForLibyang, ok := os.LookupEnv("TEMPORARY_FILEPATH_FOR_LIBYANG"); !ok {
		Cfg.TemporaryFilePathForLibyang = "/work/tmmporaryfile"
	} else {
		Cfg.TemporaryFilePathForLibyang = temporaryFilePathForLibyang
	}
}
