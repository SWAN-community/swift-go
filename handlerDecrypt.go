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
	"errors"
	"fmt"
	"net/http"
)

// HandlerDecrypt takes a Services pointer and returns a HTTP handler used to
// decrypt the result of a storage operation provided in the raw query
// parameter to the return URL.
func HandlerDecrypt(s *Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check caller can access and parse the form variables.
		if s.getAccessAllowed(w, r) == false {
			returnAPIError(s, w,
				errors.New("Not authorized"),
				http.StatusUnauthorized)
			return
		}

		// Get the node associated with the request.
		n, err := s.GetAccessNodeForHost(r.Host)
		if err != nil {
			returnAPIError(s, w, err, http.StatusInternalServerError)
			return
		}

		// Decode the query string to form the byte array.
		in, err := base64.RawStdEncoding.DecodeString(r.Form.Get("encrypted"))
		if err != nil {
			returnAPIError(s, w, err, http.StatusBadRequest)
			return
		}

		// Decrypt the byte array using the node.
		d, err := n.Decrypt(in)
		if err != nil {
			returnAPIError(s, w, err, http.StatusBadRequest)
			return
		}
		if d == nil {
			returnAPIError(
				s,
				w,
				fmt.Errorf("Could not decrypt input"),
				http.StatusBadRequest)
			return
		}

		// Send the byte array.
		sendResponse(s, w, "application/octet-stream", d)
	}
}
