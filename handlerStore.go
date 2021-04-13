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
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

// HandlerStore takes a Services pointer and returns a HTTP handler used to
// respond to a storage operation. Should not be assigned to an end point as
// the table name is the first segment of the URL path, and the encrypted
// operation data the second segment. The second optional parameter is used to
// handle responses that do not contain a valid operation request due to data
// corruption.
func HandlerStore(
	s *Services,
	e func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Extract the operation parameters from the request.
		o, err := newOperationFromRequest(s, w, r)
		if err != nil {
			if e == nil {
				storeMalformed(s, w, r)
			} else {
				e(w, r)
			}
			return
		}

		// If there are still more nodes to try and the operation is not out of
		// time then select the next node.
		if o.nodesVisited < o.nodeCount && o.IsTimeStampValid() {

			// If this is the penultimate operation in the storage operation
			// then go back to the home node that will be the first one in those
			// visited to ensure it has the most current copy of the data.
			if o.nodesVisited == o.nodeCount-1 {
				o.nextNode = o.HomeNode()
			}

			// If no node is set then find a random storage node that is not the
			// home node. Try 10 times before giving up.
			if o.nextNode == nil {
				c := 10
				for o.nextNode == nil && c > 0 {
					o.nextNode = o.network.getRandomNode(func(i *Node) bool {
						return i.role == roleStorage &&
							i.domain != o.HomeNode().domain
					})
					c--
				}
			}

			// If there is still no node them use the home node.
			if o.nextNode == nil {
				o.nextNode = o.HomeNode()
			}

			// If there is still no node then generate an error.
			if o.nextNode == nil {
				returnServerError(s, w, fmt.Errorf("No next node available"))
				return
			}
		}

		if o.nextNode != nil {

			// If this is the first node, there are values in cookies for all
			// the keys of the operation, and those values have not expired
			// meaning the rest of the network does not need to be consulted to
			// complete the operation.
			if o.nodesVisited == 1 && o.getCookiesValid() {
				o.storeComplete(s, w, r)
			} else {
				o.storeContinue(s, w, r)
			}

		} else {

			// If this is the home node and the last operation of a multi node
			// operation then validate that cookies are available. If not then a
			// warning will need to be shown and the next node will be the home
			// node. Otherwise return to the returnURL.
			if o.getAllCookiesPresent() == false &&
				o.nodesVisited == o.nodeCount {
				o.storeWarning(s, w, r)
			} else {
				o.storeComplete(s, w, r)
			}
		}
	}
}

// The operation is invalid return a malformed request.
func storeMalformed(s *Services, w http.ResponseWriter, r *http.Request) {
	var o operation
	o.request = r
	o.HTML.BackgroundColor = s.config.BackgroundColor
	o.HTML.MessageColor = s.config.MessageColor
	g := gzip.NewWriter(w)
	defer g.Close()
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusBadRequest)
	err := malformedTemplate.Execute(g, &o)
	if err != nil {
		returnServerError(s, w, err)
	}
	return
}

func (o *operation) storeWarning(
	s *Services,
	w http.ResponseWriter,
	r *http.Request) {
	var err error

	// The next node after the cookies have been set is the home node. The
	// counter and the time stamp will also need to be reset to zero.
	o.nextNode = o.HomeNode()
	o.nodesVisited = 0
	o.timeStamp = time.Now().UTC()

	// Get the next URL for the node.
	o.nextURL, err = o.getNextURL()
	if err != nil {
		returnServerError(s, w, err)
		return
	}

	g := gzip.NewWriter(w)
	defer g.Close()
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	err = warningTemplate.Execute(g, o)
	if err != nil {
		returnServerError(s, w, err)
	}
}

// If the post on complete flag is set then use the JavaScript post on complete
// template. If not then use the blank template for the return.
func (o *operation) storeComplete(
	s *Services,
	w http.ResponseWriter,
	r *http.Request) {
	if o.PostMessageOnComplete() {
		o.storePostMessage(s, w, r, postMessageTemplate)
	} else {
		if o.DisplayUserInterface() {
			if o.nodesVisited <= 1 {
				o.storeReturn(s, w, r, blankTemplate)
			} else {
				o.storeReturn(s, w, r, progressTemplate)
			}
		} else {
			o.storeReturn(s, w, r, blankTemplate)
		}
	}
}

