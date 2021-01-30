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
	"fmt"
	"net/http"
)

// HandlerEncrypt takes a Services pointer and returns a HTTP handler used to
// encrypt the result of a storage operation ready to be provided to the return
// URL.
func HandlerEncrypt(s *Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		err := r.ParseForm()
		if err != nil {
			returnAPIError(s, w, err, http.StatusInternalServerError)
			return
		}

		// Get the node associated with the request.
		n, err := getAccessNode(s, r)
		if err != nil {
			returnAPIError(s, w, err, http.StatusInternalServerError)
			return
		}

		// Decode the query string to form the byte array.
		in, err := base64.RawURLEncoding.DecodeString(r.Form.Get("data"))
		if err != nil {
			returnAPIError(s, w, err, http.StatusUnprocessableEntity)
			return
		}

		// Encrypt the byte array using the node.
		out, err := n.encrypt(in)
		if err != nil {
			returnAPIError(s, w, err, http.StatusUnprocessableEntity)
			return
		}

		// The output is a binary array.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(out)))

		// Write the encrypted byte array to the output stream.
		c, err := w.Write(out)
		if err != nil {
			returnAPIError(s, w, err, http.StatusInternalServerError)
			return
		}
		if c != len(out) {
			returnAPIError(
				s,
				w,
				fmt.Errorf("Byte count mismatch"),
				http.StatusInternalServerError)
			return
		}
	}
}
