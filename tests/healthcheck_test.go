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

package tests

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	"io"
	"net/http"
	"testing"
)

type E2EHealthSuite struct {
	suite.Suite
}

func TestE2EHealthSuite(t *testing.T) {
	suite.Run(t, new(E2EHealthSuite))
}

func (s *E2EHealthSuite) TestHappyWelcomeMsg() {
	c := http.Client{}
	resp, err := c.Get(fmt.Sprintf("%v/", hostPort))
	if err != nil {
		s.Failf("an error was not expected when sending request.", "error: %v", err)
	}
	s.Equal(http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Failf("an error was not expected when reading response body.", "error: %v", err)
	}
	bodyStr := string(body)
	s.JSONEq(`{"msg": "Hello from the SCANOSS Scanning API"}`, bodyStr)
	fmt.Println("Status: ", resp.StatusCode)
	fmt.Println("Type: ", resp.Header.Get("Content-Type"))
	fmt.Println("Body: ", bodyStr)

	resp2, err := c.Head(fmt.Sprintf("%v/", hostPort))
	if err != nil {
		s.Failf("an error was not expected when sending request.", "error: %v", err)
	}
	s.Equal(http.StatusOK, resp2.StatusCode)
	fmt.Println("Status: ", resp.StatusCode)
}

func (s *E2EHealthSuite) TestHappyHealthcheck() {
	c := http.Client{}
	resp, err := c.Get(fmt.Sprintf("%v/health-check", hostPort))
	if err != nil {
		s.Failf("an error was not expected when sending request.", "error: %v", err)
	}
	s.Equal(http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Failf("an error was not expected when reading response body.", "error: %v", err)
	}
	bodyStr := string(body)
	s.JSONEq(`{"alive": true}`, bodyStr)
	fmt.Println("Status: ", resp.StatusCode)
	fmt.Println("Type: ", resp.Header.Get("Content-Type"))
	fmt.Println("Body: ", bodyStr)

	// Test the HEAD call also
	resp2, err := c.Head(fmt.Sprintf("%v/health-check", hostPort))
	if err != nil {
		s.Failf("an error was not expected when sending request.", "error: %v", err)
	}
	s.Equal(http.StatusOK, resp2.StatusCode)
	fmt.Println("Status: ", resp.StatusCode)
}

func (s *E2EHealthSuite) TestMetrics() {
	c := http.Client{}

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "Test invalid",
			input: "invalid",
			want:  http.StatusBadRequest,
		},
		{
			name:  "Test wrong request type",
			input: "nothing",
			want:  http.StatusBadRequest,
		},
		{
			name:  "Test goroutines",
			input: "goroutines",
			want:  http.StatusOK,
		},
		{
			name:  "Test heap",
			input: "heap",
			want:  http.StatusOK,
		},
		{
			name:  "Test requests",
			input: "requests",
			want:  http.StatusOK,
		},
		{
			name:  "Test all",
			input: "all",
			want:  http.StatusOK,
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			resp, err := c.Get(fmt.Sprintf("%v/metrics/%v", hostPort, test.input))
			if err != nil {
				s.Failf("an error was not expected when sending request.", "error: %v", err)
			}
			s.Equal(test.want, resp.StatusCode)
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				s.Failf("an error was not expected when reading response body.", "error: %v", err)
			}
			fmt.Println("Status: ", resp.StatusCode)
			fmt.Println("Type: ", resp.Header.Get("Content-Type"))
			fmt.Println("Body: ", string(body))
		})
	}
}
