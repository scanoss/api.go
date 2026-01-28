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

// Package cmd handles Scanning Service REST API launch.
package cmd

import (
	_ "embed"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	myconfig "scanoss.com/go-api/pkg/config"
	"scanoss.com/go-api/pkg/protocol/rest"
)

//go:generate bash ../../get_version.sh
//go:embed version.txt
var version string

// getConfig checks command line args for an option to feed into the config parser.
func getConfig() (*myconfig.ServerConfig, error) {
	var jsonConfig, envConfig, loggingConfig string
	flag.StringVar(&jsonConfig, "json-config", "", "Application JSON config")
	flag.StringVar(&envConfig, "env-config", "", "Application dot-ENV config")
	flag.StringVar(&loggingConfig, "logging-config", "", "Logging config file")
	debug := flag.Bool("debug", false, "Enable debug")
	ver := flag.Bool("version", false, "Display current version")
	flag.Parse()
	if *ver {
		fmt.Printf("Version: %v", version)
		os.Exit(1)
	}
	var feeders []config.Feeder
	if len(jsonConfig) > 0 {
		feeders = append(feeders, feeder.Json{Path: jsonConfig})
	}
	if len(envConfig) > 0 {
		feeders = append(feeders, feeder.DotEnv{Path: envConfig})
	}
	if *debug {
		err := os.Setenv("APP_DEBUG", "1")
		if err != nil {
			fmt.Printf("Warning: Failed to set env APP_DEBUG to 1: %v", err)
			return nil, err
		}
	}
	myConfig, err := myconfig.NewServerConfig(feeders)
	if err == nil && myConfig != nil && len(loggingConfig) > 0 {
		myConfig.Logging.ConfigFile = loggingConfig // Override any logging config file with this one.
	}
	return myConfig, err
}

// setupEnvVars configures a custom env var for the scanoss engine.
func setupEnvVars(cfg *myconfig.ServerConfig) {
	setupEnvVarFileContentsURL(cfg)
	setupEnvVarFileContents(cfg)
	setupEnvVarHPSMkey(cfg)
}

// setupEnvVarFileContentsURL sets up SCANOSS_FILE_CONTENTS_URL, used by HPSM and the engine to access file contents.
func setupEnvVarFileContentsURL(cfg *myconfig.ServerConfig) {
	if len(cfg.Scanning.ScanningURL) > 0 {
		err := os.Setenv("SCANOSS_API_URL", cfg.Scanning.ScanningURL)
		if err != nil {
			zlog.S.Infof("Failed to set alternative SCANOSS_API_URL value to %s: %v", cfg.Scanning.ScanningURL, err)
		}
	}
	var contentsURL string
	customURL := os.Getenv("SCANOSS_API_URL")
	if len(customURL) > 0 {
		zlog.S.Infof("Using custom API URL: %s", customURL)
		customURL = strings.TrimSuffix(customURL, "/")
		contentsURL = fmt.Sprintf("%s/file_contents", customURL) // Assume the contents URL from the scanning URL
	}
	if len(cfg.Scanning.FileContentsURL) > 0 {
		contentsURL = cfg.Scanning.FileContentsURL // We have an explicit contents URL specified. Use it
	}
	if len(contentsURL) > 0 {
		err := os.Setenv("SCANOSS_FILE_CONTENTS_URL", contentsURL)
		if err != nil {
			zlog.S.Infof("Failed to set SCANOSS_FILE_CONTENTS_URL value to %v: %v", contentsURL, err)
		}
	}
	if customContentsURL := os.Getenv("SCANOSS_FILE_CONTENTS_URL"); len(customContentsURL) > 0 {
		zlog.S.Infof("Using custom content URL: %s.", customContentsURL)
	}
}

// setupEnvVarFileContents sets up SCANOSS_FILE_CONTENTS for backward compatibility.
func setupEnvVarFileContents(cfg *myconfig.ServerConfig) {
	err := os.Setenv("SCANOSS_FILE_CONTENTS", fmt.Sprintf("%v", cfg.Scanning.FileContents))
	if err != nil {
		zlog.S.Infof("Failed to set SCANOSS_FILE_CONTENTS value to %v: %v", cfg.Scanning.FileContents, err)
	}
	if customContents := os.Getenv("SCANOSS_FILE_CONTENTS"); len(customContents) > 0 && customContents == "false" {
		zlog.S.Infof("Skipping file_url data field.")
		err2 := os.Setenv("SCANOSS_FILE_CONTENTS_URL", customContents) // Force the contents URL to say 'false' also
		if err2 != nil {
			zlog.S.Infof("Failed to set SCANOSS_FILE_CONTENTS_URL value to %v: %v", customContents, err2)
		}
	}
}

