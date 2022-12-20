// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2022 SCANOSS.COM
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

// Package service handles all the REST scanning requests
package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"go.uber.org/zap"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	myconfig "scanoss.com/go-api/pkg/config"
	"strings"
	"time"
)

const (
	ContentTypeKey  = "content-type"
	RequestIdKey    = "x-request-id"
	ResponseIdKey   = "x-response-id"
	ApplicationJson = "application/json"
	ReqLogKey       = "reqId"
)

type ScanningService struct {
	config *myconfig.ServerConfig
}

func NewScanningService(config *myconfig.ServerConfig) *ScanningService {
	return &ScanningService{config: config}
}

// FileContents handles retrieval of sources file for a client
func (s ScanningService) FileContents(w http.ResponseWriter, r *http.Request) {

	reqId := strings.TrimSpace(r.Header.Get(RequestIdKey)) // Request ID
	if len(reqId) == 0 {                                   // If no request id, create one
		reqId = uuid.NewString()
	}
	w.Header().Set(ResponseIdKey, reqId)
	zs := sugaredLogger(context.WithValue(r.Context(), ReqLogKey, reqId)) // Setup logger with context
	vars := mux.Vars(r)
	zs.Infof("%v request from %v - %v", r.URL.Path, r.RemoteAddr, vars)
	if vars == nil || len(vars) == 0 {
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
		http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
		return
	}
	if s.config.App.Trace {
		zs.Debugf("Sending back contents: %v - '%s'", len(output), output)
	} else {
		zs.Debugf("Sending back contents: %v", len(output))
	}
	printResponse(w, string(output), zs)
}

// ScanDirect handles WFP scanning requests from a client
func (s ScanningService) ScanDirect(w http.ResponseWriter, r *http.Request) {

	reqId := strings.TrimSpace(r.Header.Get(RequestIdKey)) // Request ID
	if len(reqId) == 0 {                                   // If no request id, create one
		reqId = uuid.NewString()
	}
	w.Header().Set(ResponseIdKey, reqId)
	zs := sugaredLogger(context.WithValue(r.Context(), ReqLogKey, reqId)) // Setup logger with context
	zs.Infof("%v request from %v", r.URL.Path, r.RemoteAddr)
	file, _, err := r.FormFile("file")
	if err != nil {
		zs.Errorf("Failed to retrieve WFP file: %v", err)
		http.Error(w, "ERROR receiving WFP file", http.StatusBadRequest)
		return
	}
	defer closeMultipartFile(file, zs)
	contents, err := io.ReadAll(file) // Load the file (WFP) contents into memory
	if err != nil {
		zs.Errorf("Failed to retrieve WFP file contents: %v", err)
		http.Error(w, "ERROR receiving WFP file contents", http.StatusBadRequest)
		return
	}
	contentsTrimmed := bytes.TrimSpace(contents)
	if len(contentsTrimmed) == 0 {
		zs.Errorf("No WFP contents to scan (%v - %v)", len(contents), contents)
		http.Error(w, "ERROR no WFP contents supplied", http.StatusBadRequest)
		return
	}
	flags := strings.TrimSpace(r.Header.Get("flags"))   // Scanning flags
	scanType := strings.TrimSpace(r.Header.Get("type")) // SBOM type
	sbom := strings.TrimSpace(r.Header.Get("assets"))   // SBOM contents
	if s.config.App.Trace {
		zs.Debugf("Header: %v, flags: %v, type: %v, assets: %v", r.Header, flags, scanType, sbom)
	}
	// Check if we have an SBOM (and type) supplied
	var sbomFilename string
	if len(sbom) > 0 && len(scanType) > 0 {
		if scanType != "identify" && scanType != "blacklist" { // Make sure we have a valid SBOM scan type
			zs.Errorf("Invalid SBOM type: %v", scanType)
			http.Error(w, "ERROR invalid SBOM 'type' supplied", http.StatusBadRequest)
			return
		}
		tempFile, err := os.CreateTemp(s.config.Scanning.WfpLoc, "sbom*.json")
		if err != nil {
			zs.Errorf("Failed to create temporary SBOM file: %v", err)
			http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
			return
		}
		_, err = tempFile.WriteString(sbom + "\n")
		if err != nil {
			zs.Errorf("Failed to write to temporary SBOM file: %v - %v", tempFile.Name(), err)
			http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
			return
		}
		closeFile(tempFile, zs)
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
	// Sort chunks by size
	//sort.SliceStable(wfps, func(i, j int) bool {  // TODO is this really needed?
	//	return len(wfps[i]) < len(wfps[j])
	//})

	zs.Debugf("Need to scan %v files", wfpCount)
	// Only one worker selected, so send the whole WFP in a single command
	if s.config.Scanning.Workers <= 1 {
		zs.Debugf("Single threaded scan...")
		result, err := s.scanWfp(string(contentsTrimmed), flags, scanType, sbomFilename, zs)
		if err != nil {
			zs.Errorf("Engine scan failed: %v", err)
			http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
		} else {
			zs.Infof("Scan completed")
			response := strings.TrimSpace(result)
			if len(response) == 0 {
				zs.Warnf("Nothing in the engine response")
				http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
			} else {
				w.Header().Set(ContentTypeKey, ApplicationJson)
				printResponse(w, fmt.Sprintf("%s\n", response), zs)
			}
		}
	} else {
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
			go s.workerScan(fmt.Sprintf("%d_%s", i, uuid.New().String()), requests, results, flags, scanType, sbomFilename, zs)
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
			//requests <- "file=" + wfp // Prepend the 'file=' back onto each WFP before submitting it
			//requestCount++
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
			w.Header().Set(ContentTypeKey, ApplicationJson)
			printResponse(w, "{"+strings.Join(responses, ",")+"}\n", zs)
		}
	}
}

