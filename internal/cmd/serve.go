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
	"github.com/nttcom/kuesta/internal/core"
	"github.com/nttcom/kuesta/internal/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	FlagServeAddr       = "serve-addr"
	FlagSyncInterval    = "sync-interval"
	FlagPersistGitState = "persist-git-state"
)

func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run gNMI server to expose northbound API",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := newServeCfg(cmd, args)
			if err != nil {
				return err
			}
			logger.Setup(cfg.Devel, cfg.Verbose)
			serveError := make(chan error)
			go func() {
				serveError <- core.RunServeHttp(cmd.Context(), cfg)
			}()
			go func() {
				serveError <- core.RunServe(cmd.Context(), cfg)
			}()
			return <-serveError
		},
	}
	cmd.Flags().StringP(FlagServeAddr, "a", ":9339", "Bind address of gNMI northbound API.")
	cmd.Flags().IntP(FlagSyncInterval, "", 10, "Interval to exec git-pull from status repo.")
	cmd.Flags().BoolP(FlagPersistGitState, "", false, "Persist git workspace even when api call closed without performing hard-reset.")
	mustBindToViper(cmd)

	return cmd
}

func newServeCfg(cmd *cobra.Command, args []string) (*core.ServeCfg, error) {
	rootCfg, err := newRootCfg(cmd)
	if err != nil {
		return nil, err
	}
	cfg := &core.ServeCfg{
		RootCfg:         *rootCfg,
		Addr:            viper.GetString(FlagServeAddr),
		SyncPeriod:      viper.GetInt(FlagSyncInterval),
		PersistGitState: viper.GetBool(FlagPersistGitState),
		NoTLS:           viper.GetBool(FlagNoTLS),
		Insecure:        viper.GetBool(FlagInsecure),
		TLSCrtPath:      viper.GetString(FlagTLSCrt),
		TLSKeyPath:      viper.GetString(FlagTLSKey),
		TLSCACrtPath:    viper.GetString(FlagTLSCACrt),
	}
	return cfg, cfg.Validate()
}
