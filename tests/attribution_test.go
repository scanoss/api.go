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

type E2EAttributionSuite struct {
	suite.Suite
}

func TestE2EAttributionSuite(t *testing.T) {
	suite.Run(t, new(E2EAttributionSuite))
}

func (s *E2EAttributionSuite) TestAttribution() {
	c := http.Client{}
	tests := []struct {
		name        string
		fieldName   string
		filename    string
		shortName   string
		extraFields map[string]string
		want        int
	}{
		{
			name:        "Test Empty SBOM",
			filename:    "../pkg/service/tests/software-bom-empty.json",
			shortName:   "software-bom-empty.json",
			extraFields: map[string]string{},
			want:        http.StatusBadRequest,
		},
		{
			name:        "Test Invalid Field Name",
			fieldName:   "wrong-name",
			filename:    "../pkg/service/tests/software-bom.json",
			shortName:   "software-bom.json",
			extraFields: map[string]string{},
			want:        http.StatusBadRequest,
		},
		{
			name:        "Test Valid SBOM",
			filename:    "../pkg/service/tests/software-bom.json",
			shortName:   "software-bom.json",
			extraFields: map[string]string{},
			want:        http.StatusOK,
		},
	}
	for _, test := range tests {
		s.Run(test.name, func() {
			fieldName := "file"
			if len(test.fieldName) > 0 {
				fieldName = test.fieldName
			}
			b, w, err := createMultipartFormData(fieldName, test.filename, test.shortName, test.extraFields)
			if err != nil {
				s.Failf("an error was not creating multipart form data.", "error: %v", err)
			}
			req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%v/sbom/attribution", hostPort), &b)
			if err != nil {
				s.Failf("an error was not creating request.", "error: %v", err)
			}
			req.Header.Set("Content-Type", w.FormDataContentType())
			resp, err := c.Do(req)
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
