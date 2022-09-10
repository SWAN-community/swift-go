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
	http.HandleFunc("/swift/api/v1/alive", handlerAlive(services))
	http.HandleFunc("/swift/api/v1/create", HandlerCreate(services))
	http.HandleFunc("/swift/api/v1/encrypt", HandlerEncrypt(services))
	http.HandleFunc("/swift/api/v1/decrypt", HandlerDecrypt(services))
	http.HandleFunc("/swift/api/v1/decode-as-json", HandlerDecodeAsJSON(services))
	http.HandleFunc("/swift/api/v1/share", HandlerShare(services))
	http.HandleFunc("/", HandlerStore(services, malformedHandler))

	if services.config.Debug {
		http.HandleFunc("/swift/nodes", HandlerNodes(services))
		http.HandleFunc("/swift/api/v1/nodes", HandlerNodesJSON(services))
	}
}

func newResponseError(url string, resp *http.Response) error {
	in, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("API call '%s' returned '%d' and '%s'",
		url, resp.StatusCode, in)
}
