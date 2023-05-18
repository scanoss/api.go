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

// Package rest handles all the REST communication for the Scanning Service
// It takes care of starting and stopping the listener, etc.
package rest

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/jpillora/ipfilter"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	myconfig "scanoss.com/go-api/pkg/config"
	"scanoss.com/go-api/pkg/service"
)

// RunServer runs REST service to publish.
func RunServer(config *myconfig.ServerConfig) error {
	// Check if TLS should be enabled or not
	startTLS, err := checkTLS(config)
	if err != nil {
		return err
	}
	allowedIPs, deniedIPs, err := loadFiltering(config)
	if err != nil {
		return err
	}
	apiService := service.NewAPIService(config)
	if err := apiService.TestEngine(); err != nil {
		zlog.S.Warnf("Scanning engine test failed. Scan requests are likely to fail.")
		zlog.S.Warnf("Please make sure that %v is accessible", config.Scanning.ScanBinary)
	}
	// Set up the endpoint routing
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", service.WelcomeMsg).Methods(http.MethodGet)
	router.HandleFunc("/api/", service.WelcomeMsg).Methods(http.MethodGet)
	router.HandleFunc("/api/health", service.HealthCheck).Methods(http.MethodGet)
	router.HandleFunc("/api/health-check", service.HealthCheck).Methods(http.MethodGet)
	router.HandleFunc("/api/metrics/{type}", service.MetricsHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/file_contents/{md5}", apiService.FileContents).Methods(http.MethodGet)
	router.HandleFunc("/api/license/obligations/{license}", apiService.LicenseDetails).Methods(http.MethodGet)
	router.HandleFunc("/api/scan/direct", apiService.ScanDirect).Methods(http.MethodPost)
	router.HandleFunc("/api/sbom/attribution", apiService.SbomAttribution).Methods(http.MethodPost)
	srv := &http.Server{
		Handler:           router,
		Addr:              fmt.Sprintf("%s:%s", config.App.Addr, config.App.Port),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if len(allowedIPs) > 0 || len(deniedIPs) > 0 { // Configure the list of allowed/denied IPs to connect
		zlog.S.Debugf("Filtering requests by allowed: %v, denied: %v, block-by-default: %v", allowedIPs, deniedIPs, config.Filtering.BlockByDefault)
		handler := ipfilter.Wrap(router, ipfilter.Options{AllowedIPs: allowedIPs, BlockedIPs: deniedIPs,
			BlockByDefault: config.Filtering.BlockByDefault, TrustProxy: config.Filtering.TrustProxy,
		})
		srv.Handler = handler // assign the filtered handler
	}
	// Open TCP port (in the background) and listen for requests
	go func() {
		var httpErr error
		if startTLS {
			zlog.S.Infof("starting REST server with TLS on %v ...", srv.Addr)
			loadTLSConfig(config, srv)
			httpErr = srv.ListenAndServeTLS("", "")
		} else {
			zlog.S.Infof("starting REST server on %v ...", srv.Addr)
			httpErr = srv.ListenAndServe()
		}
		if httpErr != nil && fmt.Sprintf("%s", httpErr) != "http: Server closed" {
			zlog.S.Panicf("issue encountered when starting service: %v", httpErr)
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

// loadTLSConfig loads the TLS config into memory (decrypting if required) and updates the Server config.
func loadTLSConfig(config *myconfig.ServerConfig, srv *http.Server) {
	pemBlocks := loadCertFile(config)
	pkey := loadPrivateKey(config)
	c, err := tls.X509KeyPair(pem.EncodeToMemory(pemBlocks[0]), pkey)
	if err != nil {
		zlog.S.Panicf("Failed to load TLS key pair (%v - %v): %v", config.TLS.KeyFile, config.TLS.CertFile, err)
	}
	cfg := &tls.Config{
		MinVersion:       tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
		Certificates: []tls.Certificate{c},
	}
	// tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	// tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	// tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	// tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	// tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
	srv.TLSConfig = cfg
	srv.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0)
}

// loadCertFile load the certificate file into memory to use for hosting a TLS endpoint.
func loadCertFile(config *myconfig.ServerConfig) []*pem.Block {
	b, err := os.ReadFile(config.TLS.CertFile)
	if err != nil {
		zlog.S.Panicf("Failed to load Cert file - %v: %v", config.TLS.CertFile, err)
	}
	var pemBlocks []*pem.Block
	var v *pem.Block
	for {
		v, b = pem.Decode(b)
		if v == nil {
			break
		}
		if v.Type != "RSA PRIVATE KEY" && v.Type != "PRIVATE KEY" {
			pemBlocks = append(pemBlocks, v)
		} else {
			zlog.S.Warnf("Unknown certificate type (%v): %v", config.TLS.CertFile, v.Type)
		}
	}
	return pemBlocks
}

// loadPrivateKey loads the private key from file and attempt to decrypt it (if it's encrypted).
func loadPrivateKey(config *myconfig.ServerConfig) []byte {
	var v *pem.Block
	var pkey []byte
	b, err := os.ReadFile(config.TLS.KeyFile)
	if err != nil {
		zlog.S.Panicf("Failed to load Key file - %v: %v", config.TLS.KeyFile, err)
	}
	for {
		v, b = pem.Decode(b)
		if v == nil {
			break
		}
		if v.Type == "RSA PRIVATE KEY" || v.Type == "PRIVATE KEY" {
			zlog.S.Debugf("Private Key: %v - %v", v.Type, v.Headers)
			// pvt, err := openssl.LoadPrivateKeyFromPEMWithPassword(encryptedPEM, passPhrase)
			//nolint:staticcheck
			if x509.IsEncryptedPEMBlock(v) {
				if len(config.TLS.Password) == 0 {
					zlog.S.Panicf("Need to configure TLS Password to decrypt encrypted Key file: %v", config.TLS.KeyFile)
				}
				zlog.S.Infof("Decrypting key...")
				//nolint:staticcheck
				pkey, err = x509.DecryptPEMBlock(v, []byte(config.TLS.Password))
				if err != nil {
					zlog.S.Panicf("Failed to decrypt Key File (%v): %v", config.TLS.KeyFile, err)
				}
				pkey = pem.EncodeToMemory(&pem.Block{
					Type:  v.Type,
					Bytes: pkey,
				})
			} else {
				pkey = pem.EncodeToMemory(v)
			}
		} else {
			zlog.S.Warnf("Unexpected certificate type (%v): %v", config.TLS.KeyFile, v.Type)
		}
	}
	return pkey
}

// checkFile validates if the given file exists or not.
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

// checkTLS tests if TLS should be enabled or not.
func checkTLS(config *myconfig.ServerConfig) (bool, error) {
	var startTLS = false
	if len(config.TLS.CertFile) > 0 && len(config.TLS.KeyFile) > 0 {
		cf, err := checkFile(config.TLS.CertFile)
		if err != nil || !cf {
			zlog.S.Errorf("Cert file is not accessible: %v", config.TLS.CertFile)
			if err != nil {
				return false, err
			} else {
				return false, fmt.Errorf("cert file not accesible: %v", config.TLS.CertFile)
			}
		}
		kf, err := checkFile(config.TLS.KeyFile)
		if err != nil || !kf {
			zlog.S.Errorf("Key file is not accessible: %v", config.TLS.KeyFile)
			if err != nil {
				return false, err
			} else {
				return false, fmt.Errorf("key file not accesible: %v", config.TLS.KeyFile)
			}
		}
		startTLS = true
	}
	return startTLS, nil
}

// loadFiltering loads the IP filtering options if available.
func loadFiltering(config *myconfig.ServerConfig) ([]string, []string, error) {
	var allowedIPs []string
	if len(config.Filtering.AllowListFile) > 0 {
		cf, err := checkFile(config.Filtering.AllowListFile)
		if err != nil || !cf {
			zlog.S.Errorf("Allow List file is not accessible: %v", config.Filtering.AllowListFile)
			if err != nil {
				return nil, nil, err
			} else {
				return nil, nil, fmt.Errorf("allow list file not accesible: %v", config.Filtering.AllowListFile)
			}
		}
		allowedIPs, err = myconfig.LoadFile(config.Filtering.AllowListFile)
		if err != nil {
			return nil, nil, err
		}
	}
	var deniedIPs []string
	if len(config.Filtering.DenyListFile) > 0 {
		cf, err := checkFile(config.Filtering.DenyListFile)
		if err != nil || !cf {
			zlog.S.Errorf("Deny List file is not accessible: %v", config.Filtering.DenyListFile)
			if err != nil {
				return nil, nil, err
			} else {
				return nil, nil, fmt.Errorf("deny list file not accesible: %v", config.Filtering.DenyListFile)
			}
		}
		deniedIPs, err = myconfig.LoadFile(config.Filtering.DenyListFile)
		if err != nil {
			return nil, nil, err
		}
	}
	return allowedIPs, deniedIPs, nil
}
