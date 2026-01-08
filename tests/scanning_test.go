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

package tests

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type E2EScanningSuite struct {
	suite.Suite
}

func TestE2EScanningSuite(t *testing.T) {
	suite.Run(t, new(E2EScanningSuite))
}

func (s *E2EScanningSuite) TestScanning() {
	c := http.Client{}
	tests := []struct {
		name        string
		fieldName   string
		filename    string
		shortName   string
		extraFields map[string]string
		want        int
	}{
		{
			name:        "Test Invalid  KB name",
			filename:    "../pkg/service/tests/fingers.wfp",
			shortName:   "fingers.wfp",
			extraFields: map[string]string{"db_name": "test_kb"},
			want:        http.StatusOK, // Engine handles invalid KB names gracefully with fallback
		},
		{
			name:        "Test Empty WFP",
			filename:    "../pkg/service/tests/fingers-empty.wfp",
			shortName:   "fingers-empty.wfp",
			extraFields: map[string]string{},
			want:        http.StatusBadRequest,
		},
		{
			name:        "Test Invalid WFP",
			filename:    "../pkg/service/tests/fingers-invalid.wfp",
			shortName:   "fingers-invalid.wfp",
			extraFields: map[string]string{},
			want:        http.StatusBadRequest,
		},
		{
			name:        "Test Invalid Field Name",
			fieldName:   "wrong-name",
			filename:    "../pkg/service/tests/fingers.wfp",
			shortName:   "fingers.wfp",
			extraFields: map[string]string{},
			want:        http.StatusBadRequest,
		},
		{
			name:        "Test Invalid Type (flags)",
			filename:    "../pkg/service/tests/fingers.wfp",
			shortName:   "fingers.wfp",
			extraFields: map[string]string{"type": "invalid", "assets": "pkg:github/ignore/ignore"},
			want:        http.StatusBadRequest,
		},
		{
			name:        "Test Valid WFP",
			filename:    "../pkg/service/tests/fingers.wfp",
			shortName:   "fingers.wfp",
			extraFields: map[string]string{},
			want:        http.StatusOK,
		},
		{
			name:        "Test Flags - identify",
			filename:    "../pkg/service/tests/fingers.wfp",
			shortName:   "fingers.wfp",
			extraFields: map[string]string{"flags": "16", "type": "identify", "assets": `{"components":[{"purl":"pkg:github/scanoss/scanoss.py"}]}`},
			want:        http.StatusOK,
		},
		{
			name:        "Test Flags - blacklist",
			filename:    "../pkg/service/tests/fingers.wfp",
			shortName:   "fingers.wfp",
			extraFields: map[string]string{"flags": "16", "type": "blacklist", "assets": `{"components":[{"purl":"pkg:github/scanoss/scanoss.py"}]}`},
			want:        http.StatusOK,
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			fieldName := "file"
			if len(test.fieldName) > 0 {
				fieldName = test.fieldName
			}
			b, w, err := createMultipartFormData(fieldName, test.filename, test.shortName, test.extraFields)
			if err != nil {
				s.Failf("an error was not creating multipart form data.", "error: %v", err)
			}
			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%v/scan/direct", hostPort), &b)
			if err != nil {
				s.Failf("an error was not creating request.", "error: %v", err)
			}
			req.Header.Set("Content-Type", w.FormDataContentType())
			resp, err := c.Do(req)
			if err != nil {
				s.Failf("an error was not expected when sending request.", "error: %v", err)
			}

			s.Equal(test.want, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				s.Failf("an error was not expected when reading response body.", "error: %v", err)
			}
			fmt.Println("Status: ", resp.StatusCode)
			fmt.Println("Type: ", resp.Header.Get("Content-Type"))
			fmt.Println("Body: ", string(body))
		})
	}
}

func (s *E2EScanningSuite) TestScanSettingsHeader() {
	c := http.Client{}
	tests := []struct {
		name              string
		filename          string
		shortName         string
		scanSettingsB64   string
		extraFields       map[string]string
		want              int
		description       string
	}{
		{
			name:        "Test Valid ScanSettings - Multiple Settings",
			filename:    "../pkg/service/tests/fingers.wfp",
			shortName:   "fingers.wfp",
			extraFields: map[string]string{},
			want:        http.StatusOK,
			description: "Should successfully process valid scan settings with multiple parameters",
			// Base64 decoded JSON:
			// {
			//   "ranking_enabled": true,
			//   "ranking_threshold": 85,
			//   "min_snippet_hits": 3,
			//   "min_snippet_lines": 8,
			//   "honour_file_exts": false
			// }
			scanSettingsB64: "eyJyYW5raW5nX2VuYWJsZWQiOnRydWUsInJhbmtpbmdfdGhyZXNob2xkIjo4NSwibWluX3NuaXBwZXRfaGl0cyI6MywibWluX3NuaXBwZXRfbGluZXMiOjgsImhvbm91cl9maWxlX2V4dHMiOmZhbHNlfQ==",
		},
		{
			name:            "Test Invalid ScanSettings - Invalid Base64",
			filename:        "../pkg/service/tests/fingers.wfp",
			shortName:       "fingers.wfp",
			scanSettingsB64: "invalid-base64!!!", // Invalid base64 string - should return error
			extraFields:     map[string]string{},
			want:            http.StatusBadRequest, // Invalid scan settings should return error
			description:     "Should return error for invalid base64 scan settings",
		},
		{
			name:        "Test ScanSettings with Legacy Flags",
			filename:    "../pkg/service/tests/fingers.wfp",
			shortName:   "fingers.wfp",
			extraFields: map[string]string{"flags": "16"},
			want:        http.StatusOK,
			description: "Should successfully combine scan settings with legacy flags parameter",
			// Base64 decoded JSON:
			// {
			//   "min_snippet_hits": 5,
			//   "min_snippet_lines": 10
			// }
			scanSettingsB64: "eyJtaW5fc25pcHBldF9oaXRzIjo1LCJtaW5fc25pcHBldF9saW5lcyI6MTB9",
		},
	}

	for _, test := range tests {
		s.Run(test.name, func() {
			b, w, err := createMultipartFormData("file", test.filename, test.shortName, test.extraFields)
			if err != nil {
				s.Failf("an error was not creating multipart form data.", "error: %v", err)
			}
			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%v/scan/direct", hostPort), &b)
			if err != nil {
				s.Failf("an error was not creating request.", "error: %v", err)
			}
			req.Header.Set("Content-Type", w.FormDataContentType())

			// Set the scanoss-scan-settings header if provided
			if len(test.scanSettingsB64) > 0 {
				req.Header.Set("scanoss-scan-settings", test.scanSettingsB64)
			}

			resp, err := c.Do(req)
			if err != nil {
				s.Failf("an error was not expected when sending request.", "error: %v", err)
			}

			s.Equal(test.want, resp.StatusCode, test.description)
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				s.Failf("an error was not expected when reading response body.", "error: %v", err)
			}
			fmt.Println("Test: ", test.name)
			fmt.Println("Status: ", resp.StatusCode)
			fmt.Println("Type: ", resp.Header.Get("Content-Type"))
			fmt.Println("Body: ", string(body))
		})
	}
}
