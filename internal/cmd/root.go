/*
 Copyright (c) 2022 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/nttcom/kuesta/internal/core"
	"github.com/nttcom/kuesta/internal/gogit"
	"github.com/nttcom/kuesta/pkg/stacktrace"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		stacktrace.Show(os.Stderr, err)
		// NOTE add show cmd.UsageString() for the specific error if needed
		os.Exit(1)
	}
}

const (
	FlagConfig         = "config"
	FlagDevel          = "devel"
	FlagVerbose        = "verbose"
	FlagConfigRootPath = "config-root-path"
	FlagStatusRootPath = "status-root-path"
	FlagConfigRepoUrl  = "config-repo-url"
	FlagStatusRepoUrl  = "status-repo-url"
	FlagGitTrunk       = "git-trunk"
	FlagGitRemote      = "git-remote-name"
	FlagGitToken       = "git-token"
	FlagGitUser        = "git-user"
	FlagGitEmail       = "git-email"
	FlagPushToMain     = "push-to-main"
	FlagNoTLS          = "notls"
	FlagTLSCrt         = "tls-crt"
	FlagTLSKey         = "tls-key"
	FlagTLSCACrt       = "tls-ca-crt"
	FlagInsecure       = "insecure"
)

// NewRootCmd creates command root.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "kuesta",
		Short:        "kuesta controls Network Element Configurations.",
		SilenceUsage: true,
	}

	cobra.OnInitialize(initConfig)

	cmd.PersistentFlags().StringVar(&cfgFile, FlagConfig, "", "config file (default is $HOME/.kuesta.yaml)")

	cmd.PersistentFlags().Uint8P(FlagVerbose, "v", 1, "verbose level")
	cmd.PersistentFlags().BoolP(FlagDevel, "", false, "enable development mode")
	cmd.PersistentFlags().StringP(FlagConfigRootPath, "p", "", "path to the config repository root")
	cmd.PersistentFlags().StringP(FlagStatusRootPath, "", "", "path to the status repository root")
	cmd.PersistentFlags().StringP(FlagConfigRepoUrl, "r", "", "git config repository url")
	cmd.PersistentFlags().StringP(FlagStatusRepoUrl, "", "", "git status repository url")
	cmd.PersistentFlags().StringP(FlagGitTrunk, "", gogit.DefaultTrunkBranch, "git trunk branch")
	cmd.PersistentFlags().StringP(FlagGitRemote, "", gogit.DefaultRemoteName, "git remote name to be used for gitops")
	cmd.PersistentFlags().StringP(FlagGitToken, "", "", "git auth token")
	cmd.PersistentFlags().StringP(FlagGitUser, "", gogit.DefaultGitUser, "git username")
	cmd.PersistentFlags().StringP(FlagGitEmail, "", gogit.DefaultGitEmail, "git email")
	cmd.PersistentFlags().BoolP(FlagPushToMain, "", false, "push to main (otherwise create new branch)")
	cmd.PersistentFlags().BoolP(FlagNoTLS, "", false, "disable TLS validation")
	cmd.PersistentFlags().BoolP(FlagInsecure, "", false, "skip TLS validation. Client cert will be verified only when provided.")
	cmd.PersistentFlags().StringP(FlagTLSCrt, "", "", "path to the certificate file")
	cmd.PersistentFlags().StringP(FlagTLSKey, "", "", "path to the private key file")
	cmd.PersistentFlags().StringP(FlagTLSCACrt, "", "", "path to the CA certificate file")

	mustBindToViper(cmd)
	cmd.Version = getVcsRevision()

	cmd.AddCommand(newServiceCmd())
	cmd.AddCommand(newDeviceCmd())
	cmd.AddCommand(newGitCmd())
	cmd.AddCommand(newServeCmd())
	cmd.AddCommand(newCueCmd())
	cmd.AddCommand(newVersionCmd())

	return cmd
}

func newRootCfg(cmd *cobra.Command) (*core.RootCfg, error) {
	gitUser := viper.GetString(FlagGitUser)
	gitEmail := viper.GetString(FlagGitEmail)
	if gitUser != gogit.DefaultGitUser && gitEmail == gogit.DefaultGitEmail {
		gitEmail = fmt.Sprintf("%s@example.com", gitUser)
	}

	cfg := &core.RootCfg{
		Verbose:        cast.ToUint8(viper.GetUint(FlagVerbose)),
		Devel:          viper.GetBool(FlagDevel),
		ConfigRootPath: viper.GetString(FlagConfigRootPath),
		ConfigRepoUrl:  viper.GetString(FlagConfigRepoUrl),
		StatusRootPath: viper.GetString(FlagStatusRootPath),
		StatusRepoUrl:  viper.GetString(FlagStatusRepoUrl),
		GitTrunk:       viper.GetString(FlagGitTrunk),
		GitToken:       viper.GetString(FlagGitToken),
		GitRemote:      viper.GetString(FlagGitRemote),
		GitUser:        gitUser,
		GitEmail:       gitEmail,
		PushToMain:     viper.GetBool(FlagPushToMain),
	}
	return cfg, cfg.Validate()
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".kuesta" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".kuesta")
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.SetEnvPrefix("KUESTA")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
