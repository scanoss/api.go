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
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestScanEngineWithTelemetry(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	myConfig := setupConfig(t)
	myConfig.App.Trace = true
	myConfig.Scanning.ScanDebug = true
	myConfig.Telemetry.Enabled = true
	apiService := NewAPIService(myConfig)

	tests := []struct {
		name    string
		binary  string
		wantErr bool
	}{
		{
			name:    "Test Engine - invalid binary",
			binary:  ".scan-binary-does-not-exist.sh",
			wantErr: true,
		},
		{
			name:    "Test Engine - valid binary",
			binary:  "../../test-support/scanoss.sh",
			wantErr: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			myConfig.Scanning.ScanBinary = test.binary
			err := apiService.TestEngine()
			assert.Equal(t, test.wantErr, err != nil)
		})
	}
}

func TestScanEngineNoTelemetry(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	myConfig := setupConfig(t)
	myConfig.App.Trace = true
	myConfig.Scanning.ScanDebug = true
	myConfig.Telemetry.Enabled = false
	apiService := NewAPIService(myConfig)

	tests := []struct {
		name    string
		binary  string
		wantErr bool
	}{
		{
			name:    "Test Engine - invalid binary",
			binary:  ".scan-binary-does-not-exist.sh",
			wantErr: true,
		},
		{
			name:    "Test Engine - valid binary",
			binary:  "../../test-support/scanoss.sh",
			wantErr: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			myConfig.Scanning.ScanBinary = test.binary
			err := apiService.TestEngine()
			assert.Equal(t, test.wantErr, err != nil)
		})
	}
}

