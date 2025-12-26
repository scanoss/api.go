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
	"testing"

	"go.uber.org/zap"
	cfg "scanoss.com/go-api/pkg/config"
)

// TestDefaultScanningServiceConfig tests that default config is created correctly from server config
func TestDefaultScanningServiceConfig(t *testing.T) {
	serverConfig := &cfg.ServerConfig{}
	serverConfig.Scanning.ScanFlags = 42
	serverConfig.Scanning.ScanKbName = "test-kb"
	serverConfig.Scanning.RankingAllowed = true
	serverConfig.Scanning.RankingEnabled = false
	serverConfig.Scanning.RankingThreshold = 50
	serverConfig.Scanning.MinSnippetHits = 10
	serverConfig.Scanning.MinSnippetLines = 5
	serverConfig.Scanning.HonourFileExts = true

	config := DefaultScanningServiceConfig(serverConfig)

	if config.flags != 42 {
		t.Errorf("Expected Flags to be 42, got %d", config.flags)
	}
	if config.dbName != "test-kb" {
		t.Errorf("Expected DbName to be 'test-kb', got '%s'", config.dbName)
	}
	if !config.rankingAllowed {
		t.Error("Expected RankingAllowed to be true")
	}
	if config.rankingEnabled {
		t.Error("Expected RankingEnabled to be false")
	}
	if config.rankingThreshold != 50 {
		t.Errorf("Expected RankingThreshold to be 50, got %d", config.rankingThreshold)
	}
	if config.minSnippetHits != 10 {
		t.Errorf("Expected MinSnippetHits to be 10, got %d", config.minSnippetHits)
	}
	if config.minSnippetLines != 5 {
		t.Errorf("Expected MinSnippetLines to be 5, got %d", config.minSnippetLines)
	}
	if !config.honourFileExts {
		t.Error("Expected HonourFileExts to be true")
	}
}

// TestUpdateScanningServiceConfigDTO_JSONSettings tests parsing JSON scan settings
func TestUpdateScanningServiceConfigDTO_JSONSettings(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	baseConfig := ScanningServiceConfig{
		rankingAllowed:   true,
		rankingEnabled:   false,
		rankingThreshold: 0,
		minSnippetHits:   0,
		minSnippetLines:  0,
		honourFileExts:   false,
	}

	// Test with multiple JSON settings
	rankingEnabled := true
	rankingThreshold := 75
	minSnippetHits := 20
	minSnippetLines := 15
	honourFileExts := true

	settings := struct {
		RankingEnabled   *bool `json:"ranking_enabled,omitempty"`
		RankingThreshold *int  `json:"ranking_threshold,omitempty"`
		MinSnippetHits   *int  `json:"min_snippet_hits,omitempty"`
		MinSnippetLines  *int  `json:"min_snippet_lines,omitempty"`
		HonourFileExts   *bool `json:"honour_file_exts,omitempty"`
	}{
		RankingEnabled:   &rankingEnabled,
		RankingThreshold: &rankingThreshold,
		MinSnippetHits:   &minSnippetHits,
		MinSnippetLines:  &minSnippetLines,
		HonourFileExts:   &honourFileExts,
	}

	jsonBytes, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	result := UpdateScanningServiceConfigDTO(sugar, &baseConfig, "", "", "", "", jsonBytes)

	if !result.rankingEnabled {
		t.Error("Expected RankingEnabled to be true")
	}
	if result.rankingThreshold != 75 {
		t.Errorf("Expected RankingThreshold to be 75, got %d", result.rankingThreshold)
	}
	if result.minSnippetHits != 20 {
		t.Errorf("Expected MinSnippetHits to be 20, got %d", result.minSnippetHits)
	}
	if result.minSnippetLines != 15 {
		t.Errorf("Expected MinSnippetLines to be 15, got %d", result.minSnippetLines)
	}
	if !result.honourFileExts {
		t.Error("Expected HonourFileExts to be true")
	}
}

// TestUpdateScanningServiceConfigDTO_RankingNotAllowed tests that ranking settings are ignored when not allowed
func TestUpdateScanningServiceConfigDTO_RankingNotAllowed(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	baseConfig := ScanningServiceConfig{
		rankingAllowed:   false, // Ranking not allowed
		rankingEnabled:   false,
		rankingThreshold: 0,
	}

	// Try to enable ranking
	rankingEnabled := true
	rankingThreshold := 75

	settings := struct {
		RankingEnabled   *bool `json:"ranking_enabled,omitempty"`
		RankingThreshold *int  `json:"ranking_threshold,omitempty"`
	}{
		RankingEnabled:   &rankingEnabled,
		RankingThreshold: &rankingThreshold,
	}

	jsonBytes, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	result := UpdateScanningServiceConfigDTO(sugar, &baseConfig, "", "", "", "", jsonBytes)

	// Should remain false because RankingAllowed is false
	if result.rankingEnabled {
		t.Error("Expected RankingEnabled to remain false when RankingAllowed is false")
	}
	if result.rankingThreshold != 0 {
		t.Errorf("Expected RankingThreshold to remain 0 when RankingAllowed is false, got %d", result.rankingThreshold)
	}
}

