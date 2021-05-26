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

// aliveService is a
type aliveService struct {
	ticker          *time.Ticker
	config          Configuration  // swift config
	store           storageManager // swift storage manager
	pollingInterval time.Duration
}

// newAliveService creates a new instance of type alive and starts the background
// polling service,
func newAliveService(c Configuration, s storageManager) *aliveService {
	var a aliveService

	a.config = c
	a.store = s

	if a.config.AlivePollingSeconds == 0 {
		panic("configured for 'alivePollingSeconds' is not valid, please set to " +
			"a positive integer")
	}
	a.pollingInterval = time.Duration(time.Duration(a.config.AlivePollingSeconds) * time.Second)

	// start the polling service
	a.start()

	return &a
}

// start a goroutine which checks nodes are alive in the background.
func (a *aliveService) start() {
	go a.checkAlive()
}

// stop checking if nodes are alive.
func (a *aliveService) stop() {
	a.ticker.Stop()
}

func (a *aliveService) checkAlive() {
	ticker := time.NewTicker(a.pollingInterval)
	a.ticker = ticker
	defer ticker.Stop()
	for _ = range ticker.C {
		for _, n := range a.store.nodes {
			if time.Now().UTC().Sub(n.accessed) >= a.pollingInterval {
				nonce, err := nonce()
				if err != nil {
					if a.config.Debug {
						log.Printf("SWIFT: could not generate nonce, alive failed "+
							"to check node '%s'\r\n", n.domain)
						log.Println(err.Error())
					}
					n.alive = false
					continue
				}
				b, err := a.callAlive(n, nonce)
				if err != nil {
					if a.config.Debug {
						log.Printf("SWIFT: alive check failed for node "+
							"'%s'\r\n", n.domain)
						log.Println(err.Error())
					}
					n.alive = false
					continue
				}
				if bytes.Equal(nonce, b) {
					n.alive = true
					n.accessed = time.Now().UTC()
					continue
				}
				n.alive = false
			}
		}
	}
}

func (a *aliveService) callAlive(n *node, data []byte) ([]byte, error) {
	client := &http.Client{
		Timeout: a.pollingInterval,
	}
	url := url.URL{
		Scheme: a.config.Scheme,
		Host:   n.domain,
		Path:   "/swift/api/v1/alive",
	}

	b1, err := n.encrypt(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(b1))
	if err != nil {
		return nil, err
	}

	r, err := client.Do(req)
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

func nonce() ([]byte, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
