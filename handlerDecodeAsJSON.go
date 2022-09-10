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
	"net/http"

	"github.com/SWAN-community/common-go"
)

// HandlerDecodeAsJSON returns the incoming request as JSON data. The query
// string contains the data which must be turned into a byte array, decryped and
// the resulting data turned into JSON.
func HandlerDecodeAsJSON(s *Services) http.HandlerFunc {
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
		d, err := base64.StdEncoding.DecodeString(r.Form.Get("encrypted"))
		if err != nil {
			common.ReturnApplicationError(w, &common.HttpError{
				Message: "bad data",
				Code:    http.StatusBadRequest})
			return
		}

		// Decrypt and decode the data into a Results.
		v, err := n.DecodeAsResults(d)
		if err != nil {
			common.ReturnServerError(w, err)
			return
		}

		// Validate that the timestamp has not expired.
		if v.IsTimeStampValid() == false {
			common.ReturnApplicationError(w, &common.HttpError{
				Message: "data expired and can no longer be used",
				Code:    http.StatusBadRequest})
			return
		}

		// Turn the Results into a JSON string.
		j, err := json.Marshal(v)
		if err != nil {
			common.ReturnServerError(w, err)
			return
		}

		// Send the JSON string.
		common.SendJS(w, j)
	}
}
