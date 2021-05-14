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
	"net/http"
)

func HandlerShare(s *Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get the node associated with the request.
		a, err := s.store.getNode(r.Host)
		if err != nil {
			returnAPIError(s, w, err, http.StatusBadRequest)
			return
		}
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
	}
}