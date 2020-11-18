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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	browserWarningParam  = "browserWarning"
	titleParam           = "title"
	messageParam         = "message"
	returnURLParam       = "returnUrl"
	progressColorParam   = "progressColor"
	backgroundColorParam = "backgroundColor"
	messageColorParam    = "messageColor"
	tableParam           = "table"
	xforwarededfor       = "X-FORWARDED-FOR"
	remoteAddr           = "remoteAddr"
	bounces              = "bounces"
	stateParam           = "state"
	accessKey            = "accessKey"
)

// HandlerCreate takes a Services pointer and returns a HTTP handler used by an
// Access Node to obtain the initial URL for a storage operation.
func HandlerCreate(s *Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check caller can access
		if s.getAccessAllowed(w, r) == false {
			returnAPIError(s, w,
				errors.New("Not authorized"),
				http.StatusUnauthorized)
			return
		}

		u, err := createURL(s, r)
		if err != nil {
			returnAPIError(s, w, err, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write([]byte(u))
	}
}

func createURL(s *Services, r *http.Request) (string, error) {

	// Get the node associated with the request.
	a, err := s.store.getNode(r.Host)
	if err != nil {
		return "", err
	}
	if a == nil {
		return "", fmt.Errorf("Host '%s' is not a Swift node", r.Host)
	}

	// If the node is not an access node then return an error.
	if a.role != roleAccess {
		return "", fmt.Errorf("Domain '%s' is not an access node", a.domain)
	}

	// Create the operation.
	o := newOperation(s, a)

	// Set the network for the operation.
	o.network, err = s.store.getNodes(a.network)
	if err != nil {
		return "", err
	}

	// Set the access node domain so that the end operation can be called
	// to decrypt the data in the return url.
	o.accessNode = a.domain

	// Add the parameters to the operation.
	err = r.ParseForm()
	if err != nil {
		return "", err
	}

	// Set the node count.
	if r.Form.Get(bounces) != "" {
		c, err := strconv.Atoi(r.Form.Get(bounces))
		if err != nil {
			return "", err
		}
		if c <= 0 {
			return "", fmt.Errorf("Bounces must be greater than 0")
		} else if c < 255 {
			o.nodeCount = byte(c)
		} else {
			return "", fmt.Errorf("Bounces '%d' must be less than 255", c)
		}
	} else {
		o.nodeCount = s.config.NodeCount
	}

	// Set the return URL that will have the encrypted data appended to it.
	ru, err := url.Parse(r.Form.Get(returnURLParam))
	if err != nil {
		return "", err
	}
	if ru.Scheme == "" {
		return "", fmt.Errorf("Missing scheme from URL '%s'", ru)
	}
	o.returnURL = ru.String()

	// Set any state information if provided.
	o.state = r.Form.Get(stateParam)

	// Set the table that will be used for the storage of the key value
	// pairs.
	o.table = r.Form.Get(tableParam)
	if o.table == "" {
		return "", fmt.Errorf("Missing table name")
	}

	// Set the browser warning probability if provided.
	b, err := strconv.ParseFloat(r.Form.Get(browserWarningParam), 32)
	if err == nil {
		// Set the browser warning probability to the value provided by the
		// the caller.
		o.browserWarning = float32(b)
	} else {
		// Something went wrong. Set to zero to ensure no warning.
		o.browserWarning = 0
	}

	// Set the user interface parameters from the optional parameters
	// provided or from the configuration if node provided and the defaults
	// should be used.
	o.HTML.Title = r.Form.Get(titleParam)
	if o.HTML.Title == "" {
		o.HTML.Title = s.config.Title
	}
	o.HTML.Message = r.Form.Get(messageParam)
	if o.HTML.Message == "" {
		o.HTML.Message = s.config.Message
	}
	o.HTML.MessageColor = r.Form.Get(messageColorParam)
	if o.HTML.MessageColor == "" {
		o.HTML.MessageColor = s.config.MessageColor
	}
	o.HTML.BackgroundColor = r.Form.Get(backgroundColorParam)
	if o.HTML.BackgroundColor == "" {
		o.HTML.BackgroundColor = s.config.BackgroundColor
	}
	o.HTML.ProgressColor = r.Form.Get(progressColorParam)
	if o.HTML.ProgressColor == "" {
		o.HTML.ProgressColor = s.config.ProgressColor
	}

	// Add the key value pairs from the form parameters.
	for k, v := range r.Form {
		if isReserved(k) == false && len(v) > 0 {
			p, err := createPair(k, v[0])
			if err != nil {
				return "", err
			}
			o.values = append(o.values, p)
		}
	}

	// For this network and request find the home node.
	xff := r.Form.Get(xforwarededfor)
	if xff == "" {
		xff = r.Header.Get("X-FORWARDED-FOR")
	}
	ra := r.Form.Get(remoteAddr)
	if ra == "" {
		ra = r.RemoteAddr
	}
	o.nextNode, err = o.network.getHomeNode(xff, ra)
	if err != nil {
		return "", err
	}

	// Store the home node for the operation in case something changes about the
	// IP address mid storage operation.
	o.homeNode = o.nextNode.domain

	// Get the next URL.
	u, err := o.getNextURL()
	if err != nil {
		return "", err
	}

	return u.String(), nil
}

func createPair(k string, v string) (*pair, error) {
	var err error
	var p pair
	var i int
	o := strings.Index(k, "<")
	n := strings.Index(k, ">")
	if o < 0 && n < 0 {
		return nil, fmt.Errorf("Key '%s' must include a '<' (oldest wins) "+
			"or '>' (newest wins) character to resolve conflicts when two "+
			"different values are encountered", k)
	}
	if o > 0 && n > 0 {
		return nil, fmt.Errorf(
			"Key '%s' must contained only one '<' or '>' character", k)
	}
	if o > 0 {
		i = o
		p.conflict = conflictOldest
	} else {
		i = n
		p.conflict = conflictNewest
	}
	p.expires, err = time.Parse("2006-01-02", k[i+1:])
	if err != nil {
		return nil, err
	}
	if p.expires.Before(time.Now().UTC()) {
		return nil, fmt.Errorf(
			"Key expiry date '%s' must be in the future", k[i+1:])
	}
	p.created = time.Now().UTC()
	p.key = k[:i]
	p.value = v
	return &p, err
}

func isReserved(s string) bool {
	return s == titleParam ||
		s == messageParam ||
		s == returnURLParam ||
		s == progressColorParam ||
		s == messageColorParam ||
		s == backgroundColorParam ||
		s == tableParam ||
		s == browserWarningParam ||
		s == xforwarededfor ||
		s == remoteAddr ||
		s == bounces ||
		s == stateParam ||
		s == accessKey
}
