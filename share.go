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
	"net/http"
	"time"
)

type share struct {
	Ticker *time.Ticker
	scheme string
	store  Store
}

func newShare(store Store, config Configuration) *share {
	var s share
	s.scheme = config.Scheme
	s.store = store

	ticker := time.NewTicker(10 * time.Second)
	s.Ticker = ticker
	defer ticker.Stop()

	go func() {
		for _ = range ticker.C {
			nodes := s.store.getSharingNodes()
			for _, n := range nodes {
				b, err := s.callShare(n.domain)
				if err != nil {
					// TODO: do something
				}
				nodes, err := getNodesFromByteArray(b)
				if err != nil {
					// TODO: do something
				}
				err = setNodes(s.store, nodes)
				if err != nil {
					// TODO: do something
				}
			}
		}
	}()

	return &s
}

func (s *share) callShare(host string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", s.scheme+"://"+host+"/swift/api/v1/share", nil)
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

// addSharedNodes
func getNodesFromByteArray(data []byte) ([]*Node, error) {
	var nodes []*Node
	var nis []nodeItem

	err := json.Unmarshal(data, &nis)
	if err != nil {
		return nil, err
	}

	// Convert the marshallable nodeItem array into and array of Nodes
	for _, n := range nis {
		n, err := newNode(
			n.Network,
			n.Domain,
			n.Created,
			n.Expires,
			n.Role,
			n.ScrambleKey)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}

	return nodes, nil
}
