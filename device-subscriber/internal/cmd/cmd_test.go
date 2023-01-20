/*
 Copyright (c) 2022-2023 NTT Communications Corporation

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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	newValidStruct := func(t func(*Config)) *Config {
		cfg := &Config{
			Device:        "device1",
			Addr:          ":9339",
			AggregatorURL: "http://localhost:8000",
		}
		t(cfg)
		return cfg
	}

	tests := []struct {
		name      string
		transform func(cfg *Config)
		wantErr   bool
	}{
		{
			"ok",
			func(cfg *Config) {},
			false,
		},
		{
			"err: device is empty",
			func(cfg *Config) {
				cfg.Device = ""
			},
			true,
		},
		{
			"err: addr is empty",
			func(cfg *Config) {
				cfg.Addr = ""
			},
			true,
		},
		{
			"err: aggregator-url is empty",
			func(cfg *Config) {
				cfg.AggregatorURL = ""
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newValidStruct(tt.transform)
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestNewRootCmd(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			"err: device not set",
			[]string{"kuesta-subscribe", "-addr=:9339", "-aggregator-url=http://localhost:8080"},
			true,
		},
		{
			"err: addr not set",
			[]string{"kuesta-subscribe", "-d=device1", "-aggregator-url=http://localhost:8080"},
			true,
		},
		{
			"err: aggregator-url not set",
			[]string{"kuesta-subscribe", "-d=device1", "-addr=:9339"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewRootCmd()
			c.SetArgs(tt.args)
			err := c.Execute()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
