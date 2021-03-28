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
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// HandlerDecodeAsJSON returns the incoming request as JSON data. The query
// string contains the data which must be turned into a byte array, decryped and
// the resulting data turned into JSON.
func HandlerDecodeAsJSON(s *Services) http.HandlerFunc {
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
		d, err := base64.RawStdEncoding.DecodeString(r.Form.Get("encrypted"))
		if err != nil {
			returnAPIError(s, w, err, http.StatusBadRequest)
			return
		}

		// Decrypt and decode the data into a Results.
		v, err := n.DecryptAndDecode(d)
		if err != nil {
			returnAPIError(s, w, err, http.StatusInternalServerError)
			return
		}

		// Validate that the timestamp has not expired.
		if v.IsTimeStampValid() == false {
			returnAPIError(
				s,
				w,
				fmt.Errorf("data expired and can no longer be used"),
				http.StatusBadRequest)
			return
		}

		// Turn the Results into a JSON string.
		j, err := json.Marshal(v)
		if err != nil {
			returnAPIError(s, w, err, http.StatusInternalServerError)
			return
		}

		// Send the JSON string.
		g := gzip.NewWriter(w)
		defer g.Close()
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")
		_, err = g.Write(j)
		if err != nil {
			returnAPIError(s, w, err, http.StatusInternalServerError)
		}
	}
}
