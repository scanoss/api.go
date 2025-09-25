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
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/wlynxg/chardet"
)

// FileContents handles retrieval of sources file for a client.
func (s APIService) FileContents(w http.ResponseWriter, r *http.Request) {
	counters.incRequest("file_contents")
	reqID := getReqID(r)
	w.Header().Set(ResponseIDKey, reqID)
	var logContext context.Context
	if s.config.Telemetry.Enabled {
		_, logContext = getSpan(r.Context(), reqID)
		oltpMetrics.fileContentsCounter.Add(logContext, 1)
	} else {
		logContext = requestContext(r.Context(), reqID, "", "")
	}
	zs := sugaredLogger(logContext) // Setup logger with context
	logRequestDetails(r, zs)
	vars := mux.Vars(r)
	zs.Debugf("%v request from %v - %v", r.URL.Path, r.RemoteAddr, vars)
	if len(vars) == 0 {
		zs.Errorf("Failed to retrieve request variables")
		http.Error(w, "ERROR no request variables submitted", http.StatusBadRequest)
	}
	md5, ok := vars["md5"]
	if !ok {
		zs.Errorf("Failed to retrieve md5 request variable from: %v", vars)
		http.Error(w, "ERROR no md5 request variable submitted", http.StatusBadRequest)
	}
	zs.Debugf("Retrieving contents for %v", md5)
	var args []string
	if s.config.Scanning.ScanDebug {
		args = append(args, "-d")
	}
	args = append(args, "-k", md5)
	zs.Debugf("Executing %v %v", s.config.Scanning.ScanBinary, strings.Join(args, " "))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // put a timeout on the scan execution
	defer cancel()
	output, err := exec.CommandContext(ctx, s.config.Scanning.ScanBinary, args...).Output()
	if err != nil {
		zs.Errorf("Contents command (%v %v) failed: %v", s.config.Scanning.ScanBinary, args, err)
		zs.Errorf("Command output: %s", bytes.TrimSpace(output))
		http.Error(w, "ERROR recovering file contents", http.StatusInternalServerError)
		return
	}
	charset := detectCharset(output)
	if s.config.App.Trace {
		zs.Debugf("Sending back contents: %v - '%s'", len(output), output)
	} else {
		zs.Debugf("Sending back contents: %v", len(output))
	}
	w.Header().Set(ContentTypeKey, fmt.Sprintf("text/plain; charset=%s", charset))
	w.Header().Set(CharsetDetectedKey, charset)
	w.Header().Set(ContentLengthKey, fmt.Sprintf("%d", len(output)))
	printResponse(w, string(output), zs, false)
}

// detectCharset detects charset for a given text in a buffer.
func detectCharset(buffer []byte) string {
	if len(buffer) > 32768 {
		buffer = buffer[:32768]
	}
	// Detect charset.
	result := chardet.Detect(buffer)
	// If confidence is low, consider it as UTF-8.
	if result.Confidence < CharSetMinConfidence {
		return "UTF-8"
	}
	return result.Encoding
}
