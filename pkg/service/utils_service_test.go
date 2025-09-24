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
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/golobby/config/v3"
	"github.com/gorilla/mux"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"github.com/stretchr/testify/assert"
	myconfig "scanoss.com/go-api/pkg/config"
)

// newReq sets up a request with specified URL variables.
func newReq(method, path, body string, vars map[string]string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	return mux.SetURLVars(r, vars)
}

// setupConfig sets up the default config for use.
func setupConfig(t *testing.T) *myconfig.ServerConfig {
	var feeders []config.Feeder
	myConfig, err := myconfig.NewServerConfig(feeders)
	if err != nil {
		t.Fatalf("an error was not expected when loading config: %v", err)
	}
	myConfig.Scanning.ScanDebug = true
	myConfig.Scanning.ScanBinary = "../../test-support/scanoss.sh"
	return myConfig
}

func TestWelcomeMsg(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	req := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	w := httptest.NewRecorder()
	WelcomeMsg(w, req)
	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("an error was not expected when reading from request: %v", err)
	}
	bodyStr := string(body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	expected := `{"msg": "Hello from the SCANOSS Scanning API"}`
	assert.Equal(t, expected+"\n", bodyStr)
	fmt.Println("Status: ", resp.StatusCode)
	fmt.Println("Type: ", resp.Header.Get("Content-Type"))
	fmt.Println("Body: ", bodyStr)
}

func TestHealthCheck(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	req := httptest.NewRequest(http.MethodGet, "http://localhost/health", nil)
	w := httptest.NewRecorder()
	HealthCheck(w, req)
	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("an error was not expected when reading from request: %v", err)
	}
	bodyStr := string(body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	expected := `{"alive": true}`
	assert.Equal(t, expected+"\n", bodyStr)
	fmt.Println("Status: ", resp.StatusCode)
	fmt.Println("Type: ", resp.Header.Get("Content-Type"))
	fmt.Println("Body: ", bodyStr)
}

