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
	"encoding/json"
	"fmt"
	"net/http"
)

// HandlerDecodeAsJSON returns the incoming request as JSON data. The query
// string contains the data which must be turned into a byte array, decryped and
// the resulting data turned into JSON.
func HandlerDecodeAsJSON(s *Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get the node associated with the request.
		n, err := getAccessNode(s, r)
		if err != nil {
			returnAPIError(s, w, err, http.StatusInternalServerError)
			return
		}

		// Decode the query string to form the byte array.
		in, err := base64.RawURLEncoding.DecodeString(r.URL.RawQuery)
		if err != nil {
			returnAPIError(s, w, err, http.StatusUnprocessableEntity)
			return
		}

		// Decrypt the byte array using the node.
		d, err := n.decrypt(in)
		if err != nil {
			returnAPIError(s, w, err, http.StatusUnprocessableEntity)
			return
		}
		if d == nil {
			returnAPIError(
				s,
				w,
				fmt.Errorf("Could not decrypt input"),
				http.StatusUnprocessableEntity)
			return
		}

		// Decode the byte array to become a results array.
		a, err := DecodeResults(d)
		if err != nil {
			returnAPIError(s, w, err, http.StatusUnprocessableEntity)
			return
		}

		// Validate that the timestamp has not expired.
		if a.IsTimeStampValid() == false {
			returnAPIError(
				s,
				w,
				fmt.Errorf("Results expired and can no longer be decrypted"),
				http.StatusUnprocessableEntity)
			return
		}

		// Turn the array into a JSON string.
		json, err := json.Marshal(a.Values)
		if err != nil {
			returnAPIError(s, w, err, http.StatusInternalServerError)
			return
		}

		// The output is a json string.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		_, err = w.Write([]byte(json))
		if err != nil {
			returnAPIError(s, w, err, http.StatusInternalServerError)
		}
	}
}
