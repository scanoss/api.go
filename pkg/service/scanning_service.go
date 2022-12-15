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

package service

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gorilla/mux"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"io"
	"net/http"
	"os"
	"os/exec"
	myconfig "scanoss.com/wayuu2/pkg/config"
	"strings"
	"time"
)

type ScanningService struct {
	config *myconfig.ServerConfig
}

func NewScanningService(config *myconfig.ServerConfig) *ScanningService {
	return &ScanningService{config: config}
}

// FileContents handles retrieval of sources file for a client
func (s ScanningService) FileContents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	zlog.S.Infof("%v request from %v - %v", r.RemoteAddr, r.URL.Path, vars)
	if vars == nil || len(vars) == 0 {
		zlog.S.Errorf("Failed to retrieve request variables")
		http.Error(w, "ERROR no request variables submitted", http.StatusBadRequest)
	}
	md5, ok := vars["md5"]
	if !ok {
		zlog.S.Errorf("Failed to retrieve md5 request variable from: %v", vars)
		http.Error(w, "ERROR no md5 request variable submitted", http.StatusBadRequest)
	}
	zlog.S.Debugf("Retrieving contents for %v", md5)
	var args []string
	if s.config.Scanning.ScanDebug {
		args = append(args, "-d")
	}
	args = append(args, "-k", md5)
	zlog.S.Debugf("Executing %v %v", s.config.Scanning.ScanBinary, strings.Join(args, " "))
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // put a timeout on the scan execution
	defer cancel()
	output, err := exec.CommandContext(ctx, s.config.Scanning.ScanBinary, args...).Output()
	if err != nil {
		zlog.S.Errorf("Contents command (%v %v) failed: %v", s.config.Scanning.ScanBinary, args, err)
		http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
		return
	}
	if s.config.App.Trace {
		zlog.S.Debugf("Sending back contents: %v - '%v'", len(output), output)
	} else {
		zlog.S.Debugf("Sending back contents: %v", len(output))
	}
	_, err = fmt.Fprint(w, string(output))
	if err != nil {
		zlog.S.Errorf("Problem writing results back to client: %v", err)
	}
}

