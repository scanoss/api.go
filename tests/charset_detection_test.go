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

type E2ECharsetDetectionSuite struct {
	suite.Suite
}

func TestE2ECharsetDetectionSuite(t *testing.T) {
	suite.Run(t, new(E2ECharsetDetectionSuite))
}

func (s *E2ECharsetDetectionSuite) TestFileContentsWithCharsetHeader() {
	// Test that the file_contents endpoint includes charset detection headers.
	c := http.Client{}
	resp, err := c.Get(fmt.Sprintf("%v/file_contents/37f7cd1e657aa3c30ece35995b4c59e5", hostPort))
	if err != nil {
		s.Failf("an error was not expected when sending request.", "error: %v", err)
	}
	s.Equal(http.StatusOK, resp.StatusCode)

	// Check Content-Type header includes charset.
	contentType := resp.Header.Get("Content-Type")
	s.NotEmpty(contentType, "Content-Type header should not be empty")
	s.Contains(contentType, "charset=", "Content-Type should include charset information")

	// Check X-Detected-Charset header.
	detectedCharset := resp.Header.Get("X-Detected-Charset")
	s.NotEmpty(detectedCharset, "X-Detected-Charset header should not be empty")

	// Check Content-Length header.
	contentLength := resp.Header.Get("Content-Length")
	s.NotEmpty(contentLength, "Content-Length header should not be empty")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Failf("an error was not expected when reading response body.", "error: %v", err)
	}
	bodyStr := string(body)

	fmt.Println("Status: ", resp.StatusCode)
	fmt.Println("Content-Type: ", contentType)
	fmt.Println("X-Detected-Charset: ", detectedCharset)
	fmt.Println("Content-Length: ", contentLength)
	fmt.Println("Body length: ", len(bodyStr))
}

func (s *E2ECharsetDetectionSuite) TestFileContentsWithInvalidMD5() {
	// Test that invalid MD5 returns appropriate error.
	c := http.Client{}
	resp, err := c.Get(fmt.Sprintf("%v/file_contents/invalid_md5_hash", hostPort))
	if err != nil {
		s.Failf("an error was not expected when sending request.", "error: %v", err)
	}
	// Should return an error status since the MD5 is invalid.
	s.Equal(http.StatusInternalServerError, resp.StatusCode)
}

func (s *E2ECharsetDetectionSuite) TestFileContentsWithMissingMD5() {
	// Test that missing MD5 parameter returns appropriate error.
	c := http.Client{}
	resp, err := c.Get(fmt.Sprintf("%v/file_contents/", hostPort))
	if err != nil {
		s.Failf("an error was not expected when sending request.", "error: %v", err)
	}
	// Should return not found since the path is incomplete.
	s.Equal(http.StatusNotFound, resp.StatusCode)
}