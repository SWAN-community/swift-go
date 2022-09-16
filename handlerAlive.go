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
	"fmt"
	"io"
	"net/http"

	"github.com/SWAN-community/common-go"
)

// handlerAlive is a handler which take the value from the request body and
// tries to decrypt it using the shared secret of the node associated with the
// request. If successful then the decrypted value is returned in the response.
// The caller will then know that the shared secret they have is still valid.
func handlerAlive(s *Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get the body bytes from the request.
		b, err := io.ReadAll(r.Body)
		if err != nil {
			common.ReturnServerError(w, err)
			return
		}
		r.Body.Close()

		// Get the node associated with the request.
		n := s.store.getNode(r.Host)
		if n == nil {
			common.ReturnApplicationError(w, &common.HttpError{
				Message: fmt.Sprintf("no node for '%s'", r.Host),
				Code:    http.StatusBadRequest})
			return
		}

		// Decode the body to form the decrypted byte array.
		decrypted, err := n.decode(b)
		if err != nil {
			common.ReturnApplicationError(w, &common.HttpError{
				Message: "bad data",
				Code:    http.StatusBadRequest})
			return
		}

		// Return the decrypted information uncompressed.
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Cache-Length", fmt.Sprintf("%d", len(decrypted)))
		l, err := w.Write(decrypted)
		if err != nil {
			common.ReturnServerError(w, err)
			return
		}
		if l != len(decrypted) {
			common.ReturnServerError(w, err)
			return
		}
	}
}
