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

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/hashicorp/go-version"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"go.uber.org/zap"
)

// Structure for parsing KB & Engine version from scan response.
type matchStructure []struct {
	Server struct {
		Hostname  string `json:"hostname"`
		KbVersion struct {
			Daily   string `json:"daily"`
			Monthly string `json:"monthly"`
		} `json:"kb_version"`
		Version string `json:"version"`
	} `json:"server"`
}

var kbDetails string     // KB Details JSON string
var engineVersion string // Version of the engine in use

// validateEngineVersion validates that the current engine version meets the minimum requirement.
// Logs a critical error if the version is below minimum, or an info message if it meets the requirement.
func validateEngineVersion(zs *zap.SugaredLogger, currentEngineVersion, minEngineVersion string) {
	if minEngineVersion == "" || currentEngineVersion == "unknown" || currentEngineVersion == "" {
		return
	}
	currentVersion, err := version.NewVersion(currentEngineVersion)
	if err != nil {
		zs.Errorf("Failed to parse current engine version '%s': %v", currentEngineVersion, err)
		return
	}
	minVersion, err := version.NewVersion(minEngineVersion)
	if err != nil {
		zs.Errorf("Failed to parse minimum engine version '%s': %v", minEngineVersion, err)
		return
	}
	if currentVersion.LessThan(minVersion) {
		zs.Errorf("Engine version '%s' is below the minimum required version '%s'.Some features may not work as expected.", currentEngineVersion, minEngineVersion)
	} else {
		zs.Infof("Engine version '%s' meets minimum requirement '%s'", currentEngineVersion, minEngineVersion)
	}
}

// SetupKBDetailsCron sets up a background cron to update the KB version once an hour.
func (s APIService) SetupKBDetailsCron() {
	if s.config.Scanning.LoadKbDetails {
		scheduler := gocron.NewScheduler(time.UTC)
		_, err := scheduler.Every(30).Minutes().Do(s.loadKBDetails)
		if err != nil {
			zlog.S.Warnf("Problem setting up KB details cron: %v", err)
			return
		}
		scheduler.StartAsync()
	} else {
		zlog.L.Debug("KB version details not enabled. Not enabling cron.")
	}
}

// KBDetails retrieves the KB details and send back to the requester.
func (s APIService) KBDetails(w http.ResponseWriter, r *http.Request) {
	reqID := getReqID(r)
	w.Header().Set(ResponseIDKey, reqID)
	var logContext context.Context
	if s.config.Telemetry.Enabled {
		_, logContext = getSpan(r.Context(), reqID)
	} else {
		logContext = requestContext(r.Context(), reqID, "", "")
	}
	if len(kbDetails) == 0 {
		kbDetails = fmt.Sprintf(`{"kb_version": { "monthly": "%v", "daily": "%v"}}`, "unknown", "unknown")
	}
	zs := sugaredLogger(logContext) // Setup logger with context
	logRequestDetails(r, zs)
	w.Header().Set(ContentTypeKey, ApplicationJSON)
	w.WriteHeader(http.StatusOK)
	printResponse(w, fmt.Sprintf("%s\n", kbDetails), zlog.S, true)
}

// loadKBDetails attempts to scan a file to load the latest KB details from the server.
func (s APIService) loadKBDetails() {
	zs := sugaredLogger(context.TODO()) // Set up a logger without context
	zs.Debugf("Loading latest KB details...")
	if len(engineVersion) == 0 {
		engineVersion = "unknown"
	}
	// Load a random (hopefully non-existent) file match to extract the KB version details
	emptyConfig := DefaultScanningServiceConfig(s.config)
	result, err := s.scanWfp("file=7c53a2de7dfeaa20d057db98468d6670,2321,path/to/dummy/file.txt", "", emptyConfig, zs)
	if err != nil {
		zs.Warnf("Failed to detect KB version from eninge: %v", err)
		return
	}
	if len(result) > 0 {
		if !json.Valid([]byte(result)) {
			zs.Warnf("Invalid JSON response from engine for KB version: %v", result)
			return
		}
		resDataAny := map[string]interface{}{}
		err = json.Unmarshal([]byte(result), &resDataAny) // parse the response JSON into an interface map
		if err != nil {
			zs.Warnf("Failed to parse KB version from eninge response: %v - %v", result, err)
			return
		}
		if s.config.App.Trace {
			zs.Debugf("KB details JSON: %v", resDataAny)
		}
		var ms matchStructure
		// Go through the list of file results and extract one set of KB details
		for _, key := range resDataAny {
			data, err := json.Marshal(key) // convert the given interface to JSON
			if err != nil {
				zs.Warnf("Failed to convert KB version map to json: %v - %v", key, err)
				return
			}
			err = json.Unmarshal(data, &ms)
			if err != nil {
				zs.Warnf("Failed to parse KB version from eninge result: %v - %v", data, err)
				return
			}
		}
		if len(ms) > 0 {
			kbDetails = fmt.Sprintf(`{"kb_version": { "monthly": "%v", "daily": "%v"}}`, ms[0].Server.KbVersion.Monthly, ms[0].Server.KbVersion.Daily)
			engineVersion = ms[0].Server.Version
			validateEngineVersion(zs, engineVersion, minEngineVersion)
		}
	}
}
