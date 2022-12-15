// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2022 SCANOSS.COM
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 2 of the License, or
 * (at your option) any later version.
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package config

import (
	"github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
)

const (
	defaultGrpcPort = "8085"
)

// ServerConfig is configuration for Server
type ServerConfig struct {
	App struct {
		Name  string `env:"APP_NAME"`
		Port  string `env:"APP_PORT"`
		Addr  string `env:"APP_ADDR"`
		Debug bool   `env:"APP_DEBUG"` // true/false
		Trace bool   `env:"APP_TRACE"` // true/false
		Mode  string `env:"APP_MODE"`  // dev or prod
	}
	Logging struct {
		DynamicLogging bool   `env:"LOG_DYNAMIC"`      // true/false
		DynamicPort    string `env:"LOG_DYNAMIC_PORT"` // host:port
		ConfigFile     string `env:"LOG_JSON_CONFIG"`
	}
	Scanning struct {
		WfpLoc        string `env:"SCAN_WFP_TMP"` // specific location to write temporary WFP files to
		ScanBinary    string `env:"SCAN_BINARY"`
		ScanDebug     bool   `env:"SCAN_DEBUG"` // true/false
		ScanFlags     int    `env:"SCAN_ENGINE_FLAGS"`
		Workers       int    `env:"SCAN_WORKERS"`
		TmpFileDelete bool   `env:"SCAN_TMP_DELETE"` // true/false
	}
	Tls struct {
		CertFile string `env:"SCAN_TLS_CERT"`
		KeyFile  string `env:"SCAN_TLS_KEY"`
	}
}

// NewServerConfig loads all config options and return a struct for use
func NewServerConfig(feeders []config.Feeder) (*ServerConfig, error) {
	cfg := ServerConfig{}
	setServerConfigDefaults(&cfg)
	c := config.New()
	for _, f := range feeders {
		c.AddFeeder(f)
	}
	c.AddFeeder(feeder.Env{})
	c.AddStruct(&cfg)
	err := c.Feed()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// setServerConfigDefaults attempts to set reasonable defaults for the server config
func setServerConfigDefaults(cfg *ServerConfig) {
	cfg.App.Name = "SCANOSS API Server"
	cfg.App.Port = defaultGrpcPort
	cfg.App.Addr = ""
	cfg.App.Mode = "dev"
	cfg.App.Debug = false
	cfg.Logging.DynamicPort = "localhost:60085"
	cfg.Scanning.WfpLoc = ""
	cfg.Scanning.ScanBinary = "scanoss"
	cfg.Scanning.ScanFlags = 0
	cfg.Scanning.ScanDebug = false
	cfg.Scanning.TmpFileDelete = true
	cfg.Scanning.Workers = 3
}
