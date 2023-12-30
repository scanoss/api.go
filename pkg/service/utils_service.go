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
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"

	"go.opentelemetry.io/otel/codes"

	oteltrace "go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/metric"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap/zapcore"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"go.uber.org/zap"
	myconfig "scanoss.com/go-api/pkg/config"
)

// Constants for use through the API services.
const (
	ContentTypeKey  = "content-type"
	RequestIDKey    = "x-request-id"
	ResponseIDKey   = "x-response-id"
	ApplicationJSON = "application/json"
	TextPlain       = "text/plain"
	ReqLogKey       = "reqId"
	SpanLogKey      = "span_id"
	TraceLogKey     = "trace_id"
)

// RequestContextKey Request ID Key name for using with Context.
type RequestContextKey struct{}

// SpanContextKey Span ID Key name for using with Context.
type SpanContextKey struct{}

// TraceContextKey Trace ID Key name for using with Context.
type TraceContextKey struct{}

// APIService details.
type APIService struct {
	config *myconfig.ServerConfig
}

// NewAPIService instantiates an API Service instance for servicing the API requests.
func NewAPIService(config *myconfig.ServerConfig) *APIService {
	setupMetrics()
	return &APIService{config: config}
}

// Structure for counting the total number of requests processed.
type counterStruct struct {
	mu     sync.Mutex
	values map[string]int64
}

var counters = counterStruct{
	values: make(map[string]int64),
}

// Structure for storing OTEL metrics.
type metricsCounters struct {
	scanFileCounter           metric.Int64Counter
	scanSizeCounter           metric.Int64Counter
	fileContentsCounter       metric.Int64Counter
	licenseDetailsCounter     metric.Int64Counter
	attributionDetailsCounter metric.Int64Counter
	scanHistogram             metric.Int64Histogram // milliseconds
	scanFileHistogram         metric.Int64Histogram // milliseconds
	scanHistogramSec          metric.Float64Histogram
	scanFileHistogramSec      metric.Float64Histogram
}

var oltpMetrics = metricsCounters{}

// setupMetrics configures all the metrics recorders for the platform.
func setupMetrics() {
	meter := otel.Meter("scanoss.com/go-api")
	oltpMetrics.scanFileCounter, _ = meter.Int64Counter("scanoss-api.scan.file_count", metric.WithDescription("The number of scan request files received"))
	oltpMetrics.scanSizeCounter, _ = meter.Int64Counter("scanoss-api.scan.file_size", metric.WithDescription("The size of scan request files received"))
	oltpMetrics.fileContentsCounter, _ = meter.Int64Counter("scanoss-api.contents.req_count", metric.WithDescription("The number of file contents requests received"))
	oltpMetrics.licenseDetailsCounter, _ = meter.Int64Counter("scanoss-api.license.req_count", metric.WithDescription("The number of license details requests received"))
	oltpMetrics.attributionDetailsCounter, _ = meter.Int64Counter("scanoss-api.attribution.req_count", metric.WithDescription("The number of license attribution requests received"))
	oltpMetrics.scanHistogram, _ = meter.Int64Histogram("scanoss-api.scan.req_time", metric.WithDescription("The time taken to run a scan request (ms)"))
	oltpMetrics.scanFileHistogram, _ = meter.Int64Histogram("scanoss-api.scan.file_time", metric.WithDescription("The average time taken to scan a single file in a request (ms)"))
	oltpMetrics.scanHistogramSec, _ = meter.Float64Histogram("scanoss-api.scan.req_time_sec", metric.WithDescription("The time taken to run a scan request (seconds)"))
	oltpMetrics.scanFileHistogramSec, _ = meter.Float64Histogram("scanoss-api.scan.file_time_sec", metric.WithDescription("Average time to scan a single file per request (seconds)"))
}

// incRequest increments the count for the given request type.
func (c *counterStruct) incRequest(key string) {
	c.incRequestAmount(key, 1)
}

// incRequestAmount increments the count for the given request type by the given amount.
func (c *counterStruct) incRequestAmount(key string, amount int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] += amount
}

