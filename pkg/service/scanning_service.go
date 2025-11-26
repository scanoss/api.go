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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/google/uuid"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"go.uber.org/zap"
)

var fileRegex = regexp.MustCompile(`^\w+,(\d+),.+`) // regex to parse file size from request

// ScanDirect handles WFP scanning requests from a client.
func (s APIService) ScanDirect(w http.ResponseWriter, r *http.Request) {
	requestStartTime := time.Now() // Capture the scan start time
	counters.incRequest("scan")
	reqID := getReqID(r)
	w.Header().Set(ResponseIDKey, reqID)
	var logContext context.Context
	var span oteltrace.Span
	// Get the oltp span (if requested) and set logging context
	if s.config.Telemetry.Enabled {
		span, logContext = getSpan(r.Context(), reqID)
	} else {
		logContext = requestContext(r.Context(), reqID, "", "")
	}
	zs := sugaredLogger(logContext) // Setup logger with context
	wfpCount := s.scanDirect(w, r, zs, logContext, span)
	elapsedTime := time.Since(requestStartTime).Milliseconds() // Time taken to run the scan
	if s.config.Telemetry.Enabled {
		elapsedTimeSeconds := float64(elapsedTime) / 1000.0                 // Convert to seconds
		oltpMetrics.scanHistogram.Record(logContext, elapsedTime)           // Record scan time
		oltpMetrics.scanHistogramSec.Record(logContext, elapsedTimeSeconds) // Record scan time seconds
		var fileScanTime int64
		if wfpCount > 0 {
			fileScanTime = elapsedTime / wfpCount
			oltpMetrics.scanFileHistogram.Record(logContext, fileScanTime)                            // Record average file scan time
			oltpMetrics.scanFileHistogramSec.Record(logContext, elapsedTimeSeconds/float64(wfpCount)) // Record average file scan time seconds
		}
		if s.config.App.Trace {
			zs.Debugf("Scan stats: files: %v, scan_time: %v, file_time: %v", wfpCount, elapsedTime, fileScanTime)
		}
	}
}