func TestMetricsHandler(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	counters.incRequest("attribution")

	tests := []struct {
		name  string
		input map[string]string
		want  int
	}{
		{
			name:  "Test goroutines",
			input: map[string]string{"type": "goroutines"},
			want:  http.StatusOK,
		},
		{
			name:  "Test heap",
			input: map[string]string{"type": "heap"},
			want:  http.StatusOK,
		},
		{
			name:  "Test requests",
			input: map[string]string{"type": "requests"},
			want:  http.StatusOK,
		},
		{
			name:  "Test all",
			input: map[string]string{"type": "all"},
			want:  http.StatusOK,
		},
		{
			name:  "Test invalid",
			input: map[string]string{"type": "invalid"},
			want:  http.StatusBadRequest,
		},
		{
			name:  "Test no request",
			input: map[string]string{},
			want:  http.StatusBadRequest,
		},
		{
			name:  "Test wrong request type",
			input: map[string]string{"invalid": "nothing"},
			want:  http.StatusBadRequest,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := newReq(http.MethodGet, "http://localhost/metrics/{type}", "", test.input)
			w := httptest.NewRecorder()
			MetricsHandler(w, req)
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

func TestApiService(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	myConfig := setupConfig(t)
	apiService := NewAPIService(myConfig)
	apiService.copyWfpTempFile("", zlog.S)
	tempFile := apiService.copyWfpTempFile("utils_service.go", zlog.S)
	assert.NotEmpty(t, tempFile)
	source, err := os.Open(tempFile)
	if err != nil {
		t.Fatalf("Failed to open file %v: %v", tempFile, err)
	}
	removeFile(source, zlog.S)
	removeFile(source, zlog.S)
	closeFile(source, zlog.S)
	closeFile(source, zlog.S)

	req := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	w := httptest.NewRecorder()
	reqID := getReqID(req)
	assert.NotEmpty(t, reqID)
	fmt.Println("ReqId: ", reqID)
	zs := sugaredLogger(context.WithValue(req.Context(), RequestContextKey{}, reqID)) // Setup logger with context
	assert.NotNil(t, zs)

	printResponse(w, "test message", zs, false)
}

func TestHeadResponse(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()
	req := httptest.NewRequest(http.MethodHead, "http://localhost/health", nil)
	w := httptest.NewRecorder()
	HeadResponse(w, req)
	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("an error was not expected when reading from request: %v", err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	fmt.Println("Status: ", resp.StatusCode)
	fmt.Println("Type: ", resp.Header.Get("Content-Type"))
	fmt.Println("Body: ", string(body))
}

func TestLogRequestDetails(t *testing.T) {
	tests := []struct {
		name               string
		path               string
		method             string
		remoteAddr         string
		headers            map[string]string
		expectedLogCount   int
		expectedLogLevel   zapcore.Level
		expectedLogMessage string
		expectedFields     map[string]interface{}
	}{
		{
			name:               "Basic request without proxy headers",
			path:               "/scan/direct",
			method:             http.MethodPost,
			remoteAddr:         "192.168.1.100:12345",
			headers:            map[string]string{},
			expectedLogCount:   1,
			expectedLogLevel:   zapcore.InfoLevel,
			expectedLogMessage: "Request received",
			expectedFields: map[string]interface{}{
				"path":      "/scan/direct",
				"source_ip": "192.168.1.100:12345",
				"method":    http.MethodPost,
			},
		},
		{
			name:       "Request with X-Forwarded-For header",
			path:       "/kb/details",
			method:     http.MethodGet,
			remoteAddr: "10.0.0.1:54321",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.45",
			},
			expectedLogCount:   1,
			expectedLogLevel:   zapcore.InfoLevel,
			expectedLogMessage: "Request received",
			expectedFields: map[string]interface{}{
				"path":            "/kb/details",
				"source_ip":       "10.0.0.1:54321",
				"method":          http.MethodGet,
				"x_forwarded_for": "203.0.113.45",
			},
		},
		{
			name:       "Request with X-Real-IP header (when X-Forwarded-For is empty)",
			path:       "/health",
			method:     http.MethodGet,
			remoteAddr: "172.16.0.1:8080",
			headers: map[string]string{
				"X-Real-IP": "198.51.100.25",
			},
			expectedLogCount:   1,
			expectedLogLevel:   zapcore.InfoLevel,
			expectedLogMessage: "Request received",
			expectedFields: map[string]interface{}{
				"path":            "/health",
				"source_ip":       "172.16.0.1:8080",
				"method":          http.MethodGet,
				"x_forwarded_for": "198.51.100.25",
			},
		},
		{
			name:       "Request with CF-Connecting-IP header (Cloudflare)",
			path:       "/metrics/prometheus",
			method:     http.MethodGet,
			remoteAddr: "10.1.1.1:443",
			headers: map[string]string{
				"CF-Connecting-IP": "203.0.113.100",
			},
			expectedLogCount:   1,
			expectedLogLevel:   zapcore.InfoLevel,
			expectedLogMessage: "Request received",
			expectedFields: map[string]interface{}{
				"path":            "/metrics/prometheus",
				"source_ip":       "10.1.1.1:443",
				"method":          http.MethodGet,
				"x_forwarded_for": "203.0.113.100",
			},
		},
		{
			name:       "Request with multiple proxy headers (X-Forwarded-For takes precedence)",
			path:       "/scan/direct",
			method:     http.MethodPost,
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For":  "203.0.113.1",
				"X-Real-IP":        "198.51.100.1",
				"CF-Connecting-IP": "192.0.2.1",
			},
			expectedLogCount:   1,
			expectedLogLevel:   zapcore.InfoLevel,
			expectedLogMessage: "Request received",
			expectedFields: map[string]interface{}{
				"path":            "/scan/direct",
				"source_ip":       "10.0.0.1:12345",
				"method":          http.MethodPost,
				"x_forwarded_for": "203.0.113.1",
			},
		},
		{
			name:       "Request with comma-separated X-Forwarded-For",
			path:       "/sbom/attribution",
			method:     http.MethodPost,
			remoteAddr: "10.0.0.1:9090",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1, 198.51.100.1, 192.0.2.1",
			},
			expectedLogCount:   1,
			expectedLogLevel:   zapcore.InfoLevel,
			expectedLogMessage: "Request received",
			expectedFields: map[string]interface{}{
				"path":            "/sbom/attribution",
				"source_ip":       "10.0.0.1:9090",
				"method":          http.MethodPost,
				"x_forwarded_for": "203.0.113.1, 198.51.100.1, 192.0.2.1",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a logger with an observer to capture log entries
			observedZapCore, observedLogs := observer.New(zapcore.InfoLevel)
			observedLogger := zap.New(observedZapCore).Sugar()

			// Create a test HTTP request
			req := httptest.NewRequest(test.method, test.path, bytes.NewReader([]byte{}))
			req.RemoteAddr = test.remoteAddr

			// Set headers from the headers map
			for key, value := range test.headers {
				req.Header.Set(key, value)
			}

			// Call the function under test
			logRequestDetails(req, observedLogger)

			// Verify log entry count
			assert.Equal(t, test.expectedLogCount, observedLogs.Len(), "Expected log count mismatch")

			if test.expectedLogCount > 0 {
				// Get the first (and should be only) log entry
				logEntry := observedLogs.All()[0]

				// Verify log level
				assert.Equal(t, test.expectedLogLevel, logEntry.Level, "Expected log level mismatch")

				// Verify log message
				assert.Equal(t, test.expectedLogMessage, logEntry.Message, "Expected log message mismatch")

				// Verify log fields
				for expectedKey, expectedValue := range test.expectedFields {
					actualValue := logEntry.ContextMap()[expectedKey]
					assert.Equal(t, expectedValue, actualValue, "Expected field '%s' mismatch", expectedKey)
				}
			}
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name                string
		remoteAddr          string
		headers             map[string]string
		expectedSourceIP    string
		expectedForwardedIP string
	}{
		{
			name:                "No proxy headers",
			remoteAddr:          "192.168.1.100:12345",
			headers:             map[string]string{},
			expectedSourceIP:    "192.168.1.100:12345",
			expectedForwardedIP: "",
		},
		{
			name:       "X-Forwarded-For present",
			remoteAddr: "10.0.0.1:54321",
			headers: map[string]string{
				"X-Forwarded-For":  "203.0.113.45",
				"X-Real-IP":        "198.51.100.25",
				"CF-Connecting-IP": "192.0.2.50",
			},
			expectedSourceIP:    "10.0.0.1:54321",
			expectedForwardedIP: "203.0.113.45",
		},
		{
			name:       "X-Real-IP used when X-Forwarded-For empty",
			remoteAddr: "172.16.0.1:8080",
			headers: map[string]string{
				"X-Real-IP":        "198.51.100.25",
				"CF-Connecting-IP": "192.0.2.50",
			},
			expectedSourceIP:    "172.16.0.1:8080",
			expectedForwardedIP: "198.51.100.25",
		},
		{
			name:       "CF-Connecting-IP used when others empty",
			remoteAddr: "10.1.1.1:443",
			headers: map[string]string{
				"CF-Connecting-IP": "203.0.113.100",
			},
			expectedSourceIP:    "10.1.1.1:443",
			expectedForwardedIP: "203.0.113.100",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a test HTTP request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = test.remoteAddr
			// Set headers from the headers map
			for key, value := range test.headers {
				req.Header.Set(key, value)
			}
			// Call the function under test
			sourceIP, forwardedIP := getClientIP(req)
			// Verify results
			assert.Equal(t, test.expectedSourceIP, sourceIP, "Expected source IP mismatch")
			assert.Equal(t, test.expectedForwardedIP, forwardedIP, "Expected forwarded IP mismatch")
		})
	}
}