// WelcomeMsg responds with a welcome to the SCANOSS API.
func WelcomeMsg(w http.ResponseWriter, r *http.Request) {
	zlog.S.Debugf("%v request from %v", r.URL.Path, r.RemoteAddr)
	w.Header().Set(ContentTypeKey, ApplicationJSON)
	w.WriteHeader(http.StatusOK)
	printResponse(w, fmt.Sprintln(`{"msg": "Hello from the SCANOSS Scanning API"}`), zlog.S, true)
}

// HealthCheck responds with the health of the service.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	zlog.S.Debugf("%v request from %v", r.URL.Path, r.RemoteAddr)
	w.Header().Set(ContentTypeKey, ApplicationJSON)
	w.WriteHeader(http.StatusOK)
	printResponse(w, fmt.Sprintln(`{"alive": true}`), zlog.S, true)
}

// HeadResponse responds with the HEAD OK Status for the requested path.
func HeadResponse(w http.ResponseWriter, r *http.Request) {
	zlog.S.Debugf("%v HEAD request from %v", r.URL.Path, r.RemoteAddr)
	w.WriteHeader(http.StatusOK)
}

// MetricsHandler responds with the metrics for the requested type.
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	zlog.S.Debugf("%v request from %v - %v", r.URL.Path, r.RemoteAddr, vars)
	if len(vars) == 0 {
		zlog.S.Errorf("Failed to retrieve request variables")
		http.Error(w, "ERROR no request variables submitted", http.StatusBadRequest)
		return
	}
	mType, ok := vars["type"]
	if !ok {
		zlog.S.Errorf("Failed to retrieve type request variable from: %v", vars)
		http.Error(w, "ERROR no type request variable submitted", http.StatusBadRequest)
		return
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
		return fmt.Sprintf("{\"scans\": %v, \"files\": %v, \"file_contents\": %v, \"attribution\": %v, \"license_details\": %v}",
			counters.values["scan"], counters.values["files"], counters.values["file_contents"], counters.values["attribution"],
			counters.values["license_details"])
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
	w.Header().Set(ContentTypeKey, ApplicationJSON)
	w.WriteHeader(responseStatus)
	zlog.S.Infof("Metrics: %v", responseString)
	printResponse(w, responseString+"\n", zlog.S, true)
}

// printResponse sends the given response to the HTTP Response Writer.
func printResponse(w http.ResponseWriter, resp string, zs *zap.SugaredLogger, silent bool) {
	_, err := fmt.Fprint(w, resp)
	if err != nil {
		zs.Errorf("Failed to write HTTP response: %v", err)
	} else if !silent {
		zs.Infof("responded")
	}
}

// closeMultipartFile closes the given multipart file.
func closeMultipartFile(f multipart.File, zs *zap.SugaredLogger) {
	err := f.Close()
	if err != nil {
		zs.Warnf("Problem closing multipart file: %v", err)
	}
}

// getFormFile attempts to get the contents of the form file from the supplied request.
func (s APIService) getFormFile(r *http.Request, zs *zap.SugaredLogger, formType string) ([]byte, error) {
	var contents []byte
	var err error
	formFiles := []string{"file", "filename"}
	for _, fName := range formFiles { // Check for the contents in 'file' and 'filename'
		var file multipart.File
		file, _, err = r.FormFile(fName)
		if err != nil {
			zs.Infof("Cannot retrieve %s Form File: %v - %v. Trying an alternative name...", formType, fName, err)
			continue
		}
		contents, err = io.ReadAll(file) // Load the file contents into memory
		closeMultipartFile(file, zs)
		if err == nil {
			break // We have successfully gotten the file contents
		} else {
			zs.Infof("Cannot retrieve %s Form File (%v) contents: %v. Trying an alternative name...", formType, file, err)
		}
	}
	// Make sure we have actually got a WFP file to scan
	if err != nil {
		zs.Errorf("Failed to retrieve WFP file contents (using %v): %v", formFiles, err)
		return contents, err
	}
	return contents, nil
}

