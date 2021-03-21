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
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

const (
	browserWarningParam        = "browserWarning"
	titleParam                 = "title"
	messageParam               = "message"
	returnURLParam             = "returnUrl"
	progressColorParam         = "progressColor"
	backgroundColorParam       = "backgroundColor"
	messageColorParam          = "messageColor"
	tableParam                 = "table"
	xforwarededfor             = "X-FORWARDED-FOR"
	remoteAddr                 = "remoteAddr"
	count                      = "bounces"
	stateParam                 = "state"
	displayUserInterfaceParam  = "displayUserInterface"
	postMessageOnCompleteParam = "postMessageOnComplete"
)

// Used to determine the storage character from the key to use for the
// operation.
var operationCharacterRegEx *regexp.Regexp

func init() {
	var err error
	operationCharacterRegEx, err = regexp.Compile("\\<|\\>|\\+")
	if err != nil {
		log.Fatal(err)
	}
}

// HandlerCreate takes a Services pointer and returns a HTTP handler used by an
// Access Node to obtain the initial URL for a storage operation.
func HandlerCreate(s *Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check caller can access and parse the form variables.
		if s.getAccessAllowed(w, r) == false {
			returnAPIError(s, w,
				errors.New("Not authorized"),
				http.StatusUnauthorized)
			return
		}

		// Create the URL from the form parameters.
		u, err := Create(s, r.Host, r.Form)
		if err != nil {
			returnAPIError(s, w, err, http.StatusBadRequest)
			return
		}

		// Return the URL.
		g := gzip.NewWriter(w)
		defer g.Close()
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, err = g.Write([]byte(u))
		if err != nil {
			returnAPIError(s, w, err, http.StatusInternalServerError)
			return
		}
	}
}

// SetHomeNodeHeaders adds the HTTP headers from the request that are relevant
// to the calculation of the home node to the values collection.
func SetHomeNodeHeaders(r *http.Request, q *url.Values) {
	if r.Header.Get("X-FORWARDED-FOR") != "" {
		q.Set("X-FORWARDED-FOR", r.Header.Get("X-FORWARDED-FOR"))
	}
	q.Set("remoteAddr", r.RemoteAddr)
}

