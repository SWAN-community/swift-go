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
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type operation struct {

	// Internal persisted state fields.
	timeStamp    time.Time // The time that the state information was created
	returnURL    string    // The URL to return to when the operation completes
	accessNode   string    // The domain name of the access node
	nodesVisited byte      // Nodes visited so far including current
	nodeCount    byte      // Number of nodes that should be visited
	pairs        []*pair   // Value pairs from the operation
	table        string    // The table to store the key value pairs in
	prevNode     string    // The domain of the previous node
	homeNode     string    // The domain of the home node
	state        []string  // Optional state information

	// The following fields are calculated for each request. Not stored.
	services    *Services     // The services used for the operation
	nextURL     *url.URL      // The next URL to navigate to
	thisNode    *node         // The node that is processing the operation
	nextNode    *node         // The next node in the operation
	prevNodePtr *node         // The pointer to the previous node in the operation
	homeNodePtr *node         // The pointer to the home node
	network     *nodes        // The nodes that form the operation network
	request     *http.Request // Http request associated with the operation
	cookiePairs []*pair       // The value pairs from cookies
	resolved    []*pair       // The resolved pairs

	HTML // Include the common HTML UI members.
}

// Regular expression to get the language string.
var languageRegex *regexp.Regexp

func init() {
	languageRegex, _ = regexp.Compile("[^;,]+")
}

func (o *operation) TimeStamp() time.Time    { return o.timeStamp }
func (o *operation) Title() string           { return o.HTML.Title }
func (o *operation) Message() string         { return o.HTML.Message }
func (o *operation) BackgroundColor() string { return o.HTML.BackgroundColor }
func (o *operation) MessageColor() string    { return o.HTML.MessageColor }
func (o *operation) ProgressColor() string   { return o.HTML.ProgressColor }
func (o *operation) ReturnURL() string       { return o.returnURL }
func (o *operation) AccessNode() string      { return o.accessNode }
func (o *operation) NextURL() *url.URL       { return o.nextURL }
func (o *operation) NodesVisited() byte      { return o.nodesVisited }
func (o *operation) NodeCount() byte         { return o.nodeCount }
func (o *operation) Debug() bool             { return o.services.config.Debug }
func (o *operation) SVGStroke() int          { return svgStroke }
func (o *operation) SVGSize() int            { return svgSize }
func (o *operation) Values() []*pair         { return o.resolved }
func (o *operation) Table() string           { return o.table }

// Results of the operation to return to the caller.
func (o *operation) Results() (string, error) {
	if o.IsTimeStampValid() == false {
		return "", fmt.Errorf("Operation timestamp invalid")
	}
	if o.accessNode == "" {
		return "", fmt.Errorf("No access node provided")
	}
	return o.getResults()
}

// Language returns the language code associated with the web page.
func (o *operation) Language() string {
	v := o.request.Header.Get("accept-language")
	if v != "" {
		return languageRegex.FindString(v)
	}
	return ""
}

// HomeNode returns the home node for the web browser. Used to ensure that the
// first and last operation occur against a consistent node for the web browser.
// TODO: What should the behavior be if the home node does not exist, what to do
// with the error from getNode.
// See https://github.com/SWAN-community/swift/issues/2
func (o *operation) HomeNode() *node {
	if o.homeNodePtr == nil {
		if o.homeNode != "" {
			o.homeNodePtr = o.services.store.getNode(o.homeNode)
		}
		if o.homeNodePtr == nil {
			o.homeNodePtr = o.network.active[0]
		}
	}
	return o.homeNodePtr
}

// PrevNode returns the previous node associated with the operatation. Used to
// update the last accessed time of a node for passive alive verification.
func (o *operation) PrevNode() *node {
	if o.prevNodePtr == nil {
		if o.prevNode != "" {
			o.prevNodePtr = o.services.store.getNode(o.prevNode)
		}
	}
	return o.prevNodePtr
}

// IsTimeStampValid true if the time is without the storage operation timeout,
// otherwise false.
func (o *operation) IsTimeStampValid() bool {
	t := o.timeStamp.Add(o.services.config.StorageOperationTimeoutDuration())
	return time.Now().UTC().Before(t)
}

// PercentageComplete the progress as a percentage of the operation.
func (o *operation) PercentageComplete() int {
	var p float64
	if o.nodeCount > 0 {
		p = (float64(o.nodesVisited) / float64(o.nodeCount)) *
			float64(100)
	}
	return int(p)
}

// SVGPath the path component of the SVG element.
func (o *operation) SVGPath() string {
	return svgPath(o.PercentageComplete())
}

func newOperation(s *Services, n *node) *operation {
	var o operation
	o.services = s
	o.timeStamp = time.Now().UTC()
	o.thisNode = n
	return &o
}