// copyWfpTempFile copies a 'failed' WFP scan file to another file for later review.
func (s APIService) copyWfpTempFile(filename string, zs *zap.SugaredLogger) string {
	zs.Debugf("Backing up failed WFP file...")
	source, err := os.Open(filename)
	if err != nil {
		zs.Errorf("Failed to open file %v: %v", filename, err)
		return ""
	}
	tempFile, err := os.CreateTemp(s.config.Scanning.WfpLoc, "failed-finger*.wfp")
	if err != nil {
		zs.Errorf("Failed to create temporary file: %v", err)
		return ""
	}
	defer closeFile(tempFile, zs)
	_, err = io.Copy(tempFile, source)
	if err != nil {
		zs.Errorf("Failed to copy temporary file %v to %v: %v", filename, tempFile.Name(), err)
		return ""
	}
	zs.Warnf("Backed up failed WFP to: %v", tempFile.Name())
	return tempFile.Name()
}

// closeFile closes the given file.
func closeFile(f *os.File, zs *zap.SugaredLogger) {
	if f != nil {
		err := f.Close()
		if err != nil {
			zs.Warnf("Problem closing file: %v - %v", f.Name(), err)
		}
	}
}

// removeFile removes the given file and warns if anything went wrong.
func removeFile(f *os.File, zs *zap.SugaredLogger) {
	if f != nil {
		err := os.Remove(f.Name())
		if err != nil {
			zs.Warnf("Problem removing temp file: %v - %v", f.Name(), err)
		} else {
			zs.Debugf("Removed temporary file: %v", f.Name())
		}
	}
}

// getReqID extracts the request id from the header and if not creates one and returns it.
func getReqID(r *http.Request) string {
	reqID := strings.TrimSpace(r.Header.Get(RequestIDKey)) // Request ID
	if len(reqID) == 0 {                                   // If no request id, create one
		reqID = uuid.NewString()
	}
	return reqID
}

// sugaredLogger returns a zap logger with as much context as possible.
func sugaredLogger(ctx context.Context) *zap.SugaredLogger {
	newLogger := zlog.L
	if ctx != nil {
		var fields []zapcore.Field
		if ctxRqID, ok := ctx.Value(RequestContextKey{}).(string); ok {
			fields = append(fields, zap.String(ReqLogKey, ctxRqID))
		}
		if ctxSpanID, ok := ctx.Value(SpanContextKey{}).(string); ok {
			fields = append(fields, zap.String(SpanLogKey, ctxSpanID))
		}
		if ctxTraceID, ok := ctx.Value(TraceContextKey{}).(string); ok {
			fields = append(fields, zap.String(TraceLogKey, ctxTraceID))
		}
		if len(fields) > 0 {
			newLogger = newLogger.With(fields...)
		}
	}
	return newLogger.Sugar()
}

// requestContext returns a new context with a Request ID and optional Span & Trace IDs.
func requestContext(ctx context.Context, reqID, spanID, traceID string) context.Context {
	var logContext context.Context
	logContext = context.WithValue(ctx, RequestContextKey{}, reqID)
	if len(spanID) > 0 {
		logContext = context.WithValue(logContext, SpanContextKey{}, spanID)
	}
	if len(traceID) > 0 {
		logContext = context.WithValue(logContext, TraceContextKey{}, traceID)
	}
	return logContext
}

// getSpan retrieves the span from the context (if set) and adds tracing data to the context for logging.
func getSpan(context context.Context, reqID string) (oteltrace.Span, context.Context) {
	span := oteltrace.SpanFromContext(context)
	spanContext := oteltrace.SpanFromContext(context).SpanContext()
	logContext := requestContext(context, reqID, spanContext.SpanID().String(), spanContext.TraceID().String())
	return span, logContext
}

// setSpanError sets the span status to error with a message.
func setSpanError(span oteltrace.Span, msg string) {
	if span != nil {
		span.SetStatus(codes.Error, msg)
	}
}

// addSpanEvent adds the given event to the specified span.
func addSpanEvent(span oteltrace.Span, msg string, options ...oteltrace.EventOption) {
	if span != nil {
		span.AddEvent(msg, options...)
	}
}
