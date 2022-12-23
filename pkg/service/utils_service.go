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
	"fmt"
	"github.com/gorilla/mux"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"net/http"
	"runtime"
	"sync"
)

// Structure for counting the total number of requests processed
type counterStruct struct {
	mu     sync.Mutex
	values map[string]int64
}

var counters = counterStruct{
	values: make(map[string]int64),
}

// incRequest increments the count for the given request type
func (c *counterStruct) incRequest(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key]++
}

// HealthCheck responds with the health of the service
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	zlog.S.Debugf("%v request from %v", r.URL.Path, r.RemoteAddr)
	w.Header().Set(ContentTypeKey, ApplicationJson)
	w.WriteHeader(http.StatusOK)
	_, err := fmt.Fprintln(w, `{"alive": true}`)
	if err != nil {
		zlog.S.Errorf("Failed to write HTTP response: %v", err)
	}
}

// MetricsHandler responds with the metrics for the requested type
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	zlog.S.Debugf("%v request from %v - %v", r.URL.Path, r.RemoteAddr, vars)
	if vars == nil || len(vars) == 0 {
		zlog.S.Errorf("Failed to retrieve request variables")
		http.Error(w, "ERROR no request variables submitted", http.StatusBadRequest)
	}
	mType, ok := vars["type"]
	if !ok {
		zlog.S.Errorf("Failed to retrieve type request variable from: %v", vars)
		http.Error(w, "ERROR no type request variable submitted", http.StatusBadRequest)
	}
	// Convert bytes to megabytes
	bToMb := func(b uint64) float64 {
		return float64(b) / 1024 / 1024
	}
	// Get the heap details
	heap := func() string {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return fmt.Sprintf("{\"alloc\": \"%.2f MiB\", \"total-alloc\": \"%.2f MiB\", \"sys\": \"%.2f MiB\"}", bToMb(m.Alloc), bToMb(m.TotalAlloc), bToMb(m.Sys))
	}
	reqCount := func() string {
		return fmt.Sprintf("{\"scan\": %v, \"file_contents\": %v}", counters.values["scan"], counters.values["file_contents"])
	}
	// Get the number of goroutines
	routines := func() string {
		return fmt.Sprintf("{\"count\": %v}", runtime.NumGoroutine())
	}
	var responseString string
	responseStatus := http.StatusOK
	switch mType {
	case "goroutines":
		responseString = routines()
	case "heap":
		responseString = heap()
	case "requests":
		responseString = reqCount()
	case "all":
		responseString = fmt.Sprintf("{\"goroutines\": %s, \"heap\": %s, \"requests\": %s}", routines(), heap(), reqCount())
	default:
		responseString = fmt.Sprintf("{\"error\": \"Unknown request type: %v. Supported: goroutines, heap, requests, all\"}", mType)
		responseStatus = http.StatusBadRequest
	}
	w.Header().Set(ContentTypeKey, ApplicationJson)
	w.WriteHeader(responseStatus)
	zlog.S.Infof("Metrics: %v", responseString)
	_, err := fmt.Fprint(w, responseString+"\n")
	if err != nil {
		zlog.S.Errorf("Failed to write HTTP response: %v", err)
	}
}
