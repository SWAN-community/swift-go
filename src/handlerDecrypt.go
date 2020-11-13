/* ****************************************************************************
 * Copyright 2020 51 Degrees Mobile Experts Limited
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
	"encoding/json"
	"fmt"
	"net/http"
)

// HandlerDecrypt takes a Services pointer and returns a HTTP handler used to
// decrypt the result of a storage operation provided in the raw query
// parameter to the return URL.
func HandlerDecrypt(s *Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get the node associated with the request.
		n, err := getAccessNode(s, r)
		if err != nil {
			returnAPIError(s, w, err)
			return
		}

		// Decode the query string to form the byte array.
		in, err := base64.RawURLEncoding.DecodeString(r.URL.RawQuery)
		if err != nil {
			returnAPIError(s, w, err)
			return
		}

		// Decrypt the byte array using the node.
		d, err := n.decrypt(in)
		if err != nil {
			returnAPIError(s, w, err)
			return
		}
		if d == nil {
			returnAPIError(s, w, fmt.Errorf("Could not decrypt input"))
			return
		}

		// Decode the byte array to become a results array.
		a, err := decodeResults(d)
		if err != nil {
			returnAPIError(s, w, err)
			return
		}

		// Validate that the timestamp has not expired.
		if a.isTimeStampValid() == false {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			http.Error(
				w,
				"Results expired and can no longer be decrypted",
				http.StatusNotFound)
			return
		}

		// Turn the array into a json string.
		json, err := json.Marshal(a.values)
		if err != nil {
			returnAPIError(s, w, err)
			return
		}

		// The output is a json string.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		_, err = w.Write([]byte(json))
		if err != nil {
			returnAPIError(s, w, err)
		}
	}
}
