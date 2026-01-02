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
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"os/exec"
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

const (
	sbomIdentify  = "identify"
	sbomBlackList = "blacklist"
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
	scanConfig := s.getConfigFromRequest(r, zs)
	// Check if we have an SBOM (and type) supplied
	var sbomFilename string

	if len(scanConfig.sbomFile) > 0 && len(scanConfig.sbomType) > 0 {
		if scanConfig.sbomType != sbomIdentify && scanConfig.sbomType != sbomBlackList { // Make sure we have a valid SBOM scan type
			zs.Errorf("Invalid SBOM type: %v", scanConfig.sbomType)
			http.Error(w, "ERROR invalid SBOM 'type' supplied", http.StatusBadRequest)
			return 0
		}
		tempFile, err := s.writeSbomFile(scanConfig.sbomFile, zs)
		if err != nil {
			http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
			return 0
		}
		if s.config.Scanning.TmpFileDelete {
			defer removeFile(tempFile, zs)
		}
		sbomFilename = tempFile.Name() // Save the SBOM filename
		zs.Debugf("Stored SBOM (%v) in %v", scanConfig.sbomType, sbomFilename)
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
		s.singleScan(string(contentsTrimmed), sbomFilename, scanConfig, zs, w)
	} else {
		s.scanThreaded(wfps, int(wfpCount), sbomFilename, scanConfig, zs, w, span)
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

// getConfigFromRequest extracts the form values from a request and returns the scanning configuration.
func (s APIService) getConfigFromRequest(r *http.Request, zs *zap.SugaredLogger) ScanningServiceConfig {
	flags := strings.TrimSpace(r.FormValue("flags"))    // Check form for scanning flags
	scanType := strings.TrimSpace(r.FormValue("type"))  // Check form for SBOM type
	sbom := strings.TrimSpace(r.FormValue("assets"))    // Check form for SBOM contents
	dbName := strings.TrimSpace(r.FormValue("db_name")) // Check form for db name

	// Fall back to headers if form values are empty
	if len(flags) == 0 {
		flags = strings.TrimSpace(r.Header.Get("flags"))
	}
	if len(scanType) == 0 {
		scanType = strings.TrimSpace(r.Header.Get("type"))
	}
	if len(sbom) == 0 {
		sbom = strings.TrimSpace(r.Header.Get("assets"))
	}
	if len(dbName) == 0 {
		dbName = strings.TrimSpace(r.Header.Get("db_name"))
	}

	scanSettings := strings.TrimSpace(r.Header.Get("scanoss-settings")) // Check header for scan settings

	if s.config.App.Trace {
		zs.Debugf("Header: %v, Form: %v, flags: %v, type: %v, assets: %v, db_name: %v, scanSettings: %v",
			r.Header, r.Form, flags, scanType, sbom, dbName, scanSettings)
	}

	// Create default configuration from server config
	scanConfig := DefaultScanningServiceConfig(s.config)

	// Decode scan settings from base64 if provided
	var decoded []byte
	if len(scanSettings) > 0 {
		var err error
		decoded, err = base64.StdEncoding.DecodeString(scanSettings)
		if err != nil {
			zs.Errorf("Error decoding scan settings from base64: %v", err)
			decoded = nil
		} else if s.config.App.Trace {
			zs.Debugf("Decoded scan settings: %s", string(decoded))
		}
	}

	return UpdateScanningServiceConfigDTO(zs, &scanConfig, flags, scanType, sbom, dbName, decoded)
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
func (s APIService) singleScan(wfp, sbomFile string, config ScanningServiceConfig, zs *zap.SugaredLogger, w http.ResponseWriter) {
	zs.Debugf("Single threaded scan...")
	result, err := s.scanWfp(wfp, sbomFile, config, zs)
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
func (s APIService) scanThreaded(wfps []string, wfpCount int, sbomFile string, config ScanningServiceConfig, zs *zap.SugaredLogger, w http.ResponseWriter, span oteltrace.Span) {
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
		go s.workerScan(fmt.Sprintf("%d_%s", i, uuid.New().String()), requests, results, sbomFile, config, zs)
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
func (s APIService) workerScan(id string, jobs <-chan string, results chan<- string, sbomFile string, config ScanningServiceConfig, zs *zap.SugaredLogger) {
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
			result, err := s.scanWfp(job, sbomFile, config, zs)
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
func (s APIService) scanWfp(wfp, sbomFile string, config ScanningServiceConfig, zs *zap.SugaredLogger) (string, error) {
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

	// Build command arguments
	var args []string
	if s.config.Scanning.ScanDebug {
		args = append(args, "-d") // Set debug mode
	}

	// Database name
	if len(config.dbName) > 0 {
		args = append(args, fmt.Sprintf("-n%s", config.dbName))
	}

	// Scanning flags
	if config.flags > 0 {
		args = append(args, fmt.Sprintf("-F %v", config.flags))
	}

	// SBOM configuration
	if len(sbomFile) > 0 && len(config.sbomType) > 0 {
		switch config.sbomType {
		case sbomIdentify:
			args = append(args, "-s")
		case sbomBlackList:
			args = append(args, "-b")
		default:
			args = append(args, "-s") // Default to identify
		}
		args = append(args, sbomFile)
	}

	// Ranking threshold (only if ranking is enabled and allowed)
	if config.rankingEnabled {
		args = append(args, fmt.Sprintf("-r%d", config.rankingThreshold))
	}

	// Minimum snippet hits
	if config.minSnippetHits > 0 {
		args = append(args, fmt.Sprintf("--min-snippet-hits=%d", config.minSnippetHits))
	}

	// Minimum snippet lines
	if config.minSnippetLines > 0 {
		args = append(args, fmt.Sprintf("--min-snippet-lines=%d", config.minSnippetLines))
	}

	// Snippet range tolerance
	if config.snippetRangeTolerance > 0 {
		args = append(args, fmt.Sprintf("--range-tolerance=%d", config.snippetRangeTolerance))
	}

	// Honour file extensions (not yet implemented in scanoss engine)
	if !config.honourFileExts {
		args = append(args, "--ignore-file-ext")
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
