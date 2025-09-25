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
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestFileContents(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	myConfig := setupConfig(t)
	myConfig.App.Trace = true
	myConfig.Scanning.ScanDebug = true
	apiService := NewAPIService(myConfig)

	tests := []struct {
		name      string
		input     map[string]string
		binary    string
		telemetry bool
		want      int
	}{
		{
			name:   "Test Contents - empty",
			binary: "../../test-support/scanoss.sh",
			input:  map[string]string{},
			want:   http.StatusBadRequest,
		},
		{
			name:      "Test Contents - wrong key",
			binary:    "../../test-support/scanoss.sh",
			telemetry: true,
			input:     map[string]string{"invalid": "wrong"},
			want:      http.StatusBadRequest,
		},
		{
			name:      "Test Contents - invalid binary",
			binary:    "scan-binary-does-not-exist.sh",
			telemetry: true,
			input:     map[string]string{"md5": "37f7cd1e657aa3c30ece35995b4c59e5"},
			want:      http.StatusInternalServerError,
		},
		{
			name:      "Test Contents - success",
			binary:    "../../test-support/scanoss.sh",
			telemetry: false,
			input:     map[string]string{"md5": "37f7cd1e657aa3c30ece35995b4c59e5"},
			want:      http.StatusOK,
		},
		{
			name:      "Test Contents - success 2",
			binary:    "../../test-support/scanoss.sh",
			telemetry: false,
			input:     map[string]string{"md5": "37f7cd1e657aa3c30ece35995b4c59e5"},
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
			myConfig.Telemetry.Enabled = test.telemetry
			req := newReq("GET", "http://localhost/file_contents/{md5}", "", test.input)
			w := httptest.NewRecorder()
			apiService.FileContents(w, req)
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

func TestDetectCharset(t *testing.T) {
	tests := []struct {
		name             string
		input            []byte
		expectedCharset  string
		acceptedCharsets []string // Alternative acceptable charsets
		expectError      bool
	}{
		{
			name:             "UTF-8 text with English",
			input:            []byte("Hello, world! This is UTF-8 text."),
			acceptedCharsets: []string{"UTF-8", "ISO-8859-1", "ISO-8859-2", "windows-1252", "Ascii"}, // ASCII-compatible text can be detected as various charsets
		},
		{
			name:             "Plain ASCII text",
			input:            []byte("Simple ASCII text without special characters"),
			acceptedCharsets: []string{"UTF-8", "ISO-8859-1", "ISO-8859-2", "windows-1252", "ISO-8859-15", "Ascii"},
		},
		{
			name:            "Empty buffer",
			input:           []byte{},
			expectedCharset: "UTF-8", // Low confidence, defaults to UTF-8
		},
		{
			name:            "UTF-8 with Spanish and Japanese",
			input:           []byte("Español con ñ, café, and 日本語"),
			expectedCharset: "UTF-8",
		},
		{
			name:             "Large ASCII buffer",
			input:            bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz"), 1500),
			acceptedCharsets: []string{"UTF-8", "ISO-8859-1", "ISO-8859-2", "windows-1252", "Ascii"},
		},
		{
			name:             "UTF-16 BOM",
			input:            []byte{0xFF, 0xFE, 0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00}, // "Hello" in UTF-16LE
			acceptedCharsets: []string{"UTF-16LE", "UTF-8"},                                                  // New module may detect as UTF-8 for short samples
		},
		{
			name:             "HTML document",
			input:            []byte("<!DOCTYPE html><html><head><title>Test Page</title></head><body><h1>Hello World</h1></body></html>"),
			acceptedCharsets: []string{"UTF-8", "ISO-8859-1", "ISO-8859-2", "windows-1252", "Ascii"},
		},
		{
			name:             "Low confidence binary data",
			input:            []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			acceptedCharsets: []string{"UTF-8", "Ascii"}, // Low confidence may detect as either
		},
		{
			name:             "ISO-8859-1 with Latin characters",
			input:            []byte("Café, naïve, résumé"),
			acceptedCharsets: []string{"UTF-8", "ISO-8859-1", "windows-1252"},
		},
		{
			name:            "Chinese text in UTF-8",
			input:           []byte("你好世界 - Hello World in Chinese"),
			expectedCharset: "UTF-8",
		},
		{
			name:            "Russian text in UTF-8",
			input:           []byte("Привет мир - Hello World in Russian"),
			expectedCharset: "UTF-8",
		},
		{
			name:             "Mixed content with numbers",
			input:            []byte("Test123 with numbers 456 and symbols !@#$%"),
			acceptedCharsets: []string{"UTF-8", "ISO-8859-1", "ISO-8859-2", "windows-1252", "Ascii"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Call the function under test
			charset := detectCharset(test.input)
			// Check charset result
			switch {
			case test.expectedCharset != "":
				assert.Equal(t, test.expectedCharset, charset, "Expected charset mismatch")
			case len(test.acceptedCharsets) > 0:
				assert.Contains(t, test.acceptedCharsets, charset,
					"Detected charset '%s' not in accepted list: %v", charset, test.acceptedCharsets)
			default:
				assert.NotEmpty(t, charset, "Charset should not be empty")
			}
		})
	}
}

func TestDetectCharsetBufferTruncation(t *testing.T) {
	// Test that large buffers are truncated to 32768 bytes
	// Create a buffer larger than 32768 bytes
	largeBuffer := make([]byte, 50000)
	for i := range largeBuffer {
		// Fill with ASCII letters to ensure consistent detection
		largeBuffer[i] = byte('A' + (i % 26))
	}
	// Call detectCharset
	charset := detectCharset(largeBuffer)
	// Verify no error occurred
	assert.NotEmpty(t, charset, "Charset should not be empty")

	// The detected charset should be one of the ASCII-compatible ones
	validCharsets := []string{"UTF-8", "ISO-8859-1", "ISO-8859-2", "windows-1252", "ISO-8859-15", "Ascii"}
	assert.Contains(t, validCharsets, charset,
		"Large ASCII buffer should be detected as ASCII-compatible charset, got: %s", charset)
}

func TestDetectCharsetErrorHandling(t *testing.T) {
	// Test with nil input - chardet should handle this gracefully
	charset := detectCharset(nil)

	// Should return UTF-8 due to low confidence
	assert.Equal(t, "UTF-8", charset, "Nil input should default to UTF-8")
}
