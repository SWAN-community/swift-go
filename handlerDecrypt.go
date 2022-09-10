/* ****************************************************************************
 * Copyright 2020 51 Degrees Mobile Experts Limited (51degrees.com)
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 * License for the specific language governing permissions and limitations
 * under the License.
 * ***************************************************************************/

package swift

import (
	"encoding/base64"
	"net/http"

	"github.com/SWAN-community/common-go"
)

// HandlerDecrypt takes a Services pointer and returns a HTTP handler used to
// decrypt the result of a storage operation provided in the raw query
// parameter to the return URL.
func HandlerDecrypt(s *Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check caller can access and parse the form variables.
		if s.access.GetAllowedHttp(w, r) == false {
			return
		}

		// Get the node associated with the request.
		n, err := s.GetAccessNodeForHost(r.Host)
		if err != nil {
			common.ReturnServerError(w, err)
			return
		}

		// Decode the query string to form the byte array.
		in, err := base64.StdEncoding.DecodeString(r.Form.Get("encrypted"))
		if err != nil {
			common.ReturnApplicationError(w, &common.HttpError{
				Message: "bad data",
				Code:    http.StatusBadRequest})
			return
		}

		// Decrypt the byte array using the node.
		d, err := n.decode(in)
		if err != nil {
			common.ReturnApplicationError(w, &common.HttpError{
				Message: "bad data",
				Code:    http.StatusBadRequest})
			return
		}
		if d == nil {
			common.ReturnApplicationError(w, &common.HttpError{
				Message: "could not decrypt input",
				Code:    http.StatusBadRequest})
			return
		}

		// Send the byte array.
		common.SendByteArray(w, d)
	}
}
