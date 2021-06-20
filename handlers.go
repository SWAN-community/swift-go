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
	"compress/gzip"
	"fmt"
	"html/template"
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

func returnAPIError(
	s *Services,
	w http.ResponseWriter,
	err error,
	code int) {
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

// getWriter creates a new compressed writer for the content type provided.
func getWriter(w http.ResponseWriter, c string) *gzip.Writer {
	g := gzip.NewWriter(w)
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", c)
	w.Header().Set("Cache-Control", "no-cache")
	return g
}

func sendTemplate(s *Services,
	w http.ResponseWriter,
	t *template.Template,
	c string,
	m interface{}) {
	g := getWriter(w, c)
	defer g.Close()
	err := t.Execute(g, m)
	if err != nil {
		returnServerError(s, w, err)
	}
}

func sendHTMLTemplate(s *Services,
	w http.ResponseWriter,
	t *template.Template,
	m interface{}) {
	sendTemplate(s, w, t, "text/html; charset=utf-8", m)
}

func sendJSTemplate(s *Services,
	w http.ResponseWriter,
	t *template.Template,
	m interface{}) {
	sendTemplate(s, w, t, "application/javascript; charset=utf-8", m)
}

func sendResponse(
	s *Services,
	w http.ResponseWriter,
	c string,
	b []byte) {
	g := getWriter(w, c)
	defer g.Close()
	l, err := g.Write(b)
	if err != nil {
		returnAPIError(s, w, err, http.StatusInternalServerError)
		return
	}
	if l != len(b) {
		returnAPIError(
			s,
			w,
			fmt.Errorf("Byte count mismatch"),
			http.StatusInternalServerError)
		return
	}
}