// TestUpdateScanningServiceConfigDTO_LegacyParameters tests updating legacy string parameters
func TestUpdateScanningServiceConfigDTO_LegacyParameters(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	baseConfig := ScanningServiceConfig{
		flags:    0,
		dbName:   "default-db",
		sbomType: "",
		sbomFile: "",
	}

	result := UpdateScanningServiceConfigDTO(sugar, &baseConfig,
		"123",         // flags
		"identify",    // scanType
		"assets.json", // sbom
		"custom-db",   // dbName
		nil)

	if result.flags != 123 {
		t.Errorf("Expected Flags to be 123, got %d", result.flags)
	}
	if result.dbName != "custom-db" {
		t.Errorf("Expected DbName to be 'custom-db', got '%s'", result.dbName)
	}
	if result.sbomType != "identify" {
		t.Errorf("Expected SbomType to be 'identify', got '%s'", result.sbomType)
	}
	if result.sbomFile != "assets.json" {
		t.Errorf("Expected SbomFile to be 'assets.json', got '%s'", result.sbomFile)
	}
}

// TestUpdateScanningServiceConfigDTO_InvalidInput tests handling of invalid input
func TestUpdateScanningServiceConfigDTO_InvalidInput(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	baseConfig := ScanningServiceConfig{
		flags:          42,
		minSnippetHits: 10,
	}

	// Test with invalid flags
	result := UpdateScanningServiceConfigDTO(sugar, &baseConfig,
		"not-a-number", "", "", "", nil)

	if result.flags != 42 {
		t.Errorf("Expected Flags to remain 42 after invalid conversion, got %d", result.flags)
	}

	// Test with invalid JSON
	invalidJSON := []byte("{invalid json}")
	result = UpdateScanningServiceConfigDTO(sugar, &baseConfig, "", "", "", "", invalidJSON)

	if result.minSnippetHits != 10 {
		t.Errorf("Expected MinSnippetHits to remain 10 after invalid JSON, got %d", result.minSnippetHits)
	}
}

// TestUpdateScanningServiceConfigDTO_CombinedUpdate tests updating both JSON and legacy parameters together
func TestUpdateScanningServiceConfigDTO_CombinedUpdate(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	baseConfig := ScanningServiceConfig{
		flags:            0,
		dbName:           "default-db",
		rankingAllowed:   true,
		rankingEnabled:   false,
		rankingThreshold: 0,
		minSnippetHits:   0,
	}

	// JSON settings
	rankingEnabled := true
	rankingThreshold := 80
	minSnippetHits := 5

	settings := struct {
		RankingEnabled   *bool `json:"ranking_enabled,omitempty"`
		RankingThreshold *int  `json:"ranking_threshold,omitempty"`
		MinSnippetHits   *int  `json:"min_snippet_hits,omitempty"`
	}{
		RankingEnabled:   &rankingEnabled,
		RankingThreshold: &rankingThreshold,
		MinSnippetHits:   &minSnippetHits,
	}

	jsonBytes, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	result := UpdateScanningServiceConfigDTO(sugar, &baseConfig,
		"256",       // flags
		"blacklist", // scanType
		"",          // sbom
		"prod-db",   // dbName
		jsonBytes)

	// Check JSON settings were applied
	if !result.rankingEnabled {
		t.Error("Expected RankingEnabled to be true")
	}
	if result.rankingThreshold != 80 {
		t.Errorf("Expected RankingThreshold to be 80, got %d", result.rankingThreshold)
	}
	if result.minSnippetHits != 5 {
		t.Errorf("Expected MinSnippetHits to be 5, got %d", result.minSnippetHits)
	}

	// Check legacy string parameters were applied
	if result.flags != 256 {
		t.Errorf("Expected Flags to be 256, got %d", result.flags)
	}
	if result.dbName != "prod-db" {
		t.Errorf("Expected DbName to be 'prod-db', got '%s'", result.dbName)
	}
	if result.sbomType != "blacklist" {
		t.Errorf("Expected SbomType to be 'blacklist', got '%s'", result.sbomType)
	}
}
