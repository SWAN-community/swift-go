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
	"net/url"
	"time"
)

// callShare makes a request to a sharing node to get shared node data and
// decrypts the resulting byte array.
func callShare(n *node, scheme string) ([]byte, error) {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	url := url.URL{
		Scheme: scheme,
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

	b, err := n.Decrypt(body)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// getNodesFromByteArray takes a byte array and tries to unmarshal it as an
// array of nodeItems, these are then converted into Nodes using the newNode
// function.
func getNodesFromByteArray(data []byte) ([]*node, error) {
	var nodes []*node
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
			ni.Starts,
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
