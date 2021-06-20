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
	"crypto/rand"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"
)

// The size of the nounce used for the keep alive service.
const nounceSize = 32

// aliveService type is service which polls known nodes to determine if they are
// 'alive' and responding to requests. Only nodes that have not been accessed
// for a period of time greater than the polling interval will be polled. On a
// successful poll, the node's accessed time is updated and the 'alive' value is
// set to true. Otherwise, the node's 'alive' value is set to false.
type aliveService struct {
	ticker          *time.Ticker
	config          Configuration  // swift config
	store           storageManager // swift storage manager
	pollingInterval time.Duration
}

// newAliveService creates a new instance of type alive and starts the
// background polling service.
func newAliveService(c Configuration, s storageManager) *aliveService {
	var a aliveService

	a.config = c
	a.store = s

	if a.config.AlivePollingSeconds == 0 {
		panic("configured for 'alivePollingSeconds' is not valid, please set " +
			"to a positive integer")
	}
	a.pollingInterval = time.Duration(time.Duration(
		a.config.AlivePollingSeconds) * time.Second)

	// start the polling loop
	go a.aliveLoop()

	return &a
}

// checkAlive starts a new ticker and stores a reference to it in the
// aliveService. For each tick, all nodes known by the storageService are
// polled.
// The transport is configured to disable keep-alive to avoid exhausting the
// number of open connections in the environment. Compression is not used
// because the payload is only 32 bytes. There is no benefit from HTTP 2 so this
// is not required. There is a short timeout as an alive node will respond
// quickly.
func (a *aliveService) aliveLoop() {
	t := &http.Transport{
		DisableKeepAlives:     true,
		DisableCompression:    true,
		ForceAttemptHTTP2:     false,
		MaxConnsPerHost:       1,
		MaxIdleConnsPerHost:   1,
		MaxIdleConns:          len(a.store.nodes),
		IdleConnTimeout:       time.Second,
		ResponseHeaderTimeout: time.Second,
		ExpectContinueTimeout: time.Second}
	c := &http.Client{
		Timeout:   a.pollingInterval,
		Transport: t}

	a.ticker = time.NewTicker(a.pollingInterval)
	for _ = range a.ticker.C {
		a.ticker.Stop()
		for _, n := range a.store.nodes {
			a.pollNode(n, c)
		}
		c.CloseIdleConnections()
		a.ticker.Reset(a.pollingInterval)
	}
}

// pollNode polls the given node to determine if it is alive and responding to
// requests. If the node has not been accessed for longer than the polling
// interval then the node is polled with a nonce value that has been encrypted
// with the polled node's shared secret. If there is a response back from the
// polled node and the response value is the same as the original nonce value
// then the node's 'alive' value is set to true.
//
// n is the node to be polled
//
// c is the http.Client to use for the request
func (a *aliveService) pollNode(n *node, c *http.Client) {
	if time.Now().UTC().Sub(n.accessed) >= a.pollingInterval {

		// create a new nonce value
		nonce, err := nonce()
		if err != nil {
			if a.config.Debug {
				log.Printf("SWIFT: could not generate nonce, "+
					"aliveService failed to check node '%s'\r\n", n.domain)
				log.Println(err.Error())
			}
			n.alive = false
			return
		}

		// encrypt the nonce using the target node's shared secret
		b1, err := n.encrypt(nonce)
		if err != nil {
			if a.config.Debug {
				log.Printf("SWIFT: could not encrypt nonce using node's "+
					"shared secret, aliveService failed to check node "+
					"'%s'\r\n", n.domain)
				log.Println(err.Error())
			}
		}

		// call the node's 'alive' endpoint with the encrypted nonce value
		// and get the response.
		b2, err := a.callAlive(n, c, b1)
		if err != nil {
			if a.config.Debug {
				log.Printf("SWIFT: alive check failed for node "+
					"'%s'\r\n", n.domain)
				log.Println(err.Error())
			}
			n.alive = false
			return
		}

		// check that the response is equal to the original nonce value. This
		// confirms that the node is responding and that the known shared
		// secret is valid.
		if bytes.Equal(nonce, b2) {
			n.alive = true
			n.accessed = time.Now().UTC()
			return
		}
		n.alive = false
	}
}

// callAlive sends a POST request to a given nodes alive endpoint, the request
// contains the the given data. On a successful request, the response body is
// then returned.
func (a *aliveService) callAlive(
	n *node,
	c *http.Client,
	d []byte) ([]byte, error) {

	url := url.URL{
		Scheme: a.config.Scheme,
		Host:   n.domain,
		Path:   "/swift/api/v1/alive",
	}

	req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(d))
	if err != nil {
		return nil, err
	}

	r, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// nonce returns a new nonce generated using crpyto/rand
func nonce() ([]byte, error) {
	b := make([]byte, nounceSize)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