func TestScanDirectSingle(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	myConfig := setupConfig(t)
	myConfig.App.Trace = true
	myConfig.Scanning.ScanDebug = true
	myConfig.Scanning.MatchConfigAllowed = false
	myConfig.Scanning.RankingAllowed = false
	apiService := NewAPIService(myConfig)

	tests := []struct {
		name            string
		fieldName       string
		file            string
		binary          string
		telemetry       bool
		scanType        string
		assets          string
		scanSettingsB64 string
		settingsAllowed bool
		want            int
	}{
		{
			name:      "Scanning - wrong name",
			binary:    "../../test-support/scanoss.sh",
			telemetry: true,
			fieldName: "wrong-name",
			file:      "./tests/fingers-empty.wfp",
			want:      http.StatusBadRequest,
		},
		{
			name:      "Scanning - empty file",
			binary:    "../../test-support/scanoss.sh",
			telemetry: true,
			fieldName: "file",
			file:      "./tests/fingers-empty.wfp",
			want:      http.StatusBadRequest,
		},
		{
			name:      "Scanning - invalid content",
			binary:    "../../test-support/scanoss.sh",
			telemetry: false,
			fieldName: "file",
			file:      "./tests/fingers-invalid.wfp",
			want:      http.StatusBadRequest,
		},
		{
			name:      "Scanning - invalid binary",
			binary:    ".scan-binary-does-not-exist.sh",
			telemetry: false,
			fieldName: "file",
			file:      "./tests/fingers.wfp",
			want:      http.StatusInternalServerError,
		},
		{
			name:      "Scanning - invalid scan type",
			binary:    "../../test-support/scanoss.sh",
			telemetry: false,
			fieldName: "file",
			file:      "./tests/fingers.wfp",
			scanType:  "does-not-exist",
			assets:    "pkg:github/org/repo",
			want:      http.StatusBadRequest,
		},
		{
			name:      "Scanning - success 1",
			binary:    "../../test-support/scanoss.sh",
			telemetry: false,
			fieldName: "file",
			file:      "./tests/fingers.wfp",
			scanType:  "identify",
			assets:    "pkg:github/org/repo",
			want:      http.StatusOK,
		},
		{
			name:      "Scanning - success 2",
			binary:    "../../test-support/scanoss.sh",
			telemetry: true,
			fieldName: "filename",
			file:      "./tests/fingers.wfp",
			scanType:  "blacklist",
			assets:    "pkg:github/org/repo",
			want:      http.StatusOK,
		},
		{
			name:            "Scanning - Settings - invalid base64",
			binary:          "../../test-support/scanoss.sh",
			fieldName:       "filename",
			file:            "./tests/fingers.wfp",
			settingsAllowed: true,
			scanSettingsB64: "invalid-base64!!!",
			want:            http.StatusBadRequest,
		},
		{
			name:      "Scanning - Settings - invalid json",
			binary:    "../../test-support/scanoss.sh",
			fieldName: "filename",
			file:      "./tests/fingers.wfp",
			// Base64 decoded JSON:
			// {
			//   "field": "something,
			//   "array": [
			// }
			scanSettingsB64: "ewoiZmllbGQiOiAic29tZXRoaW5nLAogImFycmF5IjogWwp9",
			settingsAllowed: true,
			want:            http.StatusBadRequest,
		},
		{
			name:      "Scanning - Settings - not allowed",
			binary:    "../../test-support/scanoss.sh",
			fieldName: "filename",
			file:      "./tests/fingers.wfp",
			// Base64 decoded JSON:
			// {
			//   "min_snippet_hits": 5,
			//   "min_snippet_lines": 10
			// }
			scanSettingsB64: "eyJtaW5fc25pcHBldF9oaXRzIjo1LCJtaW5fc25pcHBldF9saW5lcyI6MTB9",
			settingsAllowed: false,
			want:            http.StatusBadRequest,
		},
		{
			name:      "Scanning - Settings - success 1",
			binary:    "../../test-support/scanoss.sh",
			fieldName: "filename",
			file:      "./tests/fingers.wfp",
			// Base64 decoded JSON:
			// {
			//   "ranking_enabled": true,
			//   "ranking_threshold": 85,
			//   "min_snippet_hits": 3,
			//   "min_snippet_lines": 8,
			//   "honour_file_exts": false
			// }
			scanSettingsB64: "eyJyYW5raW5nX2VuYWJsZWQiOnRydWUsInJhbmtpbmdfdGhyZXNob2xkIjo4NSwibWluX3NuaXBwZXRfaGl0cyI6MywibWluX3NuaXBwZXRfbGluZXMiOjgsImhvbm91cl9maWxlX2V4dHMiOmZhbHNlfQ==",
			settingsAllowed: true,
			want:            http.StatusOK,
		},
		{
			name:      "Scanning - Settings - success 2",
			binary:    "../../test-support/scanoss.sh",
			fieldName: "filename",
			file:      "./tests/fingers.wfp",
			// Base64 decoded JSON:
			// {
			//   "ranking_enabled": true,
			//   "ranking_threshold": -1,
			//   "min_snippet_hits": 3,
			//   "min_snippet_lines": 8,
			//   "honour_file_exts": true
			// }
			scanSettingsB64: "ewogICJyYW5raW5nX2VuYWJsZWQiOiB0cnVlLAogICJyYW5raW5nX3RocmVzaG9sZCI6IC0xLAogICJtaW5fc25pcHBldF9oaXRzIjogMywKICAibWluX3NuaXBwZXRfbGluZXMiOiA4LAogICJob25vdXJfZmlsZV9leHRzIjogdHJ1ZQp9",
			settingsAllowed: true,
			want:            http.StatusOK,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.want == http.StatusOK { // flip between Trace for some successful calls
				if myConfig.App.Trace {
					myConfig.App.Trace = false
				} else {
					myConfig.App.Trace = true
				}
			}
			myConfig.Scanning.MatchConfigAllowed = test.settingsAllowed
			myConfig.Scanning.RankingEnabled = test.settingsAllowed
			myConfig.Scanning.ScanBinary = test.binary
			myConfig.Telemetry.Enabled = test.telemetry
			filePath := test.file
			fieldName := test.fieldName
			postBody := new(bytes.Buffer)
			mw := multipart.NewWriter(postBody)
			file, err := os.Open(filePath)
			if err != nil {
				t.Fatal(err)
			}
			writer, err := mw.CreateFormFile(fieldName, filePath)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = io.Copy(writer, file); err != nil {
				t.Fatal(err)
			}
			if len(test.scanType) > 0 && len(test.assets) > 0 {
				fmt.Println("Adding asset options: ", test.scanType, test.assets)
				err = mw.WriteField("type", test.scanType)
				if err != nil {
					t.Fatal(err)
				}
				err = mw.WriteField("assets", test.assets)
				if err != nil {
					t.Fatal(err)
				}
			}
			_ = mw.Close() // close the writer before making the request
			req := httptest.NewRequest(http.MethodPost, "http://localhost/scan/direct", postBody)
			w := httptest.NewRecorder()
			if len(test.scanSettingsB64) > 0 {
				req.Header.Set("Scanoss-Settings", test.scanSettingsB64)
			}
			req.Header.Add("Content-Type", mw.FormDataContentType())
			apiService.ScanDirect(w, req)
			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("an error was not expected when reading from request: %v", err)
			}
			assert.Equal(t, test.want, resp.StatusCode)
			fmt.Println("Status: ", resp.StatusCode)
			fmt.Println("Type: ", resp.Header.Get("Content-Type"))
			fmt.Println("Body: ", string(body))
		})
	}
}

