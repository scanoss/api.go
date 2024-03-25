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
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

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
			req := newReq("GET", "http://localhost/metrics/{type}", "", test.input)
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
