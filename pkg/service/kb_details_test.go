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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestKBDetails(t *testing.T) {
	err := zlog.NewSugaredDevLogger()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a sugared logger", err)
	}
	defer zlog.SyncZap()

	myConfig := setupConfig(t)
	myConfig.App.Trace = true
	apiService := NewAPIService(myConfig)
	apiService.SetupKBDetailsCron()
	time.Sleep(time.Duration(5) * time.Second) // Sleep a little to allow the KB details to be loaded
	req := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	w := httptest.NewRecorder()
	apiService.KBDetails(w, req)

	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("an error was not expected when reading from request: %v", err)
	}
	bodyStr := string(body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	expected := `{"kb_version": { "monthly": "23.07", "daily": "23.08.09"}}`
	assert.Equal(t, expected, strings.TrimSpace(bodyStr))
	fmt.Println("Status: ", resp.StatusCode)
	fmt.Println("Type: ", resp.Header.Get("Content-Type"))
	fmt.Println("Body: ", bodyStr)

	// Test the version loading to fail
	myConfig.Scanning.ScanBinary = "../path/to/does-not-exist.sh"
	apiService.loadKBDetails()
}
