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

// base is a partial implementation of sws.Store for use with other more
// complex implementations, and the test methods.
type base struct {
	nodes    map[string]*node  // Map of domain names to nodes
	networks map[string]*nodes // Map of network names to nodes
	mutex    *sync.Mutex       // mutual-exclusion lock used for refresh
}

func (c *base) init(ns []*node) {
	c.nodes = make(map[string]*node)
	c.networks = make(map[string]*nodes)
	c.mutex = &sync.Mutex{}

	for _, n := range ns {
		c.nodes[n.domain] = n
		net := c.networks[n.network]
		if net == nil {
			net = &nodes{}
			net.dict = make(map[string]*node)
			c.networks[n.network] = net
		}
		net.all = append(net.all, n)
		net.dict[n.domain] = n
	}

	for _, net := range c.networks {
		net.order()
	}
}

// GetAccessNode returns an access node for the network, or null if there is no
// access node available.
func (c *base) GetAccessNode(network string) (string, error) {
	ns, err := c.getNodes(network)
	if err != nil {
		return "", err
	}
	if ns == nil {
		return "", fmt.Errorf("no access nodes for network '%s'", network)
	}
	n := ns.getRandomNode(func(n *node) bool {
		return n.role == roleAccess
	})
	if n == nil {
		return "", fmt.Errorf("no access node for network '%s'", network)
	}
	return n.domain, nil
}

// getNode takes a domain name and returns the associated node. If a node
// does not exist then nil is returned.
func (c *base) getNode(domain string) (*node, error) {
	return c.nodes[domain], nil
}

// getNodes returns all the nodes associated with a network.
func (c *base) getNodes(network string) (*nodes, error) {
	return c.networks[network], nil
}

func (c *base) getAllNodes() ([]*node, error) {
	var ns []*node

	for _, n := range c.nodes {
		ns = append(ns, n)
	}

	return ns, nil
}

// getSharingNodes returns all the nodes with the role share for all networks.
func (c *base) getSharingNodes() []*node {
	var n []*node
	for _, v := range c.nodes {
		if v.role == roleShare {
			n = append(n, v)
		}
	}
	return n
}