func TestScanDirectThreaded(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	myConfig := setupConfig(t)
	myConfig.App.Trace = true
	myConfig.Scanning.ScanDebug = true
	myConfig.Scanning.Workers = 2
	myConfig.Scanning.WfpGrouping = 2
	apiService := NewAPIService(myConfig)

	tests := []struct {
		name      string
		fieldName string
		file      string
		binary    string
		want      int
	}{
		{
			name:      "Scanning - wrong name",
			binary:    "../../test-support/scanoss.sh",
			fieldName: "wrong-name",
			file:      "./tests/fingers-empty.wfp",
			want:      http.StatusBadRequest,
		},
		{
			name:      "Scanning - empty file",
			binary:    "../../test-support/scanoss.sh",
			fieldName: "file",
			file:      "./tests/fingers-empty.wfp",
			want:      http.StatusBadRequest,
		},
		{
			name:      "Scanning - invalid binary",
			binary:    ".scan-binary-does-not-exist.sh",
			fieldName: "file",
			file:      "./tests/fingers.wfp",
			want:      http.StatusInternalServerError,
		},
		{
			name:      "Scanning - success",
			binary:    "../../test-support/scanoss.sh",
			fieldName: "file",
			file:      "./tests/fingers.wfp",
			want:      http.StatusOK,
		},
		{
			name:      "Scanning - success 2",
			binary:    "../../test-support/scanoss.sh",
			fieldName: "file",
			file:      "./tests/fingers.wfp",
			want:      http.StatusOK,
		},
		{
			name:      "Scanning - success 3",
			binary:    "../../test-support/scanoss.sh",
			fieldName: "file",
			file:      "./tests/single-finger.wfp",
			want:      http.StatusOK,
		},
		{
			name:      "Scanning - HPSM success",
			binary:    "../../test-support/scanoss.sh",
			fieldName: "file",
			file:      "./tests/fingers-hpsm.wfp",
			want:      http.StatusOK,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.want == http.StatusOK { // flip between Trace for some successful calls
				if myConfig.App.Trace {
					myConfig.App.Trace = false
				} else {
					myConfig.App.Trace = true
				}
			}
			myConfig.Scanning.ScanBinary = test.binary
			filePath := test.file
			fieldName := test.fieldName
			postBody := new(bytes.Buffer)
			mw := multipart.NewWriter(postBody)
			file, err := os.Open(filePath)
			if err != nil {
				t.Fatal(err)
			}
			writer, err := mw.CreateFormFile(fieldName, filePath)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = io.Copy(writer, file); err != nil {
				t.Fatal(err)
			}
			_ = mw.Close() // close the writer before making the request

			req := httptest.NewRequest(http.MethodPost, "http://localhost/scan/direct", postBody)
			w := httptest.NewRecorder()
			req.Header.Add("Content-Type", mw.FormDataContentType())
			apiService.ScanDirect(w, req)
			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("an error was not expected when reading from request: %v", err)
			}
			assert.Equal(t, test.want, resp.StatusCode)
			fmt.Println("Status: ", resp.StatusCode)
			fmt.Println("Type: ", resp.Header.Get("Content-Type"))
			fmt.Println("Body: ", string(body))
		})
	}
}

