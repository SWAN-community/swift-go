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
	"fmt"
	"html/template"
	"io/ioutil"
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
			// home node.
			if o.nextNode == nil {
				o.nextNode = o.network.getRandomNode(func(i *node) bool {
					return i != o.HomeNode() && i.role == roleStorage
				})
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

			// Process any cookies to make sure this node stores the current version
			// of the data.
			o.processCookies(w, r)

			// If this is the first node and all the cookies are valid then
			// there is no need to continue bouncing.
			if o.nodesVisited <= 1 && o.getCookiesValid() {
				o.storeReturn(s, w, r, blankTemplate)
			} else {
				o.storeContinue(s, w, r)
			}

		} else {

			// Process any cookies to make sure this node stores the current
			// version of the data.
			o.processCookies(w, r)

			// If this is the home node and the last operation then validate
			// that cookies are available. If not then a warning will need to be
			// shown and the next node will be the home node.
			// Otherwise return to the returnURL.
			if o.getCookiesPresent() == false {
				o.storeWarning(s, w, r)
			} else {
				o.storeReturn(s, w, r, progressTemplate)
			}
		}

	}
}

// The operation is invalid return a malformed request.
func storeMalformed(s *Services, w http.ResponseWriter, r *http.Request) {
	var o operation
	o.HTML.BackgroundColor = s.config.BackgroundColor
	o.HTML.MessageColor = s.config.MessageColor
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusBadRequest)
	err := malformedTemplate.Execute(w, &o)
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
	// counter will also need to be reset to zero.
	o.nextNode = o.HomeNode()
	o.nodesVisited = 0

	// Get the next URL for the node.
	o.nextURL, err = o.getNextURL()
	if err != nil {
		returnServerError(s, w, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	err = warningTemplate.Execute(w, o)
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
	if o.IsTimeStampValid() {
		// The time stamp is valid so add the data to the end of the
		// url.
		x, err := o.getResults()
		if err != nil {
			returnServerError(s, w, err)
			return
		}
		nu += x
	}

	// Turn the next URL string into a url.URL value.
	o.nextURL, err = url.Parse(nu)
	if err != nil {
		returnServerError(s, w, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	err = t.Execute(w, o)
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	err = progressTemplate.Execute(w, o)
	if err != nil {
		returnServerError(s, w, err)
	}
}

func (o *operation) getResults() (string, error) {

	// Build the results array of key value pairs.
	var r Results
	for _, p := range o.values {
		r.Values = append(
			r.Values,
			&Result{p.key, p.created, p.expires, p.value})
	}

	// Add the expiry time for the results.
	r.Expires = time.Now().UTC().Add(
		time.Second * o.services.config.BundleTimeout)

	// Add other state information from the storage operation.
	r.State = o.state

	// Add HTML user interface parameters from the storage operation.
	r.HTML = o.HTML

	// Encode them as a byte array for encryption.
	out, err := encodeResults(&r)
	if err != nil {
		return "", err
	}

	// Encrypt the result with the access node.
	u, err := url.Parse(
		o.services.config.Scheme + "://" + o.accessNode + "/swift/api/v1/encrypt")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("data", base64.RawURLEncoding.EncodeToString(out))
	u.RawQuery = q.Encode()

	res, err := http.Get(u.String())
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
	u, err := url.Parse(fmt.Sprintf("%s://%s/%s/%s",
		o.services.config.Scheme,
		o.nextNode.domain,
		o.nextNode.scramble(o.table),
		p))
	if err != nil {
		return nil, err
	}
	return u, nil
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

// For each of the keys that this operation is concerned with find the value
// from the cookies if there is a cookie that shares the key. If there is no
// cookie already for the key then set one in the response.
func (o *operation) processCookies(w http.ResponseWriter, r *http.Request) {
	for _, p := range o.values {
		c, err := r.Cookie(o.thisNode.scramble(p.key))
		if err == nil {
			processCookie(w, r, o, p, c)
		} else {
			o.setValueInCookie(w, r, p)
		}
	}
}

func processCookie(
	w http.ResponseWriter,
	r *http.Request,
	o *operation,
	p *pair,
	c *http.Cookie) error {

	// Decrypt the cookie value, and continue if valid.
	v, err := o.thisNode.getValueFromCookie(c)
	if err != nil {

		// The current cookie is invalid and can't be used. Set the cookie to
		// the operations value.
		o.setValueInCookie(w, r, p)

	} else {

		// Resolve the conflict between the operation's value and the one found
		// in the cookie.
		res := resolveConflict(p, v)

		// Update the cookie for this node with the resolved value. Even if the
		// value is not changing the pair.cookieWriteTime field needs to be
		// updated to indicate to subsequent operations when the data was last
		// current.
		o.setValueInCookie(w, r, res)

		// If the cookie pair was chosen then update the operation data.
		if res != p {
			p.conflict = v.conflict
			p.created = v.created
			p.expires = v.expires
			p.key = v.key
			p.value = v.value
			p.cookieWriteTime = res.cookieWriteTime
		}
	}

	return nil
}
