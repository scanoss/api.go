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
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"go.uber.org/zap"
)

func (s APIService) getFlags(r *http.Request, zs *zap.SugaredLogger) (string, string, string) {
	flags := strings.TrimSpace(r.FormValue("flags"))   // Check form for Scanning flags
	scanType := strings.TrimSpace(r.FormValue("type")) // Check form for SBOM type
	sbom := strings.TrimSpace(r.FormValue("assets"))   // Check form for SBOM contents
	// TODO is it necessary to check the header also for these values?
	if len(flags) == 0 {
		flags = strings.TrimSpace(r.Header.Get("flags")) // Check header for Scanning flags
	}
	if len(scanType) == 0 {
		scanType = strings.TrimSpace(r.Header.Get("type")) // Check header for SBOM type
	}
	if len(sbom) == 0 {
		sbom = strings.TrimSpace(r.Header.Get("assets")) // Check header for SBOM contents
	}
	if s.config.App.Trace {
		zs.Debugf("Header: %v, Form: %v, flags: %v, type: %v, assets: %v", r.Header, r.Form, flags, scanType, sbom)
	}
	return flags, scanType, sbom
}

// writeSbomFile writes the given string into an SBOM temporary file.
func (s APIService) writeSbomFile(sbom string, zs *zap.SugaredLogger) (*os.File, error) {
	tempFile, err := os.CreateTemp(s.config.Scanning.WfpLoc, "sbom*.json")
	if err != nil {
		zs.Errorf("Failed to create temporary SBOM file: %v", err)
		return nil, err
	}
	_, err = tempFile.WriteString(sbom + "\n")
	if err != nil {
		zs.Errorf("Failed to write to temporary SBOM file: %v - %v", tempFile.Name(), err)
		return tempFile, err
	}
	closeFile(tempFile, zs)
	return tempFile, nil
}

// singleScan runs a scan of the WFP in a single thread.
func (s APIService) singleScan(wfp, flags, sbomType, sbomFile string, zs *zap.SugaredLogger, w http.ResponseWriter) {
	zs.Debugf("Single threaded scan...")
	result, err := s.scanWfp(wfp, flags, sbomType, sbomFile, zs)
	if err != nil {
		zs.Errorf("Engine scan failed: %v", err)
		http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
	} else {
		zs.Debug("Scan completed")
		response := strings.TrimSpace(result)
		if len(response) == 0 {
			zs.Warnf("Nothing in the engine response")
			http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
		} else {
			w.Header().Set(ContentTypeKey, ApplicationJSON)
			printResponse(w, fmt.Sprintf("%s\n", response), zs, false)
		}
	}
}

// scanThreaded scan the given WFPs in multiple threads.
func (s APIService) scanThreaded(wfps []string, wfpCount int, flags, sbomType, sbomFile string, zs *zap.SugaredLogger, w http.ResponseWriter) {
	// Multiple workers, create input and output channels
	requests := make(chan string)
	results := make(chan string, wfpCount)
	numWorkers := s.config.Scanning.Workers
	if numWorkers > wfpCount {
		zs.Debugf("Requested workers (%v) greater than WFPs (%v). Reducing number.", numWorkers, wfpCount)
		numWorkers = wfpCount
	}
	zs.Debugf("Creating %v scanning workers...", numWorkers)
	// Create workers
	for i := 1; i <= numWorkers; i++ {
		go s.workerScan(fmt.Sprintf("%d_%s", i, uuid.New().String()), requests, results, flags, sbomType, sbomFile, zs)
	}
	requestCount := 0 // Count the number of actual requests sent
	var wfpRequests []string
	for _, wfp := range wfps {
		wfp = strings.TrimSpace(wfp)
		if len(wfp) == 0 { // Ignore empty requests
			continue
		}
		wfpRequests = append(wfpRequests, "file="+wfp)
		if len(wfpRequests) >= s.config.Scanning.WfpGrouping { // Reach the WFP target, submit the request
			if s.config.App.Trace {
				zs.Debugf("Submitting requests: %v", len(wfpRequests))
			}
			requests <- strings.Join(wfpRequests, "\n")
			requestCount++
			wfpRequests = wfpRequests[:0] // reset to empty (keeping the memory allocation)
		}
	}
	if len(wfpRequests) > 0 { // Submit the last unassigned WFPs to a request
		if s.config.App.Trace {
			zs.Debugf("Submitting last requests: %v", len(wfpRequests))
		}
		requests <- strings.Join(wfpRequests, "\n")
		requestCount++
	}
	close(requests) // No more requests. close the channel
	zs.Debugf("Finished sending requests: %v", requestCount)
	var responses []string
	for i := 0; i < requestCount; i++ { // Get results for the number of requests sent
		if s.config.App.Trace {
			zs.Debugf("Waiting for result %v", i)
		}
		result := <-results
		if s.config.App.Trace {
			zs.Debugf("Result %v: %v", i, strings.TrimSpace(result))
		}
		result = strings.TrimSpace(result)
		if len(result) > 0 {
			responses = append(responses, result)
		}
	}
	close(results)
	zs.Debugf("Responses: %v", len(responses))
	if len(responses) == 0 {
		zs.Errorf("Multi-engine scan failed to produce results")
		http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
	} else {
		w.Header().Set(ContentTypeKey, ApplicationJSON)
		printResponse(w, "{"+strings.Join(responses, ",")+"}\n", zs, false)
	}
}

