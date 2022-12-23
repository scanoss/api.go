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

package config

import (
	"encoding/json"
	"fmt"
	"github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
	"os"
	"testing"
)

func TestServerConfig(t *testing.T) {
	appAddr := "localhost"
	err := os.Setenv("APP_ADDR", appAddr)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when creating new config instance", err)
	}
	cfg, err := NewServerConfig(nil)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when creating new config instance", err)
	}
	if cfg.App.Addr != appAddr {
		t.Errorf("App Addr '%v' doesn't match expected: %v", cfg.App.Addr, appAddr)
	}
	fmt.Printf("Server Config1: %+v\n", cfg)
	err = os.Unsetenv("APP_ADDR")
	if err != nil {
		fmt.Printf("Warning: Problem running Unsetenv: %v\n", err)
	}
	js, err := json.MarshalIndent(cfg, "", "   ")
	if err == nil {
		fmt.Printf("Config JSON:\n------\n")
		fmt.Println(string(js))
		fmt.Println("------")
	} else {
		fmt.Printf("Warning: Problem producing json: %v\n", err)
	}
}

func TestServerConfigDotEnv(t *testing.T) {
	err := os.Unsetenv("APP_ADDR")
	if err != nil {
		fmt.Printf("Warning: Problem runn Unsetenv: %v\n", err)
	}
	appAddr := "env-addr"
	var feeders []config.Feeder
	feeders = append(feeders, feeder.DotEnv{Path: "tests/dot-env"})
	cfg, err := NewServerConfig(feeders)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when creating new config instance", err)
	}
	if cfg.App.Addr != appAddr {
		t.Errorf("App Addr '%v' doesn't match expected: %v", cfg.App.Addr, appAddr)
	}
	fmt.Printf("Server Config2: %+v\n", cfg)
}

func TestServerConfigJson(t *testing.T) {
	err := os.Unsetenv("APP_ADDR")
	if err != nil {
		fmt.Printf("Warning: Problem runn Unsetenv: %v\n", err)
	}
	appAddr := "json-addr"
	var feeders []config.Feeder
	feeders = append(feeders, feeder.Json{Path: "tests/env.json"})
	cfg, err := NewServerConfig(feeders)
	if err != nil {
		t.Fatalf("an error '%s' was not expected when creating new config instance", err)
	}
	if cfg.App.Addr != appAddr {
		t.Errorf("App Addr '%v' doesn't match expected: %v", cfg.App.Addr, appAddr)
	}
	fmt.Printf("Server Config3: %+v\n", cfg)
}
