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
	"fmt"
	"strconv"

	"go.uber.org/zap"
	cfg "scanoss.com/go-api/pkg/config"
)

type ScanningServiceConfig struct {
	flags            int
	sbomType         string
	sbomFile         string
	dbName           string
	rankingEnabled   bool
	rankingThreshold int
	minSnippetHits   int
	minSnippetLines  int
	honourFileExts   bool
}

func DefaultScanningServiceConfig(serverDefaultConfig *cfg.ServerConfig) ScanningServiceConfig {
	return ScanningServiceConfig{
		flags:            serverDefaultConfig.Scanning.ScanFlags,
		sbomType:         "",
		sbomFile:         "",
		dbName:           serverDefaultConfig.Scanning.ScanKbName,
		rankingEnabled:   serverDefaultConfig.Scanning.RankingEnabled,
		rankingThreshold: serverDefaultConfig.Scanning.RankingThreshold,
		minSnippetHits:   serverDefaultConfig.Scanning.MinSnippetHits,
		minSnippetLines:  serverDefaultConfig.Scanning.MinSnippetLines,
		honourFileExts:   serverDefaultConfig.Scanning.HonourFileExts,
	}
}

// scanSettings represents the scanning parameters that can be configured via JSON input.
type scanSettings struct {
	RankingEnabled   *bool `json:"ranking_enabled,omitempty"`
	RankingThreshold *int  `json:"ranking_threshold,omitempty"`
	MinSnippetHits   *int  `json:"min_snippet_hits,omitempty"`
	MinSnippetLines  *int  `json:"min_snippet_lines,omitempty"`
	HonourFileExts   *bool `json:"honour_file_exts,omitempty"`
}

// applyRankingSettings updates ranking-related configuration if allowed.
func (s APIService) applyRankingSettings(zs *zap.SugaredLogger, config *ScanningServiceConfig, settings *scanSettings) {
	rankingRequested := settings.RankingEnabled != nil || settings.RankingThreshold != nil
	if rankingRequested && !s.config.Scanning.RankingAllowed {
		zs.Warnf("Ranking settings ignored as RankingAllowed is false")
		return
	}
	if settings.RankingEnabled != nil {
		config.rankingEnabled = *settings.RankingEnabled
		zs.Debugf("Updated RankingEnabled to %v", config.rankingEnabled)
	}
	if settings.RankingThreshold != nil {
		config.rankingThreshold = *settings.RankingThreshold
		zs.Debugf("Updated RankingThreshold to %d", config.rankingThreshold)
	}
}

// applySnippetSettings updates snippet-related configuration and returns invalid setting names.
// Returns an error if match config settings are requested but not allowed.
func (s APIService) applySnippetSettings(zs *zap.SugaredLogger, config *ScanningServiceConfig, settings *scanSettings) ([]string, error) {
	matchConfigRequested := settings.MinSnippetHits != nil || settings.MinSnippetLines != nil || settings.HonourFileExts != nil
	if matchConfigRequested && !s.config.Scanning.MatchConfigAllowed {
		zs.Errorf("Match config settings (MinSnippetHits, MinSnippetLines, HonourFileExts) rejected as MatchConfigAllowed is false")
		return nil, fmt.Errorf("match config settings rejected: MatchConfigAllowed is disabled")
	}
	var invalidSettings []string
	if settings.MinSnippetHits != nil {
		if *settings.MinSnippetHits >= 0 {
			config.minSnippetHits = *settings.MinSnippetHits
			zs.Debugf("Updated MinSnippetHits to %d", config.minSnippetHits)
		} else {
			invalidSettings = append(invalidSettings, fmt.Sprintf("MinSnippetHits: %d", *settings.MinSnippetHits))
		}
	}
	if settings.MinSnippetLines != nil {
		if *settings.MinSnippetLines > 0 {
			config.minSnippetLines = *settings.MinSnippetLines
			zs.Debugf("Updated MinSnippetLines to %d", config.minSnippetLines)
		} else {
			invalidSettings = append(invalidSettings, fmt.Sprintf("MinSnippetLines: %d", *settings.MinSnippetLines))
		}
	}
	if settings.HonourFileExts != nil {
		config.honourFileExts = *settings.HonourFileExts
		zs.Debugf("Updated HonourFileExts to %v", config.honourFileExts)
	}
	return invalidSettings, nil
}

// applyDirectParameters updates configuration from direct string parameters.
func applyDirectParameters(zs *zap.SugaredLogger, config *ScanningServiceConfig, flags, scanType, sbom, dbName string) {
	if dbName != "" {
		config.dbName = dbName
		zs.Debugf("Updated DbName to %s", config.dbName)
	}
	if flags != "" {
		flagsInt, err := strconv.Atoi(flags)
		if err == nil {
			config.flags = flagsInt
			zs.Debugf("Updated Flags to %d", config.flags)
		} else {
			zs.Errorf("Error converting flags to integer: %v", err)
		}
	}
	if scanType != "" {
		config.sbomType = scanType
		zs.Debugf("Updated SbomType to %s", config.sbomType)
	}
	if sbom != "" {
		config.sbomFile = sbom
		zs.Debugf("Updated SbomFile to %s", config.sbomFile)
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
//     "ranking_enabled": bool,         // Enable/disable ranking (requires ranking_allowed=true)
//     "ranking_threshold": int,        // Ranking threshold value (requires ranking_allowed=true)
//     "min_snippet_hits": int,         // Minimum snippet hits to consider a match
//     "min_snippet_lines": int,        // Minimum snippet lines to consider a match
//     "honour_file_exts": bool         // Honour file extensions when filtering snippets
//     }
//
// Returns:
//   - A new ScanningServiceConfig with the updates applied. The original config remains unchanged.
//
// Note:
//   - Ranking settings (ranking_enabled, ranking_threshold) are only applied if rankingAllowed is true
//   - Invalid JSON in inputSettings will be logged and the original config will be returned
//   - Invalid flags string will be logged and that specific field will not be updated
func (s APIService) UpdateScanningServiceConfigDTO(zs *zap.SugaredLogger, currentConfig *ScanningServiceConfig,
	flags, scanType, sbom, dbName string, inputSettings []byte) (ScanningServiceConfig, error) {
	if currentConfig == nil {
		zs.Errorf("Current scanning service config is nil")
		return ScanningServiceConfig{}, fmt.Errorf("default server scanning service config is undefined")
	}
	updatedConfig := *currentConfig
	var newSettings scanSettings
	if len(inputSettings) > 0 {
		if err := json.Unmarshal(inputSettings, &newSettings); err != nil {
			zs.Errorf("Error unmarshalling scanning service config input: %v", err)
			return updatedConfig, fmt.Errorf("error unmarshalling scanning service config requested by client: %v", err)
		}
	}
	s.applyRankingSettings(zs, &updatedConfig, &newSettings)
	invalidSettings, err := s.applySnippetSettings(zs, &updatedConfig, &newSettings)
	if err != nil {
		return updatedConfig, err
	}
	if len(invalidSettings) > 0 {
		zs.Errorf("Ignoring invalid values for settings: %v", invalidSettings)
	}
	applyDirectParameters(zs, &updatedConfig, flags, scanType, sbom, dbName)
	return updatedConfig, nil
}