func (o *operation) storePostMessage(
	s *Services,
	w http.ResponseWriter,
	r *http.Request,
	t *template.Template) {
	g := gzip.NewWriter(w)
	defer g.Close()
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	err := t.Execute(g, o)
	if err != nil {
		returnServerError(s, w, err)
	}
}

func (o *operation) storeReturn(
	s *Services,
	w http.ResponseWriter,
	r *http.Request,
	t *template.Template) {
	var err error
	nu := o.returnURL

	// Get the results to append to the end of the return URL.
	x, err := o.Results()
	if err != nil && s.config.Debug == true {
		log.Println(err.Error())
	}
	nu += x

	// Sets cookies for any non empty resolved pairs.
	o.setCookies(s, w, r)

	// Turn the next URL string into a url.URL value.
	o.nextURL, err = url.Parse(nu)
	if err != nil {
		returnServerError(s, w, err)
		return
	}

	g := gzip.NewWriter(w)
	defer g.Close()
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	err = t.Execute(g, o)
	if err != nil {
		returnServerError(s, w, err)
	}
}

func (o *operation) storeContinue(
	s *Services,
	w http.ResponseWriter,
	r *http.Request) {
	var err error

	// Get the next URL for the node.
	o.nextURL, err = o.getNextURL()
	if err != nil {
		returnServerError(s, w, err)
		return
	}

	// Sets cookies for any non empty resolved pairs.
	o.setCookies(s, w, r)

	// Set the preload header to trigger a DNS lookup on the next domain before
	// the request to that domain occurs via the navigation change. Only do this
	// if the next node is not the home node which will have already been
	// visited.
	if o.nextNode != o.HomeNode() {
		w.Header().Set(
			"Link",
			fmt.Sprintf("<%s://%s>; rel=preconnect;",
				o.nextURL.Scheme,
				o.nextURL.Host))
	}

	g := gzip.NewWriter(w)
	defer g.Close()
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if o.DisplayUserInterface() {
		err = progressTemplate.Execute(g, o)
	} else {
		err = blankTemplate.Execute(g, o)
	}
	if err != nil {
		returnServerError(s, w, err)
	}
}

// setCookies for all the resolved pairs that are not empty.
func (o *operation) setCookies(
	s *Services,
	w http.ResponseWriter,
	r *http.Request) {
	for _, p := range o.resolved {
		if p.isEmpty() == false {
			err := o.setValueInCookie(w, r, p)
			if err != nil {
				returnServerError(s, w, err)
				return
			}
		}
	}
}

func (o *operation) getResults() (string, error) {

	// Build the results array of key value pairs.
	var r Results
	for _, p := range o.resolved {
		r.pairs = append(r.pairs, &p.Pair)
	}

	// Add the expiry time for the results.
	r.expires = time.Now().UTC().Add(
		time.Second * o.services.config.StorageOperationTimeout)

	// Add other state information from the storage operation.
	r.state = o.state

	// Add HTML user interface parameters from the storage operation.
	r.HTML = o.HTML

	// Encode them as a byte array for encryption.
	out, err := encodeResults(&r)
	if err != nil {
		return "", err
	}

	// Encrypt the result with the access node.
	var u url.URL
	u.Scheme = o.services.config.Scheme
	u.Host = o.accessNode
	u.Path = "/swift/api/v1/encrypt"
	q := url.Values{}
	q.Set("plain", base64.RawStdEncoding.EncodeToString(out))
	res, err := http.PostForm(u.String(), q)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", newResponseError(u.String(), res)
	}
	in, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(in), nil
}

func (o *operation) getNextURL() (*url.URL, error) {
	if o.nextNode == nil {
		return nil, fmt.Errorf("Next node must be set")
	}
	p, err := o.asURLParameter()
	if err != nil {
		return nil, err
	}
	var u url.URL
	u.Scheme = o.services.config.Scheme
	u.Host = o.nextNode.domain
	u.Path = o.nextNode.scramble(o.table) + "/" + p
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (o *operation) asURLParameter() (string, error) {
	b, err := o.asByteArray()
	if err != nil {
		return "", err
	}
	e, err := o.nextNode.encrypt(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(e), err
}
