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
	"testing"

	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestLicenseDetails(t *testing.T) {
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
			name:      "Test License Details - empty",
			binary:    "../../test-support/scanoss.sh",
			telemetry: true,
			input:     map[string]string{},
			want:      http.StatusBadRequest,
		},
		{
			name:      "Test License Details - wrong key",
			binary:    "../../test-support/scanoss.sh",
			telemetry: true,
			input:     map[string]string{"invalid": "wrong"},
			want:      http.StatusBadRequest,
		},
		{
			name:      "Test License Details - invalid binary",
			binary:    "scan-binary-does-not-exist.sh",
			telemetry: true,
			input:     map[string]string{"license": "GPL"},
			want:      http.StatusInternalServerError,
		},
		{
			name:      "Test License Details - success",
			binary:    "../../test-support/scanoss.sh",
			telemetry: false,
			input:     map[string]string{"license": "MIT"},
			want:      http.StatusOK,
		},
		{
			name:      "Test License Details - success 2",
			binary:    "../../test-support/scanoss.sh",
			telemetry: false,
			input:     map[string]string{"license": "Apache-2.0"},
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
			req := newReq("GET", "http://localhost/license/obligations/{license}", "", test.input)
			w := httptest.NewRecorder()
			apiService.LicenseDetails(w, req)
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
