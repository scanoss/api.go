// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2023 SCANOSS.COM
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

// Package config contains all the logic required load configuration for the Server.
package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/sdk/trace"

	"github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
)

const (
	defaultGrpcPort = "5443"
)

// ServerConfig is configuration for Server.
type ServerConfig struct {
	App struct {
		Name  string `env:"APP_NAME"`
		Port  string `env:"APP_PORT"`  // port to listen for incoming REST requests
		Addr  string `env:"APP_ADDR"`  // host to list for request on
		Debug bool   `env:"APP_DEBUG"` // true/false
		Trace bool   `env:"APP_TRACE"` // true/false
		Mode  string `env:"APP_MODE"`  // dev or prod
	}
	Logging struct {
		DynamicLogging bool     `env:"LOG_DYNAMIC"`      // true/false
		DynamicPort    string   `env:"LOG_DYNAMIC_PORT"` // host:port
		ConfigFile     string   `env:"LOG_JSON_CONFIG"`  // Json logging config file
		OutputPaths    []string `env:"LOG_OUTPUT_PATHS"` // List of outputs for logging (default stderr)
	}
	Telemetry struct {
		Enabled      bool   `env:"OTEL_ENABLED"`       // true/false
		ExtraMetrics bool   `env:"OTEL_EXTRA"`         // true/false
		OltpExporter string `env:"OTEL_EXPORTER_OLTP"` // OTEL OLTP exporter (default 0.0.0.0:4317)
	}
	Scanning struct {
		WfpLoc         string `env:"SCAN_WFP_TMP"`          // specific location to write temporary WFP files to
		ScanBinary     string `env:"SCAN_BINARY"`           // Binary to use for scanning
		ScanDebug      bool   `env:"SCAN_DEBUG"`            // true/false
		ScanFlags      int    `env:"SCAN_ENGINE_FLAGS"`     // Default flags to use when scanning
		ScanTimeout    int    `env:"SCAN_ENGINE_TIMEOUT"`   // timeout for waiting for the scan engine to respond
		WfpGrouping    int    `env:"SCAN_WFP_GROUPING"`     // number of WFP to group into a single scan engine command
		Workers        int    `env:"SCAN_WORKERS"`          // Number of concurrent workers to use per scan request
		TmpFileDelete  bool   `env:"SCAN_TMP_DELETE"`       // true/false
		KeepFailedWfps bool   `env:"SCAN_KEEP_FAILED_WFP"`  // true/false
		ScanningURL    string `env:"SCANOSS_API_URL"`       // URL to present back in API responses - default https://osskb.org/api
		HPSMEnabled    bool   `env:"SCAN_HPSM_ENABLED"`     // Enable HPSM (High Precision Snippet Matching) or not (default true)
		FileContents   bool   `env:"SCANOSS_FILE_CONTENTS"` // Show matched file URL in scan results (default true)
	}
	TLS struct {
		CertFile string `env:"SCAN_TLS_CERT"`   // TLS Certificate
		KeyFile  string `env:"SCAN_TLS_KEY"`    // Private TLS Key
		Password string `env:"SCAN_TLS_PASSWD"` // TLS Decryption Password
	}
	Filtering struct {
		AllowListFile  string `env:"SCAN_ALLOW_LIST"`       // Allow list file for incoming connections
		DenyListFile   string `env:"SCAN_DENY_LIST"`        // Deny list file for incoming connections
		BlockByDefault bool   `env:"SCAN_BLOCK_BY_DEFAULT"` // Block request by default if they are not in the allow list
		TrustProxy     bool   `env:"SCAN_TRUST_PROXY"`      // Trust the interim proxy or not (causes the source IP to be validated instead of the proxy)
	}
}

// NewServerConfig loads all config options and return a struct for use.
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

// setServerConfigDefaults attempts to set reasonable defaults for the server config.
func setServerConfigDefaults(cfg *ServerConfig) {
	cfg.App.Name = "scanoss-api-server"
	cfg.App.Port = defaultGrpcPort
	cfg.App.Mode = "dev"
	cfg.Logging.DynamicPort = "localhost:60085"
	cfg.Scanning.ScanBinary = "scanoss"
	cfg.Scanning.ScanFlags = 0
	cfg.Scanning.TmpFileDelete = true
	cfg.Scanning.Workers = 1       // Default to single threaded scanning
	cfg.Scanning.ScanTimeout = 120 // Default scan engine timeout to 2 minutes
	cfg.Scanning.WfpGrouping = 3   // Default number of WFPs to group into a single scan request (when Workers > 1)
	cfg.Scanning.HPSMEnabled = true
	cfg.Telemetry.Enabled = false
	cfg.Telemetry.ExtraMetrics = true           // Default to sending all the extra service metrics
	cfg.Telemetry.OltpExporter = "0.0.0.0:4317" // Default OTEL OLTP gRPC Exporter endpoint
	cfg.Scanning.FileContents = true            // Matched File URL response enabled (true) by default
}

// LoadFile loads the specified file and returns its contents in a string array.
func LoadFile(filename string) ([]string, error) {
	if len(filename) == 0 {
		return nil, fmt.Errorf("no file supplied to load")
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v - %v", filename, err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)
	var list []string
	for fileScanner.Scan() {
		line := strings.TrimSpace(fileScanner.Text())
		if len(line) > 0 && !strings.HasPrefix(line, "#") {
			list = append(list, line)
		}
	}
	return list, nil
}

// GetTraceSampler determines what level of trace sampling to run.
func GetTraceSampler(cfg *ServerConfig) trace.Sampler {
	switch cfg.App.Mode {
	case "dev":
		return trace.AlwaysSample()
	case "prod":
		return trace.ParentBased(trace.TraceIDRatioBased(0.5))
	default:
		return trace.AlwaysSample()
	}
}
