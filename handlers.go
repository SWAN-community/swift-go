/* ****************************************************************************
 * Copyright 2020 51 Degrees Mobile Experts Limited (51degrees.com) (51degrees.com)
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
	"io/ioutil"
	"net/http"
)

// AddHandlers to the http default mux for shared web state.
// The malformedHandler is used to tailor the response when a storage operation
// is invalid. If not provided then the default handler is used.
func AddHandlers(
	services *Services,
	malformedHandler func(w http.ResponseWriter, r *http.Request)) {
	http.HandleFunc("/swift/register", HandlerRegister(services))
	http.HandleFunc("/swift/api/v1/create", HandlerCreate(services))
	http.HandleFunc("/swift/api/v1/encrypt", HandlerEncrypt(services))
	http.HandleFunc("/swift/api/v1/decrypt", HandlerDecrypt(services))
	http.HandleFunc("/swift/api/v1/decode-as-json", HandlerDecodeAsJSON(services))
	http.HandleFunc("/", HandlerStore(services, malformedHandler))
}

func newResponseError(url string, resp *http.Response) error {
	in, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("API call '%s' returned '%d' and '%s'",
		url, resp.StatusCode, in)
}

func returnAPIError(
	s *Services,
	w http.ResponseWriter,
	err error,
	code int) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.Error(w, err.Error(), code)
	if s.config.Debug {
		println(err.Error())
	}
}

func returnServerError(s *Services, w http.ResponseWriter, err error) {
	w.Header().Set("Cache-Control", "no-cache")
	if s.config.Debug {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		http.Error(w, "", http.StatusInternalServerError)
	}
	if s.config.Debug {
		println(err.Error())
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
