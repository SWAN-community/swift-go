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
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type operation struct {

	// Internal persisted state fields.
	timeStamp      time.Time // The time that the state information was created
	returnURL      string    // The URL to return to when the operation completes
	browserWarning float32   // Probability of browser warning display
	accessNode     string    // The domain name of the access node
	nodesVisited   byte      // Nodes visited so far including current
	nodeCount      byte      // Number of nodes that should be visited
	values         []*pair   // Values of the data being stored
	table          string    // The table to store the key value pairs in
	homeNode       string    // The domain of the home node
	state          []string  // Optional state information

	// The following fields are calculated for each request. Not stored.
	services    *Services     // The services used for the operation
	nextURL     *url.URL      // The next URL to navigate to
	thisNode    *Node         // The node that is processing the operation
	nextNode    *Node         // The next node in the operation
	homeNodePtr *Node         // The pointer to the home node
	network     *nodes        // The nodes that form the operation network
	request     *http.Request // Http request associated with the operation

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
func (o *operation) Values() []*pair         { return o.values }

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
func (o *operation) HomeNode() *Node {
	if o.homeNodePtr == nil {
		if o.homeNode != "" {
			o.homeNodePtr, _ = o.services.store.getNode(o.homeNode)
		}
		if o.homeNodePtr == nil {
			o.homeNodePtr = o.network.active[0]
		}
	}
	return o.homeNodePtr
}

func (o *operation) IsTimeStampValid() bool {
	t := o.timeStamp.Add(time.Second * o.services.config.BundleTimeout)
	return time.Now().UTC().Before(t)
}

func (o *operation) PercentageComplete() int {
	var p float64
	if o.nodeCount > 0 {
		p = (float64(o.nodesVisited) / float64(o.nodeCount)) *
			float64(100)
	}
	return int(p)
}

func (o *operation) SVGPath() string {
	return svgPath(o.PercentageComplete())
}

func newOperation(s *Services, n *Node) *operation {
	var o operation
	o.services = s
	o.timeStamp = time.Now().UTC()
	o.thisNode = n
	return &o
}

func newOperationFromByteArray(s *Services, n *Node, b []byte) (*operation, error) {
	o := newOperation(s, n)
	err := o.setFromByteArray(b)
	if err != nil {
		return nil, err
	}
	return o, err
}

func newOperationFromString(s *Services, n *Node, v string) (*operation, error) {
	b, err := base64.RawURLEncoding.DecodeString(v)
	if err != nil {
		return nil, err
	}
	d, err := n.Decrypt(b)
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
	t, err := s.store.getNode(r.Host)
	if err != nil {
		return nil, err
	}
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
	o, err = newOperationFromString(s, t, a[len(a)-1])
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

	return o, err
}

// If cookies were used to create the values in the bundle and they're all set
// within a time frame that means they do not need to be sent to other nodes
// in the network then return true.
func (o *operation) getCookiesValid() bool {
	t := time.Now().UTC()
	for _, p := range o.values {
		if p.cookieWriteTime.Before(t) {
			t = p.cookieWriteTime
		}
	}
	d := time.Now().UTC().Sub(t) / time.Second
	return d < o.services.config.HomeNodeTimeout
}

// Returns true if cookies exist for all the values in the bundle, otherwise
// false.
func (o *operation) getCookiesPresent() bool {
	z := true
	for _, p := range o.values {
		z = z && (p.cookieWriteTime.IsZero() == false)
	}
	return z
}

func (o *operation) setValueInCookie(
	w http.ResponseWriter,
	r *http.Request,
	p *pair) error {
	var b bytes.Buffer
	err := writeTime(&b, time.Now().UTC())
	if err != nil {
		return err
	}
	err = p.writeToBuffer(&b)
	if err != nil {
		return err
	}
	v, err := o.thisNode.encrypt(b.Bytes())
	if err != nil {
		return err
	}
	cookie := http.Cookie{
		Name:     o.thisNode.scramble(p.key),
		Domain:   getDomain(r.Host),
		Value:    base64.RawURLEncoding.EncodeToString(v),
		Path:     fmt.Sprintf("/%s", o.thisNode.scramble(o.table)),
		SameSite: http.SameSiteLaxMode,
		Secure:   o.services.config.Scheme == "https",
		HttpOnly: true,
		Expires:  p.expires}
	http.SetCookie(w, &cookie)
	return nil
}

func getDomain(h string) string {
	s := strings.Split(h, ":")
	return s[0]
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
	err = writeString(&b, o.homeNode)
	if err != nil {
		return nil, err
	}
	err = writeString(&b, strings.Join(o.state, resultSeparator))
	if err != nil {
		return nil, err
	}
	err = writeByte(&b, byte(len(o.values)))
	if err != nil {
		return nil, err
	}
	for _, v := range o.values {
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
		o.values = append(o.values, &p)
	}
	r := b.Bytes()
	if len(r) != 0 {
		err = fmt.Errorf("%d bytes remaining", len(r))
	}
	return err
}