// ScanDirect handles WFP scanning requests from a client.
func (s APIService) ScanDirect(w http.ResponseWriter, r *http.Request) {
	counters.incRequest("scan")
	reqID := getReqID(r)
	w.Header().Set(ResponseIDKey, reqID)
	zs := sugaredLogger(context.WithValue(r.Context(), RequestContextKey{}, reqID)) // Setup logger with context
	zs.Infof("%v request from %v", r.URL.Path, r.RemoteAddr)
	contents, err := s.getFormFile(r, zs, "WFP")
	if err != nil {
		http.Error(w, "ERROR receiving WFP file contents", http.StatusBadRequest)
		return
	}
	contentsTrimmed := bytes.TrimSpace(contents)
	if len(contentsTrimmed) == 0 {
		zs.Errorf("No WFP contents to scan (%v - %v)", len(contents), contents)
		http.Error(w, "ERROR no WFP contents supplied", http.StatusBadRequest)
		return
	}
	flags, scanType, sbom := s.getFlags(r, zs)
	// Check if we have an SBOM (and type) supplied
	var sbomFilename string
	if len(sbom) > 0 && len(scanType) > 0 {
		if scanType != "identify" && scanType != "blacklist" { // Make sure we have a valid SBOM scan type
			zs.Errorf("Invalid SBOM type: %v", scanType)
			http.Error(w, "ERROR invalid SBOM 'type' supplied", http.StatusBadRequest)
			return
		}
		tempFile, err := s.writeSbomFile(sbom, zs)
		if err != nil {
			http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
			return
		}
		if s.config.Scanning.TmpFileDelete {
			defer removeFile(tempFile, zs)
		}
		sbomFilename = tempFile.Name() // Save the SBOM filename
		zs.Debugf("Stored SBOM (%v) in %v", scanType, sbomFilename)
	}
	wfps := strings.Split(string(contentsTrimmed), "file=")
	wfpCount := len(wfps) - 1 // First entry in the array is empty (hence the -1)
	if wfpCount <= 0 {
		zs.Errorf("No WFP (file=...) entries found to scan")
		http.Error(w, "ERROR no WFP file contents (file=...) supplied", http.StatusBadRequest)
		return
	}
	counters.incRequestAmount("files", int64(wfpCount))
	zs.Debugf("Need to scan %v files", wfpCount)
	// Only one worker selected, so send the whole WFP in a single command
	if s.config.Scanning.Workers <= 1 {
		s.singleScan(string(contentsTrimmed), flags, scanType, sbomFilename, zs, w)
	} else {
		s.scanThreaded(wfps, wfpCount, flags, scanType, sbomFilename, zs, w)
	}
}