// setupEnvVarHPSMkey sets up SCANOSS_API_KEY used by HSPM to access file contents.
func setupEnvVarHPSMkey(cfg *myconfig.ServerConfig) {
	if cfg.Scanning.HPSMEnabled && len(cfg.Scanning.HPSMcontentsAPIkey) > 0 {
		err := os.Setenv("SCANOSS_API_KEY", cfg.Scanning.HPSMcontentsAPIkey)
		if err != nil {
			zlog.S.Infof("Failed to set SCANOSS_API_KEY value to %v: %v", "*****", err)
		}
	}
}

// RunServer runs the gRPC Dependency Server.
func RunServer() error {
	// Load command line options and config
	cfg, err := getConfig()
	if err != nil || cfg == nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	// Check mode to determine which logger to load
	{
		switch strings.ToLower(cfg.App.Mode) {
		case "prod":
			if len(cfg.Logging.ConfigFile) > 0 {
				err = zlog.NewSugaredLoggerFromFile(cfg.Logging.ConfigFile)
			} else {
				err = zlog.NewSugaredProdLogger(cfg.Logging.OutputPaths...)
			}
		default:
			if len(cfg.Logging.ConfigFile) > 0 {
				err = zlog.NewSugaredLoggerFromFile(cfg.Logging.ConfigFile)
			} else {
				err = zlog.NewSugaredDevLogger()
			}
		}
		if err != nil {
			return fmt.Errorf("failed to load logger: %v", err)
		}
		if cfg.App.Debug {
			zlog.SetLevel("debug")
		}
		zlog.L.Debug("Running with debug enabled")
		defer zlog.SyncZap()
	}
	zlog.S.Infof("Starting SCANOSS Dependency Service: %v", strings.TrimSpace(version))
	// Set up a database connection pool
	if cfg.Logging.DynamicLogging && len(cfg.Logging.DynamicPort) > 0 {
		zlog.S.Infof("Setting up dynamic logging level on %v.", cfg.Logging.DynamicPort)
		zlog.SetupDynamicLogging(cfg.Logging.DynamicPort)
		zlog.S.Infof("Use the following to get the current status: curl -X GET %v/log/level", cfg.Logging.DynamicPort)
		zlog.S.Infof("Use the following to set the current status: curl -X PUT %v/log/level -d level=debug", cfg.Logging.DynamicPort)
	}
	sc := cfg.Scanning
	if sc.HPSMcontentsAPIkey != "" {
		sc.HPSMcontentsAPIkey = "<redacted>"
	}
	zlog.S.Infof("Running with %v files and %v worker(s) per scan request", cfg.Scanning.WfpGrouping, cfg.Scanning.Workers)
	zlog.S.Infof("Running with Scan config: %+v", sc)
	// Setup custom env variables if requested
	setupEnvVars(cfg)
	if cfg.Scanning.HPSMEnabled {
		err = testHPSMSetup()
		if err != nil {
			zlog.S.Errorf("HPSM setup test failed: %v - check SCANOSS_FILE_CONTENTS_URL or SCANOSS_API_KEY are correct, disabling HPSM.", err)
			cfg.Scanning.HPSMEnabled = false
		}
	}
	return rest.RunServer(cfg, version)
}

// testHPSMSetup validates that the sources server is available to enable HPSM.
func testHPSMSetup() error {
	url := os.Getenv("SCANOSS_FILE_CONTENTS_URL")
	if url == "" {
		return fmt.Errorf("SCANOSS_FILE_CONTENTS_URL is not set")
	}
	// Ensure the URL ends with "/" before appending the test MD5
	url = strings.TrimSuffix(url, "/") + "/8109a183e06165144dc8d97b791c130f"
	zlog.S.Debug("HPSM test request started")
	// Create HTTP GET request
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create HPSM test request: %w", err)
	}
	// Set the X-Session header if an API key is present
	if apiKey := os.Getenv("SCANOSS_API_KEY"); apiKey != "" {
		req.Header.Set("X-Session", apiKey)
	}
	// Perform the request with a 10-second timeout
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if resp != nil {
		defer func(Body io.ReadCloser) {
			err2 := Body.Close()
			if err2 != nil {
				zlog.S.Errorf("Failed to close HPSM response body: %v", err2)
			}
		}(resp.Body)
	}
	if err != nil {
		return fmt.Errorf("HPSM connection test failed: %w", err)
	}
	if resp == nil {
		return fmt.Errorf("HPSM connection test failed: no response")
	}
	// Treat non-2xx status codes as failures
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP status %d", resp.StatusCode)
	}
	zlog.S.Infof("HPSM setup test successful (HTTP %d)", resp.StatusCode)
	return nil
}
