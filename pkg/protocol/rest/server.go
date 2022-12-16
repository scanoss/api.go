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

// Package rest handles all the REST communication for the Scanning Service
// It takes care of starting and stopping the listener, etc.
package rest

import (
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"net/http"
	"os"
	"os/signal"
	myconfig "scanoss.com/go-api/pkg/config"
	"scanoss.com/go-api/pkg/service"
	"syscall"
	"time"
)

// RunServer runs REST service to publish
func RunServer(config *myconfig.ServerConfig) error {
	// Check if TLS should be enabled or not
	var startTls = false
	if len(config.Tls.CertFile) > 0 && len(config.Tls.KeyFile) > 0 {
		cf, err := checkFile(config.Tls.CertFile)
		if err != nil || !cf {
			zlog.S.Errorf("Cert file is not accessible: %v", config.Tls.CertFile)
			if err != nil {
				return err
			} else {
				return fmt.Errorf("cert file not accesible: %v", config.Tls.CertFile)
			}
		}
		kf, err := checkFile(config.Tls.KeyFile)
		if err != nil || !kf {
			zlog.S.Errorf("Key file is not accessible: %v", config.Tls.KeyFile)
			if err != nil {
				return err
			} else {
				return fmt.Errorf("key file not accesible: %v", config.Tls.KeyFile)
			}
		}
		startTls = true
	}
	scanningService := service.NewScanningService(config)
	// Set up the endpoint routing
	router := mux.NewRouter().StrictSlash(true)
	srv := &http.Server{
		Handler: router,
		Addr:    fmt.Sprintf("%s:%s", config.App.Addr, config.App.Port),
	}
	router.HandleFunc("/api/scan/direct", scanningService.ScanDirect).Methods(http.MethodPost)
	router.HandleFunc("/api/file_contents/{md5}", scanningService.FileContents).Methods(http.MethodGet)
	// Open TCP port (in the background) and listen for requests
	go func() {
		if startTls {
			zlog.S.Infof("starting REST server with TLS on %v ...", srv.Addr)
			if err := srv.ListenAndServeTLS(config.Tls.CertFile, config.Tls.KeyFile); err != nil {
				if fmt.Sprintf("%s", err) != "http: Server closed" {
					zlog.S.Panicf("issue encountered when starting service: %v", err)
				}
			}
		} else {
			zlog.S.Infof("starting REST server on %v ...", srv.Addr)
			if err := srv.ListenAndServe(); err != nil {
				if fmt.Sprintf("%s", err) != "http: Server closed" {
					zlog.S.Panicf("issue encountered when starting service: %v", err)
				}
			}
		}
	}()
	// graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	<-c
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Set a deadline for gracefully shutting down
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zlog.S.Warnf("error shutting down server %s", err)
		return fmt.Errorf("issue encountered while shutting down service")
	} else {
		zlog.S.Info("server gracefully stopped")
	}
	return nil
}

// checkFile validates if the given file exists or not
func checkFile(filename string) (bool, error) {
	if len(filename) == 0 {
		return false, fmt.Errorf("no file specified to check")
	}
	fileDetails, err := os.Stat(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("file doest no exist")
		}
		return false, err
	}
	if fileDetails.IsDir() {
		return false, fmt.Errorf("is a directory and not a file")
	}
	return true, nil
}
