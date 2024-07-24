package config

import "os"

type Config struct {
	GitRepoPath string
}

var Cfg Config

func init() {
	if gitRepoName, ok := os.LookupEnv("GITHUB_REPO_NAME"); !ok {
		Cfg.GitRepoPath = "/work/github-server/gitrepo/git-sample"
	} else {
		Cfg.GitRepoPath = "/work/github-server/gitrepo/" + gitRepoName
	}
}