func TestScanDirectSingleHPSM(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	myConfig := setupConfig(t)
	myConfig.App.Trace = true
	myConfig.Scanning.ScanDebug = true
	myConfig.Scanning.HPSMEnabled = false
	apiService := NewAPIService(myConfig)

	tests := []struct {
		name      string
		fieldName string
		file      string
		binary    string
		scanType  string
		assets    string
		want      int
	}{
		{
			name:      "Scanning - success 1",
			binary:    "../../test-support/scanoss.sh",
			fieldName: "file",
			file:      "./tests/fingers.wfp",
			want:      http.StatusOK,
		},
		{
			name:      "Scanning - HPSM fail",
			binary:    "../../test-support/scanoss.sh",
			fieldName: "filename",
			file:      "./tests/fingers-hpsm.wfp",
			want:      http.StatusForbidden,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.want == http.StatusOK { // flip between Trace for some successful calls
				if myConfig.App.Trace {
					myConfig.App.Trace = false
				} else {
					myConfig.App.Trace = true
				}
			} else {
				myConfig.App.Trace = true
			}
			myConfig.Scanning.ScanBinary = test.binary
			filePath := test.file
			fieldName := test.fieldName
			postBody := new(bytes.Buffer)
			mw := multipart.NewWriter(postBody)
			file, err := os.Open(filePath)
			if err != nil {
				t.Fatal(err)
			}
			writer, err := mw.CreateFormFile(fieldName, filePath)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = io.Copy(writer, file); err != nil {
				t.Fatal(err)
			}
			if len(test.scanType) > 0 && len(test.assets) > 0 {
				fmt.Println("Adding asset options: ", test.scanType, test.assets)
				err = mw.WriteField("type", test.scanType)
				if err != nil {
					t.Fatal(err)
				}
				err = mw.WriteField("assets", test.assets)
				if err != nil {
					t.Fatal(err)
				}
			}
			_ = mw.Close() // close the writer before making the request

			req := httptest.NewRequest(http.MethodPost, "http://localhost/scan/direct", postBody)
			w := httptest.NewRecorder()
			req.Header.Add("Content-Type", mw.FormDataContentType())
			apiService.ScanDirect(w, req)
			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("an error was not expected when reading from request: %v", err)
			}
			assert.Equal(t, test.want, resp.StatusCode)
			fmt.Println("Status: ", resp.StatusCode)
			fmt.Println("Type: ", resp.Header.Get("Content-Type"))
			fmt.Println("Body: ", string(body))
		})
	}
}