// Create creates a storage operation URL from the parameters passed to the
// method for the node associated with the host.
// s an instance of swift.Services
// h the name of the SWIFT internet domain
// q the form paramters to be used to create the storage operation URL
func Create(s *Services, h string, q url.Values) (string, error) {

	// Get the node associated with the request.
	a, err := s.store.getNode(h)
	if err != nil {
		return "", err
	}
	if a == nil {
		return "", fmt.Errorf("Host '%s' is not a SWIFT node", h)
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

	// Set the access node for the operation.
	err = setAccessNode(s, o, &q, a)
	if err != nil {
		return "", err
	}

	// Set any state information if provided.
	o.state = q[stateParam]

	// Set the number of SWIFT nodes to use for the operation.
	err = setCount(o, &q, s)
	if err != nil {
		return "", err
	}

	// Check the flag for the posting of a message on completion rather than
	// using the return URL.
	if q.Get(postMessageOnCompleteParam) == "true" {
		o.SetPostMessageOnComplete(true)
	}

	// Set the return URL to use when posting the message or to redirect the
	// browser to with the encrypted SWAN data appended.
	ru, err := url.Parse(q.Get(returnURLParam))
	if err != nil {
		return "", err
	}
	if ru.Host == "" {
		return "", fmt.Errorf("Missing host from URL '%s'", ru)
	}
	if ru.Scheme == "" {
		return "", fmt.Errorf("Missing scheme from URL '%s'", ru)
	}
	o.returnURL = ru.String()

	// Set the table that will be used for the storage of the key value
	// pairs.
	o.table = q.Get(tableParam)
	if o.table == "" {
		return "", fmt.Errorf("Missing table name")
	}

	// Check the flag for the display of the user interface.
	o.SetDisplayUserInterface(q.Get(displayUserInterfaceParam) != "false")

	// Set the browser warning probability if provided.
	b, err := strconv.ParseFloat(q.Get(browserWarningParam), 32)
	if err == nil {
		// Set the browser warning probability to the value provided by the
		// the caller.
		o.browserWarning = float32(b)
	} else {
		// Something went wrong. Set to zero to ensure no warning.
		o.browserWarning = 0
	}

	// Set the user interface parameters from the optional parameters provided
	// or from the configuration if node provided and the defaults should be
	// used.
	o.HTML.Title = q.Get(titleParam)
	if o.HTML.Title == "" {
		o.HTML.Title = s.config.Title
	}
	o.HTML.Message = q.Get(messageParam)
	if o.HTML.Message == "" {
		o.HTML.Message = s.config.Message
	}
	o.HTML.MessageColor = q.Get(messageColorParam)
	if o.HTML.MessageColor == "" {
		o.HTML.MessageColor = s.config.MessageColor
	}
	o.HTML.BackgroundColor = q.Get(backgroundColorParam)
	if o.HTML.BackgroundColor == "" {
		o.HTML.BackgroundColor = s.config.BackgroundColor
	}
	o.HTML.ProgressColor = q.Get(progressColorParam)
	if o.HTML.ProgressColor == "" {
		o.HTML.ProgressColor = s.config.ProgressColor
	}

	// Add the key value pairs from the form parameters.
	for k, v := range q {
		if isReserved(k) == false && len(v) > 0 {
			p, err := createPair(k, v[0])
			if err != nil {
				return "", err
			}
			if p.conflict == conflictInvalid {
				return "", fmt.Errorf(
					"Pair does not contain valid conflict flag")
			}
			o.values = append(o.values, p)
		}
	}

	// For this network and request find the home node.
	o.nextNode, err = o.network.getHomeNode(
		q.Get(xforwarededfor),
		q.Get(remoteAddr))
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

	// Get the command for the storage operation.
	i := operationCharacterRegEx.FindStringIndex(k)
	if i == nil {
		return nil, fmt.Errorf("Key '%s' must include a '+' to add the value "+
			"to a list of values, or '<' (oldest wins) or '>' (newest wins) "+
			"character to determine how to resolve two values for the same "+
			"key, followed by a date in YYYY-MM-DD format to indicate when "+
			"the value expires and is automatically deleted", k)
	}
	if len(i) > 2 || i[1]-i[0] != 1 {
		return nil, fmt.Errorf(
			"Key '%s' must contained only one '+', '<' or '>' character", k)
	}

	// Set how multipe values for the same key are handled.
	switch k[i[0]] {
	case '+':
		p.conflict = conflictAdd
		break
	case '<':
		p.conflict = conflictOldest
		break
	case '>':
		p.conflict = conflictNewest
		break
	default:
		return nil, fmt.Errorf("Character '%c' invalid", k[i[0]])
	}

	// Work out the expiry time from the date that appears after the conflict
	// character.
	p.expires, err = time.Parse("2006-01-02", k[i[0]+1:])
	if err != nil {
		return nil, err
	}
	if p.expires.Before(time.Now().UTC()) {
		return nil, fmt.Errorf(
			"Key expiry date '%s' must be in the future", k[i[0]+1:])
	}

	// Complete the data for the pair.
	p.created = time.Now().UTC()
	p.key = k[:i[0]]
	p.value = v
	return &p, err
}

// Set the access node domain so that the end operation can be called to decrypt
// the data in the return url. Verify that the access node provided is a valid
// access node in the store. This prevents spoof access nodes being provided by
// bad actors attempting to gain access to the network. If no access node is
// provided then the default one will be used. The access node is not valid for
// other purposes so remove it from the parameters.
func setAccessNode(s *Services, o *operation, q *url.Values, a *Node) error {
	v := q.Get("accessNode")
	if v == "" {
		o.accessNode = a.domain
	} else {
		n, err := s.store.getNode(v)
		if err != nil {
			return err
		}
		if n == nil {
			return fmt.Errorf("'%s' is not a valid access node", v)
		}
		if a.network != n.network {
			return fmt.Errorf(
				"'%s' is node a valid access node for network '%s'",
				v,
				a.network)
		}
		o.accessNode = n.domain
	}
	q.Del("accessNode")
	return nil
}

// Set the number of SWIFT nodes that should be used for the operation.
func setCount(o *operation, q *url.Values, s *Services) error {
	if q.Get(count) != "" {
		c, err := strconv.Atoi(q.Get(count))
		if err != nil {
			return err
		}
		if c <= 0 {
			return fmt.Errorf("SWIFT node count must be greater than 0")
		} else if c < 255 {
			o.nodeCount = byte(c)
		} else {
			return fmt.Errorf(
				"SWIFT node count '%d' must be less than 255", c)
		}
	} else {
		o.nodeCount = s.config.NodeCount
	}
	return nil
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
		s == count ||
		s == stateParam ||
		s == displayUserInterfaceParam ||
		s == postMessageOnCompleteParam
}