// ScanDirect handles WFP scanning requests from a client
func (s ScanningService) ScanDirect(w http.ResponseWriter, r *http.Request) {

	zlog.S.Infof("%v request from %v", r.RemoteAddr, r.URL.Path)
	file, _, err := r.FormFile("file")
	if err != nil {
		zlog.S.Errorf("Failed to retrieve WFP file: %v", err)
		http.Error(w, "ERROR receiving WFP file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	contents, err := io.ReadAll(file) // Load the file (WFP) contents into memory
	if err != nil {
		zlog.S.Errorf("Failed to retrieve WFP file contents: %v", err)
		http.Error(w, "ERROR receiving WFP file contents", http.StatusBadRequest)
		return
	}
	contentsTrimmed := bytes.TrimSpace(contents)
	if len(contentsTrimmed) == 0 {
		zlog.S.Errorf("No WFP contents to scan")
		http.Error(w, "ERROR no WFP contents supplied", http.StatusBadRequest)
		return
	}
	flags := r.Header.Get("flags")   // Scanning flags
	scanType := r.Header.Get("type") // SBOM type
	sbom := r.Header.Get("assets")

	zlog.S.Debugf("Header: %v, %v, %v, %v", r.Header, flags, scanType, sbom)

	var sbomFilename string
	if len(sbom) > 0 && len(scanType) > 0 {
		if scanType != "identify" && scanType != "blacklist" { // Make sure we have a valid SBOM scan type
			zlog.S.Errorf("Invalid SBOM type: %v", scanType)
			http.Error(w, "ERROR invalid SBOM 'type' supplied", http.StatusBadRequest)
			return
		}
		tempFile, err := os.CreateTemp(s.config.Scanning.WfpLoc, "sbom*.json")
		if err != nil {
			zlog.S.Errorf("Failed to create temporary SBOM file: %v", err)
			http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
			return
		}
		if s.config.Scanning.TmpFileDelete {
			defer removeFile(tempFile)
		}
		sbomFilename = tempFile.Name()
	}
	wfps := strings.Split(string(contentsTrimmed), "file=")
	wfpCount := len(wfps) - 1 // First entry in the array is empty (hence the -1)
	if wfpCount <= 0 {
		zlog.S.Errorf("No WFP (file=...) entries found to scan")
		http.Error(w, "ERROR no WFP file contents (file=...) supplied", http.StatusBadRequest)
		return
	}
	// Sort chunks by size
	//sort.SliceStable(wfps, func(i, j int) bool {  // TODO is this really needed?
	//	return len(wfps[i]) < len(wfps[j])
	//})

	zlog.S.Debugf("Need to scan %v files", wfpCount)
	// Only one worker selected, so send the whole WFP in a single command
	if s.config.Scanning.Workers <= 1 {
		zlog.S.Debugf("Single threaded scan...")
		result, err := s.scanWfp(string(contentsTrimmed), flags, scanType, sbomFilename)
		if err != nil {
			zlog.S.Errorf("Engine scan failed: %v", err)
			http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
		} else {
			zlog.S.Infof("Scan completed")
			fmt.Fprint(w, fmt.Sprintf("%s\n", strings.TrimSpace(result)))
		}
	} else {
		// Multiple workers, create input and output channels
		requests := make(chan string)
		results := make(chan string, wfpCount)
		numWorkers := s.config.Scanning.Workers
		if numWorkers > wfpCount {
			zlog.S.Debugf("Requested workers (%v) greater than WFPs (%v). Reducing number.", numWorkers, wfpCount)
			numWorkers = wfpCount
		}
		zlog.S.Debugf("Creating %v scanning workers...", numWorkers)
		// Create workers
		for i := 1; i <= numWorkers; i++ {
			go s.workerScan(i, requests, results, flags, scanType, sbomFilename)
		}
		requestCount := 0 // Count the number of actual requests sent
		for _, wfp := range wfps {
			if len(wfp) == 0 || wfp == "" { // Ignore empty requests
				continue
			}
			requests <- "file=" + wfp // Prepend the 'file=' back onto each WFP before submitting it
			requestCount++
		}
		close(requests) // No more requests. close the channel
		zlog.S.Debugf("Finished sending requests: %v", requestCount)
		var responses []string
		for i := 0; i < requestCount; i++ { // Get results for the number of requests sent
			if s.config.App.Trace {
				zlog.S.Debugf("Waiting for result %v", i)
			}
			result := <-results
			if s.config.App.Trace {
				zlog.S.Debugf("Result %v: %v", i, strings.TrimSpace(result))
			}
			if len(result) > 0 {
				responses = append(responses, result)
			}
		}
		close(results)
		zlog.S.Debugf("Responses: %v", len(responses))
		if len(responses) == 0 {
			zlog.S.Errorf("Multi-engine scan failed to produce results")
			http.Error(w, "ERROR engine scan failed", http.StatusInternalServerError)
		} else {
			fmt.Fprint(w, "{"+strings.Join(responses, ",")+"}\n")
		}
	}
}

// workerScan attempts to process all incoming scanning jobs and dumps the results into the subsequent results channel
func (s ScanningService) workerScan(id int, jobs <-chan string, results chan<- string, flags, sbomType, sbomFile string) {

	for job := range jobs {
		if s.config.App.Trace {
			zlog.S.Debugf("Scanning (%v): '%v'", id, job)
		} else {
			zlog.S.Debugf("Scanning (%v)", id)
		}
		if len(job) == 0 {
			zlog.S.Warnf("Nothing in the job request to scan. Ignoring")
			results <- ""
		} else {
			result, err := s.scanWfp(job, flags, sbomType, sbomFile)
			if s.config.App.Trace {
				zlog.S.Debugf("scan result (%v): %v, %v", id, result, err)
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
					zlog.S.Debugf("Saving result: '%v'", result)
				}
				results <- strings.TrimSpace(result)
			}
		}
	}
}

// scanWfp run the scanoss engine scan of the supplied WFP
func (s ScanningService) scanWfp(wfp, flags, sbomType, sbomFile string) (string, error) {

	if len(wfp) == 0 {
		zlog.S.Warnf("Nothing in the job request to scan. Ignoring")
		return "", fmt.Errorf("no wfp supplied to scan. ignoring")
	}
	tempFile, err := os.CreateTemp(s.config.Scanning.WfpLoc, "finger*.wfp")
	if err != nil {
		zlog.S.Errorf("Failed to create temporary file: %v", err)
		return "", fmt.Errorf("failed to create temporary WFP file")
	}
	if s.config.Scanning.TmpFileDelete {
		defer removeFile(tempFile)
	}
	zlog.S.Debugf("Using temporary file: %v", tempFile.Name())
	tempFile.WriteString(wfp + "\n")
	tempFile.Close()
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
	zlog.S.Debugf("Executing %v %v", s.config.Scanning.ScanBinary, strings.Join(args, " "))
	ctx, _ := context.WithTimeout(context.Background(), 120*time.Second) // put a timeout on the scan execution
	output, err := exec.CommandContext(ctx, s.config.Scanning.ScanBinary, args...).Output()
	if err != nil {
		zlog.S.Errorf("Scan command (%v %v) failed: %v", s.config.Scanning.ScanBinary, args, err)
		return "", fmt.Errorf("failed to scan WFP: %v", err)
	}
	return string(output), nil
}

// removeFile removes the given file and warns if anything went wrong
func removeFile(f *os.File) {
	err := os.Remove(f.Name())
	if err != nil {
		zlog.S.Warnf("Problem removing temp file: %v - %v", f.Name(), err)
	} else {
		zlog.S.Debugf("Removed temporary file: %v", f.Name())
	}
}