func TestScanDirectSingleSlow(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	myConfig := setupConfig(t)
	myConfig.App.Trace = true
	myConfig.Scanning.ScanDebug = true
	myConfig.Scanning.ScanTimeout = 5
	apiService := NewAPIService(myConfig)

	tests := []struct {
		name      string
		fieldName string
		file      string
		binary    string
		scanType  string
		assets    string
		want      int
	}{
		{
			name:      "Scanning - success 1",
			binary:    "../../test-support/scanoss.sh",
			fieldName: "file",
			file:      "./tests/fingers.wfp",
			want:      http.StatusOK,
		},
		{
			name:      "Scanning - Slow fail",
			binary:    "../../test-support/scanoss-slow.sh",
			fieldName: "filename",
			file:      "./tests/fingers-hpsm.wfp",
			want:      http.StatusGatewayTimeout,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			myConfig.Scanning.ScanBinary = test.binary
			filePath := test.file
			fieldName := test.fieldName
			postBody := new(bytes.Buffer)
			mw := multipart.NewWriter(postBody)
			file, err := os.Open(filePath)
			if err != nil {
				t.Fatal(err)
			}
			writer, err := mw.CreateFormFile(fieldName, filePath)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = io.Copy(writer, file); err != nil {
				t.Fatal(err)
			}
			_ = mw.Close() // close the writer before making the request

			req := httptest.NewRequest(http.MethodPost, "http://localhost/scan/direct", postBody)
			w := httptest.NewRecorder()
			req.Header.Add("Content-Type", mw.FormDataContentType())
			apiService.ScanDirect(w, req)
			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("an error was not expected when reading from request: %v", err)
			}
			assert.Equal(t, test.want, resp.StatusCode)
			fmt.Println("Status: ", resp.StatusCode)
			fmt.Println("Type: ", resp.Header.Get("Content-Type"))
			fmt.Println("Body: ", string(body))
		})
	}
}

func TestScanDirectSingleFlags(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	myConfig := setupConfig(t)
	myConfig.App.Trace = true
	myConfig.Scanning.ScanDebug = true
	myConfig.Scanning.ScanTimeout = 5
	apiService := NewAPIService(myConfig)
	fieldName := "file"
	filePath := "./tests/fingers.wfp"
	binary := "../../test-support/scanoss.sh"

	tests := []struct {
		name               string
		serverFlags        int
		allowFlagsOverride bool
		clientFlags        string
		want               int
	}{
		{
			name:               "Scanning - no flags",
			serverFlags:        0,
			allowFlagsOverride: false,
			clientFlags:        "",
			want:               http.StatusOK,
		},
		{
			name:               "Scanning - client flags",
			serverFlags:        0,
			allowFlagsOverride: false,
			clientFlags:        "256",
			want:               http.StatusOK,
		},
		{
			name:               "Scanning - server flags",
			serverFlags:        1248,
			allowFlagsOverride: false,
			clientFlags:        "",
			want:               http.StatusOK,
		},
		{
			name:               "Scanning - server/clients flags - allowed",
			serverFlags:        1248,
			allowFlagsOverride: true,
			clientFlags:        "256",
			want:               http.StatusOK,
		},
		{
			name:               "Scanning - server/clients flags - not allowed",
			serverFlags:        1248,
			allowFlagsOverride: false,
			clientFlags:        "256",
			want:               http.StatusOK,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			myConfig.Scanning.ScanFlags = test.serverFlags
			myConfig.Scanning.AllowFlagsOverride = test.allowFlagsOverride
			myConfig.Scanning.ScanBinary = binary
			postBody := new(bytes.Buffer)
			mw := multipart.NewWriter(postBody)
			file, err := os.Open(filePath)
			if err != nil {
				t.Fatal(err)
			}
			writer, err := mw.CreateFormFile(fieldName, filePath)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = io.Copy(writer, file); err != nil {
				t.Fatal(err)
			}
			_ = mw.Close() // close the writer before making the request

			req := httptest.NewRequest(http.MethodPost, "http://localhost/scan/direct", postBody)
			w := httptest.NewRecorder()
			req.Header.Add("Content-Type", mw.FormDataContentType())
			if len(test.clientFlags) > 0 {
				req.Header.Add("flags", test.clientFlags)
			}
			apiService.ScanDirect(w, req)
			resp := w.Result()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("an error was not expected when reading from request: %v", err)
			}
			assert.Equal(t, test.want, resp.StatusCode)
			fmt.Println("Status: ", resp.StatusCode)
			fmt.Println("Type: ", resp.Header.Get("Content-Type"))
			fmt.Println("Body: ", string(body))
		})
	}
}