// workerScan attempts to process all incoming scanning jobs and dumps the results into the subsequent results channel
func (s ScanningService) workerScan(id string, jobs <-chan string, results chan<- string, flags, sbomType, sbomFile string, zs *zap.SugaredLogger) {
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

// scanWfp run the scanoss engine scan of the supplied WFP
func (s ScanningService) scanWfp(wfp, flags, sbomType, sbomFile string, zs *zap.SugaredLogger) (string, error) {

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
		args = append(args, "-F", fmt.Sprintf("%v", s.config.Scanning.ScanFlags))
	} else if len(flags) > 0 && flags != "0" { // Set user supplied flags if enabled
		args = append(args, "-F", flags)
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

// printResponse sends the given response to the HTTP Response Writer
func printResponse(w http.ResponseWriter, resp string, zs *zap.SugaredLogger) {
	_, err := fmt.Fprint(w, resp)
	if err != nil {
		zs.Errorf("Failed to write HTTP response: %v", err)
	} else {
		zs.Infof("responded")
	}
}

// closeMultipartFile closes the given multipart file
func closeMultipartFile(f multipart.File, zs *zap.SugaredLogger) {
	err := f.Close()
	if err != nil {
		zs.Warnf("Problem closing multipart file: %v", err)
	}
}

// copyWfpTempFile copies a 'failed' WFP scan file to another file for later review
func (s ScanningService) copyWfpTempFile(filename string, zs *zap.SugaredLogger) {
	zs.Debugf("Backing up failed WFP file...")
	source, err := os.Open(filename)
	if err != nil {
		zs.Errorf("Failed to open file %v: %v", filename, err)
		return
	}
	tempFile, err := os.CreateTemp(s.config.Scanning.WfpLoc, "failed-finger*.wfp")
	if err != nil {
		zs.Errorf("Failed to create temporary file: %v", err)
		return
	}
	defer closeFile(tempFile, zs)
	_, err = io.Copy(tempFile, source)
	if err != nil {
		zs.Errorf("Failed to copy temporary file %v to %v: %v", filename, tempFile.Name(), err)
		return
	}
	zs.Warnf("Backed up failed WFP to: %v", tempFile.Name())
}

// closeFile closes the given file
func closeFile(f *os.File, zs *zap.SugaredLogger) {
	err := f.Close()
	if err != nil {
		zs.Warnf("Problem closing file: %v - %v", f.Name(), err)
	}
}

// removeFile removes the given file and warns if anything went wrong
func removeFile(f *os.File, zs *zap.SugaredLogger) {
	err := os.Remove(f.Name())
	if err != nil {
		zs.Warnf("Problem removing temp file: %v - %v", f.Name(), err)
	} else {
		zs.Debugf("Removed temporary file: %v", f.Name())
	}
}

// sugaredLogger returns a zap logger with as much context as possible
func sugaredLogger(ctx context.Context) *zap.SugaredLogger {
	newLogger := zlog.L
	if ctx != nil {
		if ctxRqId, ok := ctx.Value(ReqLogKey).(string); ok {
			newLogger = newLogger.With(zap.String(ReqLogKey, ctxRqId))
		}
		//if ctxSessionId, ok := ctx.Value(sessionIdKey).(string); ok {
		//	newLogger = newLogger.With(zap.String("sessionId", ctxSessionId))
		//}
	}
	return newLogger.Sugar()
}
