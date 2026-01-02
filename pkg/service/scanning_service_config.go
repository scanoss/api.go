// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2025 SCANOSS.COM
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
	flags                 int
	sbomType              string
	sbomFile              string
	dbName                string
	rankingAllowed        bool
	rankingEnabled        bool
	rankingThreshold      int
	minSnippetHits        int
	minSnippetLines       int
	snippetRangeTolerance int
	honourFileExts        bool
}

func DefaultScanningServiceConfig(serverDefaultConfig *cfg.ServerConfig) ScanningServiceConfig {
	return ScanningServiceConfig{
		flags:                 serverDefaultConfig.Scanning.ScanFlags,
		sbomType:              "",
		sbomFile:              "",
		dbName:                serverDefaultConfig.Scanning.ScanKbName,
		rankingAllowed:        serverDefaultConfig.Scanning.RankingAllowed,
		rankingEnabled:        serverDefaultConfig.Scanning.RankingEnabled,
		rankingThreshold:      serverDefaultConfig.Scanning.RankingThreshold,
		minSnippetHits:        serverDefaultConfig.Scanning.MinSnippetHits,
		minSnippetLines:       serverDefaultConfig.Scanning.MinSnippetLines,
		snippetRangeTolerance: serverDefaultConfig.Scanning.SnippetRangeTol,
		honourFileExts:        serverDefaultConfig.Scanning.HonourFileExts,
	}
}

func UpdateScanningServiceConfigDTO(s *zap.SugaredLogger, currentConfig *ScanningServiceConfig,
	flags, scanType, sbom, dbName string, inputSettings []byte) ScanningServiceConfig {
	// ScanSettings represents the scanning parameters that can be configured
	type scanSettings struct {
		RankingEnabled        *bool `json:"ranking_enabled,omitempty"`
		RankingThreshold      *int  `json:"ranking_threshold,omitempty"`
		MinSnippetHits        *int  `json:"min_snippet_hits,omitempty"`
		MinSnippetLines       *int  `json:"min_snippet_lines,omitempty"`
		SnippetRangeTolerance *int  `json:"snippet_range_tolerance,omitempty"`
		HonourFileExts        *bool `json:"honour_file_exts,omitempty"`
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

	if newSettings.RankingEnabled != nil {
		if currentConfig.rankingAllowed {
			currentConfig.rankingEnabled = *newSettings.RankingEnabled
			s.Debugf("Updated RankingEnabled to %v", currentConfig.rankingEnabled)
		} else {
			s.Warnf("RankingEnabled setting ignored as RankingAllowed is false")
		}
	}

	if newSettings.RankingThreshold != nil {
		if currentConfig.rankingAllowed {
			currentConfig.rankingThreshold = *newSettings.RankingThreshold
			s.Debugf("Updated RankingThreshold to %d", currentConfig.rankingThreshold)
		} else {
			s.Warnf("RankingThreshold setting ignored as RankingAllowed is false")
		}
	}

	if newSettings.MinSnippetHits != nil {
		currentConfig.minSnippetHits = *newSettings.MinSnippetHits
		s.Debugf("Updated MinSnippetHits to %d", currentConfig.minSnippetHits)
	}

	if newSettings.MinSnippetLines != nil {
		currentConfig.minSnippetLines = *newSettings.MinSnippetLines
		s.Debugf("Updated MinSnippetLines to %d", currentConfig.minSnippetLines)
	}

	if newSettings.SnippetRangeTolerance != nil {
		currentConfig.snippetRangeTolerance = *newSettings.SnippetRangeTolerance
		s.Debugf("Updated SnippetRangeTol to %d", currentConfig.snippetRangeTolerance)
	}

	if newSettings.HonourFileExts != nil {
		currentConfig.honourFileExts = *newSettings.HonourFileExts
		s.Debugf("Updated HonourFileExts to %v", currentConfig.honourFileExts)
	}

	if dbName != "" {
		currentConfig.dbName = dbName
		s.Debugf("Updated DbName to %s", currentConfig.dbName)
	}

	if flags != "" {
		flagsInt, err := strconv.Atoi(flags)
		if err != nil {
			s.Errorf("Error converting flags to integer: %v", err)
		} else {
			currentConfig.flags = flagsInt
			s.Debugf("Updated Flags to %d", currentConfig.flags)
		}
	}

	if scanType != "" {
		currentConfig.sbomType = scanType
		s.Debugf("Updated SbomType to %s", currentConfig.sbomType)
	}

	if sbom != "" {
		currentConfig.sbomFile = sbom
		s.Debugf("Updated SbomFile to %s", currentConfig.sbomFile)
	}

	return *currentConfig
}
