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
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// SbomAttribution handles retrieving the attribution notices for the given SBOM.
func (s APIService) SbomAttribution(w http.ResponseWriter, r *http.Request) {
	counters.incRequest("attribution")
	reqID := getReqID(r)
	w.Header().Set(ResponseIDKey, reqID)
	zs := sugaredLogger(context.WithValue(r.Context(), RequestContextKey{}, reqID)) // Setup logger with context
	zs.Infof("%v request from %v", r.URL.Path, r.RemoteAddr)
	var contents []byte
	var err error
	formFiles := []string{"file", "filename"}
	for _, fName := range formFiles { // Check for the SBOM contents in 'file' and 'filename'
		var file multipart.File
		file, _, err = r.FormFile(fName)
		if err != nil {
			zs.Infof("Cannot retrieve SBOM Form File: %v - %v. Trying an alternative name...", fName, err)
			continue
		}
		contents, err = io.ReadAll(file) // Load the file (SBOM) contents into memory
		closeMultipartFile(file, zs)
		if err == nil {
			break // We have successfully gotten the file contents
		} else {
			zs.Infof("Cannot retrieve SBOM Form File (%v) contents: %v. Trying an alternative name...", file, err)
		}
	}
	if err != nil {
		zs.Errorf("Failed to retrieve SBOM file contents (using %v): %v", formFiles, err)
		http.Error(w, "ERROR receiving SBOM file contents", http.StatusBadRequest)
		return
	}
	contentsTrimmed := bytes.TrimSpace(contents)
	if len(contentsTrimmed) == 0 {
		zs.Errorf("No SBOM contents to attribute (%v - %v)", len(contents), contents)
		http.Error(w, "ERROR no SBOM contents supplied", http.StatusBadRequest)
		return
	}
	// Check if we have an SBOM (and type) supplied
	tempFile, err := os.CreateTemp(s.config.Scanning.WfpLoc, "sbom-attr*.json")
	if err != nil {
		zs.Errorf("Failed to create temporary SBOM file: %v", err)
		http.Error(w, "ERROR engine attribution failed", http.StatusInternalServerError)
		return
	}
	_, err = tempFile.Write(contentsTrimmed)
	if err != nil {
		zs.Errorf("Failed to write to temporary SBOM file: %v - %v", tempFile.Name(), err)
		http.Error(w, "ERROR engine attribution failed", http.StatusInternalServerError)
		return
	}
	closeFile(tempFile, zs)
	if s.config.Scanning.TmpFileDelete {
		defer removeFile(tempFile, zs)
	}
	sbomFilename := tempFile.Name() // Save the SBOM filename

	zs.Debugf("Retrieving attribution for %v", sbomFilename)
	var args []string
	if s.config.Scanning.ScanDebug {
		args = append(args, "-d")
	}
	args = append(args, "-a", sbomFilename)
	zs.Debugf("Executing %v %v", s.config.Scanning.ScanBinary, strings.Join(args, " "))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // put a timeout on the scan execution
	defer cancel()
	output, err := exec.CommandContext(ctx, s.config.Scanning.ScanBinary, args...).Output()
	if err != nil {
		zs.Errorf("Attribution command (%v %v) failed: %v", s.config.Scanning.ScanBinary, args, err)
		zs.Errorf("Command output: %s", bytes.TrimSpace(output))
		http.Error(w, "ERROR engine attribution failed", http.StatusInternalServerError)
		return
	}
	if s.config.App.Trace {
		zs.Debugf("Sending back attribution: %v - '%s'", len(output), output)
	} else {
		zs.Debugf("Sending back attribution: %v", len(output))
	}
	w.Header().Set(ContentTypeKey, TextPlain)
	printResponse(w, string(output), zs, false)
}
