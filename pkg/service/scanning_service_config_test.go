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

	if config.Flags != 42 {
		t.Errorf("Expected Flags to be 42, got %d", config.Flags)
	}
	if config.DbName != "test-kb" {
		t.Errorf("Expected DbName to be 'test-kb', got '%s'", config.DbName)
	}
	if !config.RankingAllowed {
		t.Error("Expected RankingAllowed to be true")
	}
	if config.RankingEnabled {
		t.Error("Expected RankingEnabled to be false")
	}
	if config.RankingThreshold != 50 {
		t.Errorf("Expected RankingThreshold to be 50, got %d", config.RankingThreshold)
	}
	if config.MinSnippetHits != 10 {
		t.Errorf("Expected MinSnippetHits to be 10, got %d", config.MinSnippetHits)
	}
	if config.MinSnippetLines != 5 {
		t.Errorf("Expected MinSnippetLines to be 5, got %d", config.MinSnippetLines)
	}
	if !config.HonourFileExts {
		t.Error("Expected HonourFileExts to be true")
	}
	if config.SbomType != "" {
		t.Errorf("Expected SbomType to be empty, got '%s'", config.SbomType)
	}
	if config.SbomFile != "" {
		t.Errorf("Expected SbomFile to be empty, got '%s'", config.SbomFile)
	}
}

// TestUpdateScanningServiceConfigDTO_EmptyInput tests that empty input doesn't change config
func TestUpdateScanningServiceConfigDTO_EmptyInput(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	baseConfig := ScanningServiceConfig{
		Flags:            42,
		DbName:           "original-db",
		RankingAllowed:   true,
		RankingEnabled:   false,
		RankingThreshold: 50,
		MinSnippetHits:   10,
		MinSnippetLines:  5,
		HonourFileExts:   true,
	}

	result := UpdateScanningServiceConfigDTO(sugar, &baseConfig, "", "", "", "", nil)

	if result.Flags != 42 {
		t.Errorf("Expected Flags to remain 42, got %d", result.Flags)
	}
	if result.DbName != "original-db" {
		t.Errorf("Expected DbName to remain 'original-db', got '%s'", result.DbName)
	}
}

// TestUpdateScanningServiceConfigDTO_JSONSettings tests parsing JSON scan settings
func TestUpdateScanningServiceConfigDTO_JSONSettings(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	baseConfig := ScanningServiceConfig{
		RankingAllowed:   true,
		RankingEnabled:   false,
		RankingThreshold: 0,
		MinSnippetHits:   0,
		MinSnippetLines:  0,
		HonourFileExts:   false,
	}

	// Create JSON input
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

	if !result.RankingEnabled {
		t.Error("Expected RankingEnabled to be true")
	}
	if result.RankingThreshold != 75 {
		t.Errorf("Expected RankingThreshold to be 75, got %d", result.RankingThreshold)
	}
	if result.MinSnippetHits != 20 {
		t.Errorf("Expected MinSnippetHits to be 20, got %d", result.MinSnippetHits)
	}
	if result.MinSnippetLines != 15 {
		t.Errorf("Expected MinSnippetLines to be 15, got %d", result.MinSnippetLines)
	}
	if !result.HonourFileExts {
		t.Error("Expected HonourFileExts to be true")
	}
}

