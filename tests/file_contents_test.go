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

type E2EContentsSuite struct {
	suite.Suite
}

func TestE2EContentsSuite(t *testing.T) {
	suite.Run(t, new(E2EContentsSuite))
}

func (s *E2EContentsSuite) TestHappyFileContents() {
	c := http.Client{}
	resp, err := c.Get(fmt.Sprintf("%v/file_contents/37f7cd1e657aa3c30ece35995b4c59e5", hostPort))
	if err != nil {
		s.Failf("an error was not expected when sending request.", "error: %v", err)
	}
	s.Equal(http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Failf("an error was not expected when reading response body.", "error: %v", err)
	}
	bodyStr := string(body)
	fmt.Println("Status: ", resp.StatusCode)
	fmt.Println("Type: ", resp.Header.Get("Content-Type"))
	fmt.Println("Body: ", bodyStr)
}