// workerScan attempts to process all incoming scanning jobs and dumps the results into the subsequent results channel.
func (s APIService) workerScan(id string, jobs <-chan string, results chan<- string, flags, sbomType, sbomFile string, zs *zap.SugaredLogger) {
	if s.config.App.Trace {
		zs.Debugf("Starting up scanning worker: %v", id)
	}
	for job := range jobs {
		if s.config.App.Trace {
			zs.Debugf("Scanning (%v): '%v'", id, job)
		} else {
			zs.Debugf("Scanning (%v)", id)
		}
		if len(job) == 0 {
			zs.Warnf("Nothing in the job request to scan. Ignoring")
			results <- ""
		} else {
			result, err := s.scanWfp(job, flags, sbomType, sbomFile, zs)
			if s.config.App.Trace {
				zs.Debugf("scan result (%v): %v, %v", id, result, err)
			}
			if err != nil {
				results <- ""
			} else {
				result = strings.TrimSpace(result) // remove any leading/trailing spaces
				resLen := len(result)
				if resLen > 1 && result[0] == '{' && result[resLen-1] == '}' {
					result = result[1 : resLen-1] // Strip leading/trailing brackets ({})
				}
				if s.config.App.Trace {
					zs.Debugf("Saving result: '%v'", result)
				}
				results <- strings.TrimSpace(result)
			}
		}
	}
	if s.config.App.Trace {
		zs.Debugf("Shutting down scanning worker: %v", id)
	}
}

// scanWfp run the scanoss engine scan of the supplied WFP.
func (s APIService) scanWfp(wfp, flags, sbomType, sbomFile string, zs *zap.SugaredLogger) (string, error) {
	if len(wfp) == 0 {
		zs.Warnf("Nothing in the job request to scan. Ignoring")
		return "", fmt.Errorf("no wfp supplied to scan. ignoring")
	}
	tempFile, err := os.CreateTemp(s.config.Scanning.WfpLoc, "finger*.wfp")
	if err != nil {
		zs.Errorf("Failed to create temporary file: %v", err)
		return "", fmt.Errorf("failed to create temporary WFP file")
	}
	if s.config.Scanning.TmpFileDelete {
		defer removeFile(tempFile, zs)
	}
	zs.Debugf("Using temporary file: %v", tempFile.Name())
	_, err = tempFile.WriteString(wfp + "\n")
	if err != nil {
		zs.Errorf("Failed to write WFP to temporary file: %v", err)
		return "", fmt.Errorf("failed to write to temporary WFP file")
	}
	closeFile(tempFile, zs)
	var args []string
	if s.config.Scanning.ScanDebug {
		args = append(args, "-d") // Set debug mode
	}
	if s.config.Scanning.ScanFlags > 0 { // Set system flags if enabled
		args = append(args, fmt.Sprintf("-F %v", s.config.Scanning.ScanFlags))
	} else if len(flags) > 0 && flags != "0" { // Set user supplied flags if enabled
		args = append(args, fmt.Sprintf("-F %s", flags))
	}
	if len(sbomFile) > 0 && len(sbomType) > 0 { // Add SBOM to scanning process
		switch sbomType {
		case "identify":
			args = append(args, "-s")
		case "blacklist":
			args = append(args, "-b")
		default:
			args = append(args, "-s") // Default to identify
		}
		args = append(args, sbomFile)
	}
	args = append(args, "-w", tempFile.Name())
	zs.Debugf("Executing %v %v", s.config.Scanning.ScanBinary, strings.Join(args, " "))
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Scanning.ScanTimeout)*time.Second) // put a timeout on the scan execution
	defer cancel()
	output, err := exec.CommandContext(ctx, s.config.Scanning.ScanBinary, args...).Output()
	if err != nil {
		zs.Errorf("Scan command (%v %v) failed: %v", s.config.Scanning.ScanBinary, args, err)
		zs.Errorf("Command output: %s", bytes.TrimSpace(output))
		if s.config.Scanning.KeepFailedWfps {
			s.copyWfpTempFile(tempFile.Name(), zs)
		}
		return "", fmt.Errorf("failed to scan WFP: %v", err)
	}
	return string(output), nil
}

// TestEngine tests if the SCANOSS engine is accessible and running.
func (s APIService) TestEngine() error {
	zlog.S.Infof("Testing engine command: %v", s.config.Scanning.ScanBinary)
	var args []string
	args = append(args, "-h")
	zlog.S.Debugf("Executing %v %v", s.config.Scanning.ScanBinary, strings.Join(args, " "))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // put a timeout on the scanoss execution
	defer cancel()
	output, err := exec.CommandContext(ctx, s.config.Scanning.ScanBinary, args...).Output()
	if err != nil {
		zlog.S.Errorf("Scan test command (%v %v) failed: %v", s.config.Scanning.ScanBinary, args, err)
		zlog.S.Errorf("Command output: %s", bytes.TrimSpace(output))
		return fmt.Errorf("failed to test scan engine: %v", err)
	}
	return nil
}
