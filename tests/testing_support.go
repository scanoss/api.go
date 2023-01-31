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

// Package tests provides a set of integration tests to exercise the SCANOSS Scanning GO API
// These tests include:
// * Scanning
// * Attribution
// * File Contents
// * License Obligations
// * Health and Metrics
package tests

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
)

var hostPort = "http://localhost:5443"

// createMultipartFormData loads the given file and adds it to a multipart writer to be used when posting a request
func createMultipartFormData(fileFieldName, filePath string, fileName string, extraFormFields map[string]string) (b bytes.Buffer, w *multipart.Writer, err error) { //nolint:lll
	w = multipart.NewWriter(&b)
	var fw io.Writer
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Problem opening file: ", err)
		return
	}
	if fw, err = w.CreateFormFile(fileFieldName, fileName); err != nil {
		fmt.Println("Problem creating form file: ", err)
		return
	}
	if _, err = io.Copy(fw, file); err != nil {
		fmt.Println("Problem copying multipart file: ", err)
		return
	}
	// Add extra field data, if given
	for k, v := range extraFormFields {
		err = w.WriteField(k, v)
		if err != nil {
			return bytes.Buffer{}, nil, err
		}
	}
	err = w.Close()
	if err != nil {
		return bytes.Buffer{}, nil, err
	}
	return
}
