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
	"log"
)

// TODO make configurable
const MaxStores = 100

// storageManager... TODO
type storageManager struct {
	// stores is a readonly array of Stores populated when the storage manager
	// is created
	stores []Store
	// nodes is a readonly map of nodes (by domain) to the associated nodes
	nodes map[string]*node
	// alive is a background service which polls nodes periodically to ensure
	// that they are alive
	alive *aliveService
}

// NewStorageManager...TODO
func newStorageManager(c Configuration, sts ...Store) (*storageManager, error) {
	var sm storageManager
	sm.nodes = make(map[string]*node)
	checkedNodes := make(map[string]bool)

	for i := 0; i < len(sts); i++ {
		// check the maximum number of stores has not been reached
		if len(sts) > MaxStores {
			return nil, fmt.Errorf("too many stores have been configured, max is "+
				"number of stores %d", MaxStores)
		}

		// get the sharing nodes from this store
		ns, err := getSharingNodesFromStore(sts[i])
		if err != nil {
			log.Println(err.Error())
		}
		for _, n := range ns {
			// skip if this sharing node has been evaluated already
			if checkedNodes[n.domain] {
				continue
			} else {
				checkedNodes[n.domain] = true
			}

			// get all the nodes the shaing node knows about
			b, err := callShare(n, c.Scheme)
			if err != nil {
				log.Println(err.Error())
			}
			nodes, err := getNodesFromByteArray(b)
			if err != nil {
				log.Println(err.Error())
			}
			// check if shared nodes contain any storage nodes
			addStore := false
			for _, sn := range nodes {
				if addStore = sn.role == roleStorage; addStore {
					break
				}
			}
			// create a new readonly store
			if addStore {
				v := newVolatile(
					fmt.Sprintf("v-%d", i),
					true,
					nodes)
				sts = append(sts, v)
			}
		}

		err = sts[i].iterateNodes(addNode, sm.nodes)
		if err != nil {
			panic(err)
		}

		sm.stores = append(sm.stores, sts[i])
	}

	// create new alive service
	sm.alive = newAliveService(c, sm)

	return &sm, nil
}

func addNode(n *node, s interface{}) error {
	st, ok := s.(map[string]*node)
	if ok {
		st[n.domain] = n
		return nil
	}
	return nil
}

// getNode gets the node associated with the domain.
func (sm *storageManager) getNode(domain string) *node { return sm.nodes[domain] }

// getStores returns an array of all the stores.
func (sm *storageManager) getStores() []Store { return sm.stores }

// GetAccessNode returns an access node for the network, or null if there is no
// access node available.
func (sm *storageManager) GetAccessNode(network string) (string, error) {
	ns, err := sm.getNodes(network)
	if err != nil {
		return "", err
	}
	if ns == nil {
		return "", fmt.Errorf("No access nodes for network '%s'", network)
	}
	n := ns.getRandomNode(func(n *node) bool {
		return n.role == roleAccess
	})
	if n == nil {
		return "", fmt.Errorf("No access node for network '%s'", network)
	}
	return n.domain, nil
}

// getNodes returns the nodes object
func (sm *storageManager) getNodes(network string) (*nodes, error) {
	for _, s := range sm.stores {
		nets, err := s.getNodes(network)
		if err != nil {
			return nil, err
		}
		if nets != nil {
			return nets, nil
		}
	}
	return nil, nil
}

// getAllNodes returns all the nodes for all networks.
func (sm *storageManager) getAllActiveNodes() ([]*node, error) {
	var n []*node
	for _, s := range sm.stores {
		err := s.iterateNodes(func(n *node, s interface{}) error {
			st, ok := s.(*[]*node)
			if ok && n.alive {
				*st = append(*st, n)
				return nil
			}
			return fmt.Errorf("%v not a []*node", s)
		}, &n)
		if err != nil {
			return nil, err
		}

	}
	return n, nil
}

// getAllNodes returns all the nodes for all networks.
func (sm *storageManager) getAllNodes() ([]*node, error) {
	var n []*node
	for _, s := range sm.stores {
		err := s.iterateNodes(func(n *node, s interface{}) error {
			st, ok := s.(*[]*node)
			if ok {
				*st = append(*st, n)
				return nil
			}
			return fmt.Errorf("%v not a []*node", s)
		}, &n)
		if err != nil {
			return nil, err
		}

	}
	return n, nil
}

// getSharingNodes returns all the nodes with the role share for all networks.
func (sm *storageManager) getSharingNodes() ([]*node, error) {
	var ns []*node

	for _, s := range sm.stores {
		ns1, err := getSharingNodesFromStore(s)
		ns = append(ns, ns1...)
		if err != nil {
			return nil, err
		}
	}

	return ns, nil
}

func getSharingNodesFromStore(s Store) ([]*node, error) {
	var ns []*node

	err := s.iterateNodes(func(n *node, sta interface{}) error {
		st, ok := sta.(*[]*node)
		if ok && n.role == roleShare {
			*st = append(*st, n)
			return nil
		}
		return nil
	}, &ns)

	if err != nil {
		return nil, err
	}

	return ns, nil
}

func (sm *storageManager) setNodes(ns ...*node) error {
	var stores []Store

	if len(ns) == 0 {
		return fmt.Errorf("supply some nodes to set")
	}

	for _, s := range sm.stores {
		if !s.getReadOnly() {
			stores = append(stores, s)
		}
	}

	for _, n := range ns {
		for _, s := range stores {
			if s == nil {
				return fmt.Errorf("store for node '%s' does not exist", n.domain)
			}
			err := s.setNode(n)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
