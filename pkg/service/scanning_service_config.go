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
	"encoding/json"
	"strconv"

	"go.uber.org/zap"
	cfg "scanoss.com/go-api/pkg/config"
)

type ScanningServiceConfig struct {
	Flags            int    `json:"scan_flags"` // Additional flags to pass to the scanoss binary
	SbomType         string `json:"sbom_type"`  // SBOM type to generate (spdx-json, cyclonedx-json, etc)
	SbomFile         string `json:"sbom_file"`  // SBOM output file name
	DbName           string `json:"db_name"`    // Database name to use
	RankingAllowed   bool   `json:"ranking_allowed"`
	RankingEnabled   bool   `json:"ranking_enabled"`
	RankingThreshold int    `json:"ranking_threshold"`
	MinSnippetHits   int    `json:"min_snippet_hits"`
	MinSnippetLines  int    `json:"min_snippet_lines"`
	HonourFileExts   bool   `json:"honour_file_exts"`
}

func DefaultScanningServiceConfig(serverDefaultConfig *cfg.ServerConfig) ScanningServiceConfig {
	return ScanningServiceConfig{
		Flags:            serverDefaultConfig.Scanning.ScanFlags,
		SbomType:         "",
		SbomFile:         "",
		DbName:           serverDefaultConfig.Scanning.ScanKbName,
		RankingAllowed:   serverDefaultConfig.Scanning.RankingAllowed,
		RankingEnabled:   serverDefaultConfig.Scanning.RankingEnabled,
		RankingThreshold: serverDefaultConfig.Scanning.RankingThreshold,
		MinSnippetHits:   serverDefaultConfig.Scanning.MinSnippetHits,
		MinSnippetLines:  serverDefaultConfig.Scanning.MinSnippetLines,
		HonourFileExts:   serverDefaultConfig.Scanning.HonourFileExts,
	}
}

func UpdateScanningServiceConfigDTO(s *zap.SugaredLogger, currentConfig *ScanningServiceConfig,
	flags, scanType, sbom, dbName string, inputSettings []byte) ScanningServiceConfig {
	// ScanSettings represents the scanning parameters that can be configured
	type scanSettings struct {
		RankingEnabled   *bool `json:"ranking_enabled,omitempty"`
		RankingThreshold *int  `json:"ranking_threshold,omitempty"`
		MinSnippetHits   *int  `json:"min_snippet_hits,omitempty"`
		MinSnippetLines  *int  `json:"min_snippet_lines,omitempty"`
		HonourFileExts   *bool `json:"honour_file_exts,omitempty"`
	}

	// Parse scan settings from JSON if provided
	var newSettings scanSettings
	if len(inputSettings) > 0 {
		err := json.Unmarshal(inputSettings, &newSettings)
		if err != nil {
			s.Errorf("Error unmarshalling scanning service config input: %v", err)
			return *currentConfig
		}
	}

	if newSettings.RankingEnabled != nil && currentConfig.RankingAllowed {
		currentConfig.RankingEnabled = *newSettings.RankingEnabled
		s.Debugf("Updated RankingEnabled to %v", currentConfig.RankingEnabled)
	}

	if newSettings.RankingThreshold != nil && currentConfig.RankingAllowed {
		currentConfig.RankingThreshold = *newSettings.RankingThreshold
		s.Debugf("Updated RankingThreshold to %d", currentConfig.RankingThreshold)
	}

	if newSettings.MinSnippetHits != nil {
		currentConfig.MinSnippetHits = *newSettings.MinSnippetHits
		s.Debugf("Updated MinSnippetHits to %d", currentConfig.MinSnippetHits)
	}

	if newSettings.MinSnippetLines != nil {
		currentConfig.MinSnippetLines = *newSettings.MinSnippetLines
		s.Debugf("Updated MinSnippetLines to %d", currentConfig.MinSnippetLines)
	}

	if newSettings.HonourFileExts != nil {
		currentConfig.HonourFileExts = *newSettings.HonourFileExts
		s.Debugf("Updated HonourFileExts to %v", currentConfig.HonourFileExts)
	}

	if len(dbName) > 0 && dbName != "" {
		currentConfig.DbName = dbName
		s.Debugf("Updated DbName to %s", currentConfig.DbName)
	}

	if len(flags) > 0 && flags != "" {
		flagsInt, err := strconv.Atoi(flags)
		if err != nil {
			s.Errorf("Error converting flags to integer: %v", err)
		} else {
			currentConfig.Flags = flagsInt
			s.Debugf("Updated Flags to %d", currentConfig.Flags)
		}
	}

	if len(scanType) > 0 && scanType != "" {
		currentConfig.SbomType = scanType
		s.Debugf("Updated SbomType to %s", currentConfig.SbomType)
	}

	if len(sbom) > 0 && sbom != "" {
		currentConfig.SbomFile = sbom
		s.Debugf("Updated SbomFile to %s", currentConfig.SbomFile)
	}

	return *currentConfig
}
