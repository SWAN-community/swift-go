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
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// share is a background service which polls known sharing nodes to fetch
// shared node data. The data is decrypted and then new Nodes are added to the
// Store.
type share struct {
	Ticker *time.Ticker
	config Configuration
	store  Store
}

// newShare creates a new instance of share
func newShare(store Store, config Configuration) *share {
	var s share
	s.config = config
	s.store = store

	s.start()

	return &s
}

// start a goroutine which fetches shared nodes in the background.
func (s *share) start() {
	go fetchSharedNodes(s)
}

// stop fetching shared nodes.
func (s *share) stop() {
	s.Ticker.Stop()
}

// cllShare makes a request to a sharing node to get shared node data and
// decrypts the resulting byte array.
func (s *share) callShare(node *Node) ([]byte, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	url := url.URL{
		Scheme: s.config.Scheme,
		Host:   n.domain,
		Path:   "/swift/api/v1/share",
	}

	req, err := http.NewRequest("GET", url.String(), nil)
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

	b, err := node.Decrypt(body)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// getNodesFromByteArray takes a byte array and tries to unmarshal it as an
// array of nodeItems, these are then converted into Nodes using the newNode
// function.
func getNodesFromByteArray(data []byte) ([]*Node, error) {
	var nodes []*Node
	var nis []nodeShareItem

	err := json.Unmarshal(data, &nis)
	if err != nil {
		return nil, err
	}

	// Convert the marshallable nodeItem array into and array of Nodes
	for _, ni := range nis {
		n, err := newNode(
			ni.Network,
			ni.Domain,
			ni.Created,
			ni.Expires,
			ni.Role,
			ni.ScrambleKey)
		if err != nil {
			return nil, err
		}
		var secrets []*secret
		for _, k := range ni.Secrets {
			s, err := newSecretFromKey(k.Key, k.Timestamp)
			if err != nil {
				log.Println(err.Error())
				continue
			}
			secrets = append(secrets, s)
		}
		if len(secrets) == 0 {
			log.Printf("shared node for %s missing secrets, skipping.../r/n", ni.Domain)
			continue
		}
		n.secrets = secrets
		nodes = append(nodes, n)
	}

	return nodes, nil
}

// fetchSharedNodes polls known sharing nodes to retrieve shared nodes.
func fetchSharedNodes(s *share) {
	d := 30 * time.Minute
	if s.config.Debug {
		d = 30 * time.Second
	}
	ticker := time.NewTicker(d)
	s.Ticker = ticker
	defer ticker.Stop()
	for _ = range ticker.C {
		nodes := s.store.getSharingNodes()
		for _, n := range nodes {
			b, err := s.callShare(n)
			if err != nil {
				log.Println(err.Error())
			}
			nodes, err := getNodesFromByteArray(b)
			if err != nil {
				log.Println(err.Error())
			}
			err = setNodes(s.store, nodes)
			if err != nil {
				log.Println(err.Error())
			}
		}
	}
}
