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
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// storageManager maintains a list of stores and a map of domains to nodes.
// Once a storageManager has been initialized, then the map of nodes is read
// only. Any new nodes that are added are only available from the storage
// manager once the storage manager has been recreated.
type storageManager struct {
	// stores is a readonly array of Stores populated when the storage manager
	// is created
	stores []Store
	// nodes is a readonly map of nodes (by domain) to the associated node
	nodes map[string]*node
	// alive is a background service which polls nodes periodically to ensure
	// that they are alive
	alive *aliveService
}

// NewStorageManager creates a new instance of storage manager and returns the
// reference. The stores provided in the sts argument are used to initialize the
// storage manager, each store is checked for nodes with the role 'roleShare'.
// If any sharing nodes are found then they are polled for any known good nodes.
// The returned nodes are added to a new Volatile read only store which is held
// in memory and then added to the list of stores. As stores are added, they are
// checked in turn for additional sharing nodes. A list of checked sharing nodes
// is maintained to prevent the same node being checked more than once.
func newStorageManager(c Configuration, sts ...Store) (*storageManager, error) {
	var sm storageManager
	sm.nodes = make(map[string]*node)
	checkedNodes := make(map[string]bool)

	for i := 0; i < len(sts); i++ {
		// check the maximum number of stores has not been reached
		if len(sts) > c.MaxStores {
			return nil, fmt.Errorf("too many stores have been configured, max is "+
				"number of stores %d", c.MaxStores)
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
				if c.Debug {
					log.Println(err.Error())
				}
			}

			nodes, err := getNodesFromByteArray(b)
			if err != nil {
				if c.Debug {
					log.Println(err.Error())
				}
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

		// add nodes in store to the map of nodes
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

// getNode gets the node associated with the domain.
func (sm *storageManager) getNode(domain string) *node { return sm.nodes[domain] }

// GetAccessNode returns an access node for the network, or null if there is no
// access node available.
func (sm *storageManager) GetAccessNode(network string) (string, error) {
	ns, err := sm.getNodes(network)
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

// getNodes returns the nodes object associated with a network.
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

// getAllActiveNodes returns all the nodes for all networks which have the alive
// flag set to true and have a start date that is before the current time.
func (sm *storageManager) getAllActiveNodes() ([]*node, error) {
	var n []*node
	for _, s := range sm.stores {
		err := s.iterateNodes(func(n *node, s interface{}) error {
			st, ok := s.(*[]*node)
			if ok &&
				n.alive &&
				n.starts.Before(time.Now().UTC()) {
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

// getAllNodes returns all the nodes from all store instances combined.
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

// setNodes adds or if supported, updates a node in the specified store.
// setNodes will also succeed if no store name is provided and only one
// writeable store exists in the storageManager.
func (sm *storageManager) setNodes(store string, ns ...*node) error {
	var stores []Store

	if len(ns) == 0 {
		return fmt.Errorf("supply some nodes to set")
	}

	for _, s := range sm.stores {
		if !s.getReadOnly() &&
			(store == "" || s.getName() == store) {
			stores = append(stores, s)
		}
	}

	if len(stores) == 0 {
		if store == "" {
			return fmt.Errorf("no writable stores found")
		} else {
			return fmt.Errorf("no writable stores by the name of '%s' found", store)
		}
	} else if len(stores) > 1 {
		var strs []string

		for _, s := range stores {
			strs = append(strs, s.getName())
		}

		return fmt.Errorf("multiple writable stores available, please select "+
			"a store from the following: '%s'", strings.Join(strs[:], ", "))
	}

	for _, n := range ns {
		err := stores[0].setNode(n)
		if err != nil {
			return err
		}
	}
	return nil
}

// addNode function for use as an argument for the store.iterateNodes function,
// adds a node to the interface which is a type of map[string]*node.
func addNode(n *node, s interface{}) error {
	st, ok := s.(map[string]*node)
	if !ok {
		return fmt.Errorf("s interface{} is not a type of 'map[string]*node'")
	}
	st[n.domain] = n
	return nil
}

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

// getSharingNodesFromStore is a helper method with iterates through all the
// nodes in a given store and returns all that have the role of 'roleShare'
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
			log.Printf("shared node for %s missing secrets, skipping..."+
				"/r/n", ni.Domain)
			continue
		}
		n.secrets = secrets
		nodes = append(nodes, n)
	}

	return nodes, nil
}