// TestUpdateScanningServiceConfigDTO_RankingNotAllowed tests that ranking settings are ignored when not allowed
func TestUpdateScanningServiceConfigDTO_RankingNotAllowed(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	baseConfig := ScanningServiceConfig{
		RankingAllowed:   false, // Ranking not allowed
		RankingEnabled:   false,
		RankingThreshold: 0,
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
	if result.RankingEnabled {
		t.Error("Expected RankingEnabled to remain false when RankingAllowed is false")
	}
	if result.RankingThreshold != 0 {
		t.Errorf("Expected RankingThreshold to remain 0 when RankingAllowed is false, got %d", result.RankingThreshold)
	}
}

// TestUpdateScanningServiceConfigDTO_StringParameters tests updating string parameters
func TestUpdateScanningServiceConfigDTO_StringParameters(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	baseConfig := ScanningServiceConfig{
		Flags:    0,
		DbName:   "default-db",
		SbomType: "",
		SbomFile: "",
	}

	result := UpdateScanningServiceConfigDTO(sugar, &baseConfig,
		"123",         // flags
		"identify",    // scanType
		"assets.json", // sbom
		"custom-db",   // dbName
		nil)

	if result.Flags != 123 {
		t.Errorf("Expected Flags to be 123, got %d", result.Flags)
	}
	if result.DbName != "custom-db" {
		t.Errorf("Expected DbName to be 'custom-db', got '%s'", result.DbName)
	}
	if result.SbomType != "identify" {
		t.Errorf("Expected SbomType to be 'identify', got '%s'", result.SbomType)
	}
	if result.SbomFile != "assets.json" {
		t.Errorf("Expected SbomFile to be 'assets.json', got '%s'", result.SbomFile)
	}
}

// TestUpdateScanningServiceConfigDTO_InvalidFlags tests handling of invalid flags
func TestUpdateScanningServiceConfigDTO_InvalidFlags(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	baseConfig := ScanningServiceConfig{
		Flags: 42,
	}

	result := UpdateScanningServiceConfigDTO(sugar, &baseConfig,
		"not-a-number", // invalid flags
		"", "", "", nil)

	// Should remain unchanged because conversion failed
	if result.Flags != 42 {
		t.Errorf("Expected Flags to remain 42 after invalid conversion, got %d", result.Flags)
	}
}

// TestUpdateScanningServiceConfigDTO_InvalidJSON tests handling of invalid JSON
func TestUpdateScanningServiceConfigDTO_InvalidJSON(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	baseConfig := ScanningServiceConfig{
		MinSnippetHits: 10,
	}

	invalidJSON := []byte("{invalid json}")

	result := UpdateScanningServiceConfigDTO(sugar, &baseConfig, "", "", "", "", invalidJSON)

	// Should remain unchanged because JSON parsing failed
	if result.MinSnippetHits != 10 {
		t.Errorf("Expected MinSnippetHits to remain 10 after invalid JSON, got %d", result.MinSnippetHits)
	}
}

// TestUpdateScanningServiceConfigDTO_PartialUpdate tests updating only some fields
func TestUpdateScanningServiceConfigDTO_PartialUpdate(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	baseConfig := ScanningServiceConfig{
		RankingAllowed:   true,
		RankingEnabled:   false,
		RankingThreshold: 50,
		MinSnippetHits:   10,
		MinSnippetLines:  5,
		HonourFileExts:   false,
	}

	// Only update MinSnippetHits
	minSnippetHits := 25
	settings := struct {
		MinSnippetHits *int `json:"min_snippet_hits,omitempty"`
	}{
		MinSnippetHits: &minSnippetHits,
	}

	jsonBytes, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	result := UpdateScanningServiceConfigDTO(sugar, &baseConfig, "", "", "", "", jsonBytes)

	// MinSnippetHits should be updated
	if result.MinSnippetHits != 25 {
		t.Errorf("Expected MinSnippetHits to be 25, got %d", result.MinSnippetHits)
	}

	// Other fields should remain unchanged
	if result.RankingEnabled {
		t.Error("Expected RankingEnabled to remain false")
	}
	if result.RankingThreshold != 50 {
		t.Errorf("Expected RankingThreshold to remain 50, got %d", result.RankingThreshold)
	}
	if result.MinSnippetLines != 5 {
		t.Errorf("Expected MinSnippetLines to remain 5, got %d", result.MinSnippetLines)
	}
	if result.HonourFileExts {
		t.Error("Expected HonourFileExts to remain false")
	}
}

// TestUpdateScanningServiceConfigDTO_CombinedUpdate tests updating both JSON and string parameters
func TestUpdateScanningServiceConfigDTO_CombinedUpdate(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	baseConfig := ScanningServiceConfig{
		Flags:            0,
		DbName:           "default-db",
		RankingAllowed:   true,
		RankingEnabled:   false,
		RankingThreshold: 0,
		MinSnippetHits:   0,
	}

	// JSON settings
	rankingEnabled := true
	rankingThreshold := 80
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

	result := UpdateScanningServiceConfigDTO(sugar, &baseConfig,
		"256",        // flags
		"blacklist",  // scanType
		"",           // sbom
		"prod-db",    // dbName
		jsonBytes)

	// Check JSON settings were applied
	if !result.RankingEnabled {
		t.Error("Expected RankingEnabled to be true")
	}
	if result.RankingThreshold != 80 {
		t.Errorf("Expected RankingThreshold to be 80, got %d", result.RankingThreshold)
	}

	// Check string parameters were applied
	if result.Flags != 256 {
		t.Errorf("Expected Flags to be 256, got %d", result.Flags)
	}
	if result.DbName != "prod-db" {
		t.Errorf("Expected DbName to be 'prod-db', got '%s'", result.DbName)
	}
	if result.SbomType != "blacklist" {
		t.Errorf("Expected SbomType to be 'blacklist', got '%s'", result.SbomType)
	}
}

// TestUpdateScanningServiceConfigDTO_ZeroValues tests that zero values can be set
func TestUpdateScanningServiceConfigDTO_ZeroValues(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	baseConfig := ScanningServiceConfig{
		RankingAllowed:   true,
		RankingEnabled:   true,
		RankingThreshold: 50,
		MinSnippetHits:   10,
		MinSnippetLines:  5,
		HonourFileExts:   true,
	}

	// Set values to zero/false
	rankingEnabled := false
	rankingThreshold := 0
	minSnippetHits := 0
	minSnippetLines := 0
	honourFileExts := false

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

	// All values should be updated to zero/false
	if result.RankingEnabled {
		t.Error("Expected RankingEnabled to be false")
	}
	if result.RankingThreshold != 0 {
		t.Errorf("Expected RankingThreshold to be 0, got %d", result.RankingThreshold)
	}
	if result.MinSnippetHits != 0 {
		t.Errorf("Expected MinSnippetHits to be 0, got %d", result.MinSnippetHits)
	}
	if result.MinSnippetLines != 0 {
		t.Errorf("Expected MinSnippetLines to be 0, got %d", result.MinSnippetLines)
	}
	if result.HonourFileExts {
		t.Error("Expected HonourFileExts to be false")
	}
}
