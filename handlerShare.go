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
	"encoding/json"
	"fmt"
	"net/http"
)

// HandlerShare returns an encrypted json document which contains details for
// all known active nodes.
func HandlerShare(s *Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		// Get the node associated with the request.
		a := s.store.getNode(r.Host)
		if a == nil {
			err = fmt.Errorf("host '%s' is not a SWIFT node", r.Host)
			returnAPIError(s, w, err, http.StatusBadRequest)
			return
		}

		// If the node is not an access node then return an error.
		if a.role != roleShare {
			err = fmt.Errorf("domain '%s' is not a share node", a.domain)
			returnAPIError(s, w, err, http.StatusBadRequest)
			return
		}

		// Get all active nodes.
		ns, err := s.store.getAllActiveNodes()
		if err != nil {
			returnAPIError(s, w, err, http.StatusBadRequest)
			return
		}

		// Create JSON response.
		j, err := json.Marshal(ns)
		if err != nil {
			returnAPIError(s, w, err, http.StatusBadRequest)
			return
		}

		// Encrypt the JSON response using the nodes shared secret.
		b, err := a.encode(j)
		if err != nil {
			returnAPIError(s, w, err, http.StatusBadRequest)
			return
		}

		w.Write(b)
	}
}
