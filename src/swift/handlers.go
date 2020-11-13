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
	"fmt"
	"net/http"
)

// AddHandlers to the http default mux for shared web state.
func AddHandlers(s *Services) {
	http.HandleFunc("/register", handlerRegister(s))
	http.HandleFunc("/api/v1/create", handlerCreate(s))
	http.HandleFunc("/api/v1/encrypt", handlerEncrypt(s))
	http.HandleFunc("/api/v1/decrypt", handlerDecrypt(s))
	http.HandleFunc("/", handlerStore(s))
}

func returnAPIError(s *Services, w http.ResponseWriter, err error) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func returnError(s *Services, w http.ResponseWriter, err error) {
	w.Header().Set("Cache-Control", "no-cache")
	if s.config.Debug {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		http.Error(w, "", http.StatusInternalServerError)
	}
}

func getStorageNode(s *Services, r *http.Request) (*node, error) {
	return getNodeFromRequest(s, r, roleStorage)
}

func getAccessNode(s *Services, r *http.Request) (*node, error) {
	return getNodeFromRequest(s, r, roleAccess)
}

func getNodeFromRequest(s *Services, r *http.Request, q int) (*node, error) {

	// Get the node associated with the request.
	n, err := s.store.getNode(r.Host)
	if err != nil {
		return nil, err
	}

	// Verify that this node is the right type.
	if n.role != q {
		return nil, fmt.Errorf("Node '%s' incorrect type", n.domain)
	}

	return n, nil
}