func newOperationFromByteArray(s *Services, n *node, b []byte) (*operation, error) {
	o := newOperation(s, n)
	err := o.setFromByteArray(b)
	if err != nil {
		return nil, err
	}
	return o, err
}

func newOperationFromString(s *Services, n *node, v string) (*operation, error) {
	b, err := base64.RawURLEncoding.DecodeString(v)
	if err != nil {
		return nil, err
	}
	d, err := n.decode(b)
	if err != nil {
		return nil, err
	}
	return newOperationFromByteArray(s, n, d)
}

func newOperationFromRequest(
	s *Services,
	w http.ResponseWriter,
	r *http.Request) (*operation, error) {
	var o *operation

	// Get the node associated with the request.
	t := s.store.getNode(r.Host)
	if t == nil {
		return nil, fmt.Errorf("'%s' is not a registered Swift node", r.Host)
	}

	// Get the operation data from the request using the node to decrypt.
	a := strings.Split(r.URL.Path, "/")
	if len(a) < 2 {
		return nil, fmt.Errorf(
			"Path '%s' contains insufficient segments",
			r.URL.Path)
	}
	o, err := newOperationFromString(s, t, a[len(a)-1])
	if err != nil {
		return nil, err
	}

	// Store the request incase it's needed to calculate values.
	o.request = r

	// Get the table name from the second to last segment of the URL.
	o.table, err = o.thisNode.unscramble(a[len(a)-2])
	if err != nil {
		return nil, err
	}

	// Get the network the current node is associated with.
	o.network, err = s.store.getNodes(o.thisNode.network)
	if err != nil {
		return nil, err
	}

	// Increase the number of nodes visited count.
	o.nodesVisited++

	// Get any values from the cookies and resolve any conflicts with the
	// operations values.
	o.cookiePairs = make([]*pair, 0, len(o.pairs))
	o.resolved = make([]*pair, len(o.pairs))
	for i, p := range o.pairs {

		// Default the resolved pair to the one from the operation.
		o.resolved[i] = p

		// Get the cookie if it exists for this pair.
		c, err := r.Cookie(t.scramble(p.key))
		if err == nil && c != nil {

			// Decrypt the cookie value, and if valid add it to the array of
			// cookies and resolve any conflicts with the operations pair.
			cp, err := t.getValueFromCookie(c)

			// It is possible the cookie is corrupt and therefore the value
			// should be ignored. Only log this situation in debug mode as the
			// scenario is legitimate in production.
			if s.config.Debug {
				log.Println(err)
			}

			if cp != nil {

				// Add to the array of cookie pairs.
				o.cookiePairs = append(o.cookiePairs, cp)

				// Resolve any conflict between the operation pair and the
				// cookie pair. Use this value for further storage operations.
				o.resolved[i], err = resolveConflict(p, cp)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return o, err
}

// done returns true if all the nodes needed have been visited
// The storage operation is complete id all the required nodes (nodeCount) have
// been visited OR the current node is the same as the next node and more than
// two nodes have been visited. This is for situations where the SWAN network
// contains fewer nodes than the node count for the operation and we need to
// check that cookies have been written to the browser.
func (o *operation) done() bool {
	return (o.thisNode == o.nextNode && o.nodesVisited >= 2) ||
		o.nodeCount == o.nodesVisited
}

// getCookiesValid confirms that the cookies that are present were written
// within the home node timeout and are still valid. This can be used to
// determine if the rest of the network needs to be checked for the storage
// operation. If there is no cookie AND the value is not empty then the rest of
// the network will need to be visited. If all the values are empty then cookies
// can never be valid.
func (o *operation) getCookiesValid() bool {
	e := 0
	t := time.Now().UTC()
	for _, p := range o.resolved {
		c := o.getCookie(p)
		if c != nil {
			if c.cookieWriteTime.IsZero() == false &&
				c.cookieWriteTime.Before(t) {
				t = c.cookieWriteTime
			}
		} else {
			if p.isEmpty() {
				e++
			} else {
				return false
			}
		}
	}
	d := time.Now().UTC().Sub(t)
	return d < o.services.config.HomeNodeTimeoutDuration() &&
		e < len(o.resolved)
}

// getAnyCookiesPresent returns true if any cookies are present, otherwise
// false.
func (o *operation) getAnyCookiesPresent() bool {
	return len(o.request.Cookies()) > 0
}

// setValueInCookie writes a node cookie for the pair provided.
func (o *operation) setValueInCookie(
	w http.ResponseWriter,
	r *http.Request,
	p *pair) error {
	var b bytes.Buffer
	var v []byte
	err := writeTime(&b, time.Now().UTC())
	if err != nil {
		return err
	}
	err = p.writeToBuffer(&b)
	if err != nil {
		return err
	}
	if b.Len() == 0 {
		return nil
	}
	v, err = o.thisNode.encode(b.Bytes())
	if err != nil {
		return err
	}
	s := o.services.config.Scheme == "https"
	var ss http.SameSite
	if s {
		ss = http.SameSiteNoneMode
	} else {
		ss = http.SameSiteLaxMode
	}
	cookie := http.Cookie{
		Name:     o.thisNode.scramble(p.key),
		Domain:   o.getCookieDomain(),
		Value:    base64.StdEncoding.EncodeToString(v),
		Path:     fmt.Sprintf("/%s", o.thisNode.scramble(o.table)),
		SameSite: ss,
		Secure:   s,
		HttpOnly: true,
		Expires:  p.expires}
	http.SetCookie(w, &cookie)
	return nil
}

// getCookieDomain returns the domain to be used when setting the cookie in the
// response.
func (o *operation) getCookieDomain() string {
	d := o.request.Host
	if o.thisNode.cookieDomain != "" {
		d = o.thisNode.cookieDomain
	}
	s := strings.Split(d, ":")
	return s[0]
}

// getCookie returns the cookie pair that relates to the pair provided.
func (o *operation) getCookie(p *pair) *pair {
	for _, i := range o.cookiePairs {
		if i.key == p.key {
			return i
		}
	}
	return nil
}

// resolvePairs returns an array of the pairs to use considering the values
// contained in the operation and the values from the cookies.
func (o *operation) resolvePairs() ([]*pair, error) {
	var err error

	// Create an array of pairs to store the resolved values.
	var r = make([]*pair, len(o.pairs))

	// Loop through the operation pairs resolving conflicts with the cookie
	// pairs if present.
	for i, p := range o.pairs {

		// Get the cookie pair that corresponds to this one in the operation.
		c := o.getCookie(p)
		if c != nil {

			// If there are two possible values then resolve the conflict.
			r[i], err = resolveConflict(p, c)
			if err != nil {
				return nil, err
			}
		} else {

			// These is no cookie so use the operation pair.
			r[i] = p
		}
	}
	return r, nil
}

func (o *operation) asByteArray() ([]byte, error) {
	var b bytes.Buffer
	var err error
	err = writeTime(&b, o.timeStamp)
	if err != nil {
		return nil, err
	}
	err = writeString(&b, o.returnURL)
	if err != nil {
		return nil, err
	}
	err = writeString(&b, o.accessNode)
	if err != nil {
		return nil, err
	}
	err = o.HTML.write(&b)
	if err != nil {
		return nil, err
	}
	err = writeByte(&b, o.nodesVisited)
	if err != nil {
		return nil, err
	}
	err = writeByte(&b, o.nodeCount)
	if err != nil {
		return nil, err
	}
	err = writeString(&b, o.prevNode)
	if err != nil {
		return nil, err
	}
	err = writeString(&b, o.homeNode)
	if err != nil {
		return nil, err
	}
	err = writeString(&b, strings.Join(o.state, resultSeparator))
	if err != nil {
		return nil, err
	}
	err = writeByte(&b, byte(len(o.resolved)))
	if err != nil {
		return nil, err
	}
	for _, v := range o.resolved {
		err = v.writeToBuffer(&b)
		if err != nil {
			return nil, err
		}
	}
	return b.Bytes(), nil
}

func (o *operation) setFromByteArray(d []byte) error {
	var err error
	if d == nil {
		return errors.New("Byte array empty")
	}
	b := bytes.NewBuffer(d)
	o.timeStamp, err = readTime(b)
	if err != nil {
		return err
	}
	o.returnURL, err = readString(b)
	if err != nil {
		return err
	}
	o.accessNode, err = readString(b)
	if err != nil {
		return err
	}
	err = o.HTML.set(b)
	if err != nil {
		return err
	}
	o.nodesVisited, err = readByte(b)
	if err != nil {
		return err
	}
	o.nodeCount, err = readByte(b)
	if err != nil {
		return err
	}
	o.prevNode, err = readString(b)
	if err != nil {
		return err
	}
	o.homeNode, err = readString(b)
	if err != nil {
		return err
	}
	s, err := readString(b)
	if err != nil {
		return err
	}
	o.state = strings.Split(s, resultSeparator)
	c, err := readByte(b)
	if err != nil {
		return err
	}
	for i := 0; i < int(c); i++ {
		var p pair
		err = p.setFromBuffer(b)
		if err != nil {
			return err
		}
		o.pairs = append(o.pairs, &p)
	}
	r := b.Bytes()
	if len(r) != 0 {
		err = fmt.Errorf("%d bytes remaining", len(r))
	}
	return err
}
