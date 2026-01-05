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

// UpdateScanningServiceConfigDTO creates an updated copy of the scanning service configuration.
//
// This function does NOT modify the original currentConfig. Instead, it creates a copy,
// applies the requested updates to the copy, and returns the updated configuration.
//
// Parameters:
//   - s: Sugared logger for debug/error output
//   - currentConfig: Pointer to the current configuration (will NOT be modified)
//   - flags: String representation of scan flags (converted to int). Empty string = no change
//   - scanType: SBOM type to use for scanning. Empty string = no change
//   - sbom: SBOM file path. Empty string = no change
//   - dbName: Database name for scanning. Empty string = no change
//   - inputSettings: JSON bytes containing optional scan settings. Format:
//     {
//       "ranking_enabled": bool,         // Enable/disable ranking (requires ranking_allowed=true)
//       "ranking_threshold": int,        // Ranking threshold value (requires ranking_allowed=true)
//       "min_snippet_hits": int,         // Minimum snippet hits to consider a match
//       "min_snippet_lines": int,        // Minimum snippet lines to consider a match
//       "snippet_range_tolerance": int,  // Snippet range tolerance for matching
//       "honour_file_exts": bool         // Honor file extensions when filtering snippets
//     }
//
// Returns:
//   - A new ScanningServiceConfig with the updates applied. The original config remains unchanged.
//
// Note:
//   - Ranking settings (ranking_enabled, ranking_threshold) are only applied if rankingAllowed is true
//   - Invalid JSON in inputSettings will be logged and the original config will be returned
//   - Invalid flags string will be logged and that specific field will not be updated
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

	// Create a copy of the current config to avoid modifying the original
	updatedConfig := *currentConfig

	// Parse scan settings from JSON if provided
	var newSettings scanSettings
	if len(inputSettings) > 0 {
		err := json.Unmarshal(inputSettings, &newSettings)
		if err != nil {
			s.Errorf("Error unmarshalling scanning service config input: %v", err)
			return updatedConfig
		}
	}

	if newSettings.RankingEnabled != nil {
		if updatedConfig.rankingAllowed {
			updatedConfig.rankingEnabled = *newSettings.RankingEnabled
			s.Debugf("Updated RankingEnabled to %v", updatedConfig.rankingEnabled)
		} else {
			s.Warnf("RankingEnabled setting ignored as RankingAllowed is false")
		}
	}

	if newSettings.RankingThreshold != nil {
		if updatedConfig.rankingAllowed {
			updatedConfig.rankingThreshold = *newSettings.RankingThreshold
			s.Debugf("Updated RankingThreshold to %d", updatedConfig.rankingThreshold)
		} else {
			s.Warnf("RankingThreshold setting ignored as RankingAllowed is false")
		}
	}

	if newSettings.MinSnippetHits != nil {
		updatedConfig.minSnippetHits = *newSettings.MinSnippetHits
		s.Debugf("Updated MinSnippetHits to %d", updatedConfig.minSnippetHits)
	}

	if newSettings.MinSnippetLines != nil {
		updatedConfig.minSnippetLines = *newSettings.MinSnippetLines
		s.Debugf("Updated MinSnippetLines to %d", updatedConfig.minSnippetLines)
	}

	if newSettings.SnippetRangeTolerance != nil {
		updatedConfig.snippetRangeTolerance = *newSettings.SnippetRangeTolerance
		s.Debugf("Updated SnippetRangeTol to %d", updatedConfig.snippetRangeTolerance)
	}

	if newSettings.HonourFileExts != nil {
		updatedConfig.honourFileExts = *newSettings.HonourFileExts
		s.Debugf("Updated HonourFileExts to %v", updatedConfig.honourFileExts)
	}

	if dbName != "" {
		updatedConfig.dbName = dbName
		s.Debugf("Updated DbName to %s", updatedConfig.dbName)
	}

	if flags != "" {
		flagsInt, err := strconv.Atoi(flags)
		if err != nil {
			s.Errorf("Error converting flags to integer: %v", err)
		} else {
			updatedConfig.flags = flagsInt
			s.Debugf("Updated Flags to %d", updatedConfig.flags)
		}
	}

	if scanType != "" {
		updatedConfig.sbomType = scanType
		s.Debugf("Updated SbomType to %s", updatedConfig.sbomType)
	}

	if sbom != "" {
		updatedConfig.sbomFile = sbom
		s.Debugf("Updated SbomFile to %s", updatedConfig.sbomFile)
	}

	return updatedConfig
}
