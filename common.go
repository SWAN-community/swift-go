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
	"fmt"
	"sync"
)

// common is a partial implementation of sws.Store for use with other more
// complex implementations, and the test methods.
type common struct {
	nodes    map[string]*node  // Map of domain names to nodes
	networks map[string]*nodes // Map of network names to nodes
	mutex    *sync.Mutex       // mutual-exclusion lock used for refresh
}

func (c *common) init() {
	c.nodes = make(map[string]*node)
	c.networks = make(map[string]*nodes)
	c.mutex = &sync.Mutex{}
}

// GetAccessNode returns an access node for the network, or null if there is no
// access node available.
func (c *common) GetAccessNode(network string) (string, error) {
	nodes, err := c.getNodes(network)
	if err != nil {
		return "", err
	}
	if nodes == nil {
		return "", fmt.Errorf("No access nodes for network '%s'", network)
	}
	node := nodes.getRandomNode(func(n *node) bool {
		return n.role == roleAccess
	})
	if node == nil {
		return "", fmt.Errorf("No access node for network '%s'", network)
	}
	return node.domain, nil
}

// getNode takes a domain name and returns the associated node. If a node
// does not exist then nil is returned.
func (c *common) getNode(domain string) (*node, error) {
	return c.nodes[domain], nil
}

// getNodes returns all the nodes associated with a network.
func (c *common) getNodes(network string) (*nodes, error) {
	return c.networks[network], nil
}