// ScanBatch handles batch WFP scanning requests with chunked uploads from a client.
// Chunks are accumulated in a session-specific WFP file, and the scan is launched when X-Final-Chunk: true is received.
func (s APIService) ScanBatch(w http.ResponseWriter, r *http.Request) {
	requestStartTime := time.Now() // Capture the scan start time
	counters.incRequest("scan")
	reqID := getReqID(r)
	w.Header().Set(ResponseIDKey, reqID)

	var logContext context.Context
	var span oteltrace.Span
	// Get the oltp span (if requested) and set logging context
	if s.config.Telemetry.Enabled {
		span, logContext = getSpan(r.Context(), reqID)
	} else {
		logContext = requestContext(r.Context(), reqID, "", "")
	}
	zs := sugaredLogger(logContext) // Setup logger with context
	logRequestDetails(r, zs)

	// Extract Session-Id header (required)
	sessionID := strings.TrimSpace(r.Header.Get("Session-Id"))
	if len(sessionID) == 0 {
		zs.Errorf("Missing Session-Id header")
		http.Error(w, "ERROR Session-Id header is required", http.StatusBadRequest)
		setSpanError(span, "Missing Session-Id header")
		return
	}

	// Validate session ID to prevent path traversal attacks
	if strings.Contains(sessionID, "/") || strings.Contains(sessionID, "..") {
		zs.Errorf("Invalid Session-Id: %v", sessionID)
		http.Error(w, "ERROR invalid Session-Id", http.StatusBadRequest)
		setSpanError(span, "Invalid Session-Id")
		return
	}

	// Extract X-Final-Chunk header (optional, defaults to false)
	finalChunk := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Final-Chunk"))) == "true"

	// Get WFP chunk from multipart form
	chunkContents, err := s.getFormFile(r, zs, "WFP")
	if err != nil {
		http.Error(w, "ERROR receiving WFP chunk contents", http.StatusBadRequest)
		return
	}

	chunkTrimmed := bytes.TrimSpace(chunkContents)
	if len(chunkTrimmed) == 0 {
		zs.Errorf("No WFP chunk contents to append (%v - %v)", len(chunkContents), chunkContents)
		http.Error(w, "ERROR no WFP chunk contents supplied", http.StatusBadRequest)
		setSpanError(span, "No WFP chunk contents supplied")
		return
	}

	// Append chunk to session file
	sessionFilePath := filepath.Join(s.config.Scanning.WfpLoc, sessionID+".wfp")
	if err := s.appendWfpChunk(sessionID, sessionFilePath, chunkTrimmed, zs); err != nil {
		zs.Errorf("Failed to append WFP chunk: %v", err)
		http.Error(w, "ERROR failed to append WFP chunk", http.StatusInternalServerError)
		setSpanError(span, fmt.Sprintf("Failed to append WFP chunk: %v", err))
		return
	}

	zs.Debugf("Appended WFP chunk to session %v (final: %v)", sessionID, finalChunk)

	// If this is the final chunk, launch the scan
	if finalChunk {
		// Release the session lock after scan completes
		defer sessionLocks.releaseSessionLock(sessionID)
		defer removeFileByPath(sessionFilePath, zs) // Clean up session file after scan

		zs.Infof("Final chunk received for session %v, launching scan...", sessionID)

		// Read the complete WFP file
		completeWfp, err := os.ReadFile(sessionFilePath)
		if err != nil {
			zs.Errorf("Failed to read complete WFP file: %v", err)
			http.Error(w, "ERROR failed to read complete WFP file", http.StatusInternalServerError)
			setSpanError(span, fmt.Sprintf("Failed to read WFP file: %v", err))
			return
		}

		contentsTrimmed := bytes.TrimSpace(completeWfp)
		if len(contentsTrimmed) == 0 {
			zs.Errorf("No WFP contents to scan in session %v", sessionID)
			http.Error(w, "ERROR no WFP contents in session", http.StatusBadRequest)
			setSpanError(span, "No WFP contents in session")
			return
		}

		// Extract optional parameters (same as scan/direct)
		flags, scanType, sbom, dbName := s.getFlags(r, zs)

		// Handle SBOM if provided
		var sbomFilename string
		if len(sbom) > 0 && len(scanType) > 0 {
			if scanType != "identify" && scanType != "blacklist" {
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
			sbomFilename = tempFile.Name()
			zs.Debugf("Stored SBOM (%v) in %v", scanType, sbomFilename)
		}

		// Parse and validate WFP
		wfps := strings.Split(string(contentsTrimmed), "file=")
		wfpCount := int64(len(wfps) - 1)
		if wfpCount <= 0 {
			zs.Errorf("No WFP (file=...) entries found to scan in session %v", sessionID)
			http.Error(w, "ERROR no WFP file contents (file=...) supplied", http.StatusBadRequest)
			setSpanError(span, "No WFP (file=...) entries found.")
			return
		}

		if !s.validateHPSM(contentsTrimmed, zs, w) {
			setSpanError(span, "HPSM disabled.")
			return
		}

		s.countScanSize(wfps, wfpCount, zs, logContext, span)

		// Execute scan (single or multi-threaded)
		if s.config.Scanning.Workers <= 1 {
			s.singleScan(string(contentsTrimmed), flags, scanType, sbomFilename, dbName, zs, w)
		} else {
			s.scanThreaded(wfps, int(wfpCount), flags, scanType, sbomFilename, dbName, zs, w, span)
		}

		// Record metrics for elapsed time
		elapsedTime := time.Since(requestStartTime).Milliseconds()
		if s.config.Telemetry.Enabled {
			elapsedTimeSeconds := float64(elapsedTime) / 1000.0
			oltpMetrics.scanHistogram.Record(logContext, elapsedTime)
			oltpMetrics.scanHistogramSec.Record(logContext, elapsedTimeSeconds)
			var fileScanTime int64
			if wfpCount > 0 {
				fileScanTime = elapsedTime / wfpCount
				oltpMetrics.scanFileHistogram.Record(logContext, fileScanTime)
				oltpMetrics.scanFileHistogramSec.Record(logContext, elapsedTimeSeconds/float64(wfpCount))
			}
			if s.config.App.Trace {
				zs.Debugf("Batch scan stats: session: %v, files: %v, scan_time: %v, file_time: %v", sessionID, wfpCount, elapsedTime, fileScanTime)
			}
		}
	} else {
		// Not the final chunk, return 202 Accepted
		w.WriteHeader(http.StatusAccepted)
		w.Header().Set(ContentTypeKey, ApplicationJSON)
		printResponse(w, fmt.Sprintf("{\"message\":\"Chunk received for session %s\"}\n", sessionID), zs, false)
	}
}

// appendWfpChunk appends a WFP chunk to the session-specific WFP file with proper locking.
func (s APIService) appendWfpChunk(sessionID, filePath string, chunk []byte, zs *zap.SugaredLogger) error {
	// Get session-specific lock to prevent concurrent writes
	lock := sessionLocks.getSessionLock(sessionID)
	lock.Lock()
	defer lock.Unlock()

	// Open file in append mode (create if doesn't exist)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		zs.Errorf("Failed to open session file %v: %v", filePath, err)
		return fmt.Errorf("failed to open session file: %w", err)
	}
	defer closeFile(file, zs)

	// Append chunk with newline
	if _, err := file.Write(chunk); err != nil {
		zs.Errorf("Failed to write to session file %v: %v", filePath, err)
		return fmt.Errorf("failed to write to session file: %w", err)
	}

	// Add newline if chunk doesn't end with one
	if len(chunk) > 0 && chunk[len(chunk)-1] != '\n' {
		if _, err := file.WriteString("\n"); err != nil {
			zs.Errorf("Failed to write newline to session file %v: %v", filePath, err)
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	zs.Debugf("Successfully appended %v bytes to session file %v", len(chunk), filePath)
	return nil
}

// scanDirect handles WFP scanning requests from a client.
func (s APIService) scanDirect(w http.ResponseWriter, r *http.Request, zs *zap.SugaredLogger, context context.Context, span oteltrace.Span) int64 {
	logRequestDetails(r, zs)
	contents, err := s.getFormFile(r, zs, "WFP")
	if err != nil {
		http.Error(w, "ERROR receiving WFP file contents", http.StatusBadRequest)
		return 0
	}
	contentsTrimmed := bytes.TrimSpace(contents)
	if len(contentsTrimmed) == 0 {
		zs.Errorf("No WFP contents to scan (%v - %v)", len(contents), contents)
		http.Error(w, "ERROR no WFP contents supplied", http.StatusBadRequest)
		setSpanError(span, "No WFP contents supplied")
		return 0
	}
	flags, scanType, sbom, dbName := s.getFlags(r, zs)
	// Check if we have an SBOM (and type) supplied
	var sbomFilename string
	if len(sbom) > 0 && len(scanType) > 0 {
		if scanType != "identify" && scanType != "blacklist" { // Make sure we have a valid SBOM scan type
			zs.Errorf("Invalid SBOM type: %v", scanType)
			http.Error(w, "ERROR invalid SBOM 'type' supplied", http.StatusBadRequest)
			return 0
		}
		tempFile, err := s.writeSbomFile(sbom, zs)
		if err != nil {
			http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
			return 0
		}
		if s.config.Scanning.TmpFileDelete {
			defer removeFile(tempFile, zs)
		}
		sbomFilename = tempFile.Name() // Save the SBOM filename
		zs.Debugf("Stored SBOM (%v) in %v", scanType, sbomFilename)
	}
	wfps := strings.Split(string(contentsTrimmed), "file=")
	wfpCount := int64(len(wfps) - 1) // First entry in the array is empty (hence the -1)
	if wfpCount <= 0 {
		zs.Errorf("No WFP (file=...) entries found to scan")
		http.Error(w, "ERROR no WFP file contents (file=...) supplied", http.StatusBadRequest)
		setSpanError(span, "No WFP (file=...) entries found.")
		return 0
	}
	if !s.validateHPSM(contentsTrimmed, zs, w) {
		setSpanError(span, "HPSM disabled.")
		return 0
	}
	s.countScanSize(wfps, wfpCount, zs, context, span)
	// Only one worker selected, so send the whole WFP in a single command
	if s.config.Scanning.Workers <= 1 {
		s.singleScan(string(contentsTrimmed), flags, scanType, sbomFilename, dbName, zs, w)
	} else {
		s.scanThreaded(wfps, int(wfpCount), flags, scanType, sbomFilename, dbName, zs, w, span)
	}
	return wfpCount
}

// countScanSize parses the WFPs to calculate the size of the scan request and record it for metrics.
func (s APIService) countScanSize(wfps []string, wfpCount int64, zs *zap.SugaredLogger, context context.Context, span oteltrace.Span) {
	var sizeCount int64 = 0
	for _, wfp := range wfps {
		matches := fileRegex.FindStringSubmatch(wfp)
		if len(matches) > 0 {
			i, err := strconv.ParseInt(matches[1], 10, 64)
			if err == nil {
				sizeCount += i
			} else {
				zs.Warnf("Problem parsing file size from %v - %v: %v", matches[1], err, wfp)
			}
		}
	}
	counters.incRequestAmount("files", wfpCount)
	if s.config.Telemetry.Enabled {
		oltpMetrics.scanFileCounter.Add(context, wfpCount)
		span.SetAttributes(attribute.Int64("scan.file_count", wfpCount), attribute.String("scan.engine_version", engineVersion))
		if sizeCount > 0 {
			oltpMetrics.scanSizeCounter.Add(context, sizeCount)
			span.SetAttributes(attribute.Int64("scan.file_size", sizeCount))
		}
	}
	zs.Infof("Need to scan %v files of size %v", wfpCount, sizeCount)
}

// getFlags extracts the form values from a request returns the flags, scan type, and sbom data if detected.
func (s APIService) getFlags(r *http.Request, zs *zap.SugaredLogger) (string, string, string, string) {
	flags := strings.TrimSpace(r.FormValue("flags"))    // Check form for Scanning flags
	scanType := strings.TrimSpace(r.FormValue("type"))  // Check form for SBOM type
	sbom := strings.TrimSpace(r.FormValue("assets"))    // Check form for SBOM contents
	dbName := strings.TrimSpace(r.FormValue("db_name")) // Check form for db name
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
	if len(dbName) == 0 {
		dbName = strings.TrimSpace(r.Header.Get("db_name")) // Check header for SBOM contents
	}
	if s.config.App.Trace {
		zs.Debugf("Header: %v, Form: %v, flags: %v, type: %v, assets: %v, db_name %v", r.Header, r.Form, flags, scanType, sbom, dbName)
	}
	return flags, scanType, sbom, dbName
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
func (s APIService) singleScan(wfp, flags, sbomType, sbomFile, dbName string, zs *zap.SugaredLogger, w http.ResponseWriter) {
	zs.Debugf("Single threaded scan...")
	result, err := s.scanWfp(wfp, flags, sbomType, sbomFile, dbName, zs)
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
func (s APIService) scanThreaded(wfps []string, wfpCount int, flags, sbomType, sbomFile, dbName string, zs *zap.SugaredLogger, w http.ResponseWriter, span oteltrace.Span) {
	addSpanEvent(span, "Started Scanning.")
	numWorkers := s.config.Scanning.Workers
	groupedWfps := wfpCount / s.config.Scanning.WfpGrouping
	if numWorkers > groupedWfps {
		zs.Debugf("Requested workers (%v) greater than WFPs (%v). Reducing number.", numWorkers, groupedWfps)
		numWorkers = groupedWfps
	}
	if numWorkers < 1 {
		numWorkers = 1 // Make sure we have at least one worker
	}
	// Multiple workers, create input and output channels
	requests := make(chan string)
	results := make(chan string, groupedWfps+1)
	zs.Debugf("Creating %v scanning workers...", numWorkers)
	// Create workers
	for i := 1; i <= numWorkers; i++ {
		go s.workerScan(fmt.Sprintf("%d_%s", i, uuid.New().String()), requests, results, flags, sbomType, sbomFile, dbName, zs)
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
	addSpanEvent(span, "Finished Scanning.")
	responsesLength := len(responses)
	if requestCount != responsesLength {
		zs.Warnf("Received fewer scan responses (%v) than requested (%v)", responsesLength, requestCount)
		addSpanEvent(span, "Unmatched scan responses", oteltrace.WithAttributes(attribute.Int("requested", requestCount), attribute.Int("received", responsesLength)))
	}
	zs.Debugf("Responses: %v", responsesLength)
	if responsesLength == 0 {
		zs.Errorf("Multi-engine scan failed to produce results")
		http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
	} else {
		w.Header().Set(ContentTypeKey, ApplicationJSON)
		printResponse(w, "{"+strings.Join(responses, ",")+"}\n", zs, false)
	}
}

// validateHPSM checks if HPSM is enabled or not. If it's not and HPSM is detected. Fail the scan request.
func (s APIService) validateHPSM(contents []byte, zs *zap.SugaredLogger, w http.ResponseWriter) bool {
	if !s.config.Scanning.HPSMEnabled {
		if s.config.App.Trace {
			zs.Debugf("Checking if HPSM is present in the submitted WFP...")
		}
		if strings.Contains(string(contents), "hpsm=") {
			zs.Errorf("HPSM (hpsm=...) detected in WFPs and HPSM support is disabled")
			http.Error(w, "ERROR HPSM detected in WFP. HPSM is disabled", http.StatusForbidden)
			return false
		}
	}
	return true
}

// workerScan attempts to process all incoming scanning jobs and dumps the results into the subsequent results channel.
func (s APIService) workerScan(id string, jobs <-chan string, results chan<- string, flags, sbomType, sbomFile, dbName string, zs *zap.SugaredLogger) {
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
			result, err := s.scanWfp(job, flags, sbomType, sbomFile, dbName, zs)
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
func (s APIService) scanWfp(wfp, flags, sbomType, sbomFile, dbName string, zs *zap.SugaredLogger) (string, error) {
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
	if len(dbName) > 0 && dbName != "" { // we want to prefer request over the local config
		args = append(args, fmt.Sprintf("-n%s", dbName))
	} else if s.config.Scanning.ScanKbName != "" { // Set scanning KB name
		args = append(args, fmt.Sprintf("-n%s", s.config.Scanning.ScanKbName))
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
