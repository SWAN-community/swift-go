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

// HandlerEncrypt takes a Services pointer and returns a HTTP handler used to
// encrypt the result of a storage operation ready to be provided to the return
// URL.
func HandlerEncrypt(s *Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// No access control is needed here. All access nodes can encrypt data.
		// Access keys are needed to decrypt the data.

		err := r.ParseForm()
		if err != nil {
			common.ReturnServerError(w, err)
			return
		}

		// Get the node associated with the request.
		n, err := s.GetAccessNodeForHost(r.Host)
		if err != nil {
			common.ReturnServerError(w, err)
			return
		}

		// Decode the query string to form the byte array.
		in, err := base64.StdEncoding.DecodeString(r.Form.Get("plain"))
		if err != nil {
			common.ReturnApplicationError(w, &common.HttpError{
				Message: "bad data",
				Code:    http.StatusBadRequest})
			return
		}

		// Encrypt the byte array using the node.
		b, err := n.encode(in)
		if err != nil {
			common.ReturnApplicationError(w, &common.HttpError{
				Message: "bad data",
				Code:    http.StatusBadRequest})
			return
		}

		// The output is a binary array.
		common.SendByteArray(w, b)
	}
}
