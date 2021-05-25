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
	"os"
	"path"
	"sync"
	"time"
)

// Local store implementation for SWIFT - data is stored in maps in memory and
// persisted on disk in JSON files.
type Local struct {
	name        string    //
	timestamp   time.Time // The last time the maps were refreshed
	nodesFile   string    // Reference to the node table
	secretsFile string    // Reference to the table of node secrets
	common
}

// secretItem is the JSON representation of a SWIFT Secret
type secretItem struct {
	Timestamp time.Time
	Key       string
}

// NewLocalStore creates a new instance of Local and configures the paths for
// the persistent JSON files.
func NewLocalStore(name string, secretsFile string, nodesFile string) (*Local, error) {
	var l Local

	l.name = name
	l.nodesFile = nodesFile
	l.secretsFile = secretsFile

	l.mutex = &sync.Mutex{}
	err := l.refresh()
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (l *Local) getName() string {
	return l.name
}

func (l *Local) getReadOnly() bool {
	return false
}

// GetNode takes a domain name and returns the associated node. If a node
// does not exist then nil is returned.
func (l *Local) getNode(domain string) (*node, error) {
	n, err := l.common.getNode(domain)
	if err != nil {
		return nil, err
	}
	if n == nil {
		err = l.refresh()
		if err != nil {
			return nil, err
		}
		n, err = l.common.getNode(domain)
	}
	return n, err
}

// GetNodes returns all the nodes associated with a network.
func (l *Local) getNodes(network string) (*nodes, error) {
	ns, err := l.common.getNodes(network)
	if err != nil {
		return nil, err
	}
	if ns == nil {
		err = l.refresh()
		if err != nil {
			return nil, err
		}
		ns, err = l.getNodes(network)
	}
	return ns, err
}

// getAllNodes refreshes internal data and returns all nodes.
func (l *Local) getAllNodes() ([]*node, error) {
	err := l.refresh()
	if err != nil {
		return nil, err
	}
	return l.common.getAllNodes()
}

// iterateNodes calls the callback function for each node
func (l *Local) iterateNodes(
	callback func(n *node, s interface{}) error,
	s interface{}) error {
	for _, n := range l.nodes {
		err := callback(n, s)
		if err != nil {
			return err
		}
	}
	return nil
}

// SetNode inserts or updates the node.
func (l *Local) setNode(n *node) error {
	err := l.setNodeSecrets(n)
	if err != nil {
		return err
	}
	nis := make(map[string]*nodeItem)

	// Fetch all the records from the nodes file.
	data, err := ioutil.ReadFile(l.nodesFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &nis)
	if err != nil && len(data) > 0 {
		return err
	}

	nis[n.domain] = &nodeItem{
		Network:     n.network,
		Domain:      n.domain,
		Created:     n.created,
		Starts:      n.starts,
		Expires:     n.expires,
		Role:        n.role,
		ScrambleKey: n.scrambler.key,
	}

	data, err = json.MarshalIndent(&nis, "", "\t")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(l.nodesFile, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (l *Local) refresh() error {
	nets := make(map[string]*nodes)

	// Fetch the nodes and then add the secrets.
	ns, err := l.fetchNodes()
	if err != nil {
		return err
	}
	err = l.addSecrets(ns)
	if err != nil {
		return err
	}

	// Create a map of networks from the nodes found.
	for _, v := range ns {
		net := nets[v.network]
		if net == nil {
			net = &nodes{}
			net.dict = make(map[string]*node)
			nets[v.network] = net
		}
		net.all = append(net.all, v)
		net.dict[v.domain] = v
	}

	// Finally sort the nodes by hash values and whether they are active.
	for _, net := range nets {
		net.order()
	}

	// In a single atomic operation update the reference to the networks and
	// nodes.
	l.mutex.Lock()
	l.nodes = ns
	l.networks = nets
	l.mutex.Unlock()

	return nil
}

func (l *Local) addSecrets(ns map[string]*node) error {
	sc := make(map[string][]*secretItem)

	// Fetch all records from the secrets file
	data, err := readLocalStore(l.secretsFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &sc)
	if err != nil && len(data) > 0 {
		return err
	}

	// Iterate over the secrets adding them to nodes.
	for k, s := range sc {
		if err != nil {
			return err
		}
		if ns[k] != nil {
			for _, i := range s {
				s, err := newSecretFromKey(i.Key, i.Timestamp)
				if err != nil {
					return err
				}
				ns[k].addSecret(s)
			}
		}
	}

	// Sort the secrets so the most recent is at the start of the array.
	for _, n := range ns {
		n.sortSecrets()
	}

	return nil
}

func (l *Local) fetchNodes() (map[string]*node, error) {
	var err error
	ns := make(map[string]*node)
	nis := make(map[string]*nodeItem)

	// Fetch all the records from the nodes file.
	data, err := readLocalStore(l.nodesFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &nis)
	if err != nil && len(data) > 0 {
		return nil, err
	} else if len(data) == 0 {
		return ns, nil
	}

	// Iterate over the records creating nodes and adding them to the networks
	// map.
	for k, n := range nis {
		ns[k], err = newNode(
			n.Network,
			n.Domain,
			n.Created,
			n.Starts,
			n.Expires,
			n.Role,
			n.ScrambleKey)
		if err != nil {
			return nil, err
		}
	}

	return ns, err
}

func (l *Local) setNodeSecrets(n *node) error {
	sic := make(map[string][]*secretItem)

	// Fetch all records from the secrets file
	data, err := readLocalStore(l.secretsFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &sic)
	if err != nil && len(data) > 0 {
		return err
	}

	for _, i := range n.secrets {
		s := sic[n.domain]

		// check if secret exists already
		add := true
		for _, v := range s {
			if v.Key == i.key {
				add = false
				break
			}
		}

		// skip if secret already exists
		if !add {
			continue
		}

		// add new secret item to store
		sic[n.domain] = append(sic[n.domain], &secretItem{
			Timestamp: i.timeStamp,
			Key:       i.key,
		})
	}

	data, err = json.MarshalIndent(&sic, "", "\t")
	if err != nil {
		return err
	}

	err = writeLocalStore(l.secretsFile, data)
	if err != nil {
		return err
	}

	return nil
}

// readLocalStore reads the contents of a file and returns the binary data.
func readLocalStore(file string) ([]byte, error) {
	err := createLocalStore(file)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// writeLocalStore writes binary data to a file.
func writeLocalStore(file string, data []byte) error {
	err := createLocalStore(file)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(file, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

// createLocalStore creates the persistent JSON file and any parents specified
// in the path.
func createLocalStore(file string) error {
	if _, err := os.Stat(file); os.IsNotExist(err) {

		if _, err := os.Stat(path.Dir(file)); os.IsNotExist(err) {
			os.MkdirAll(path.Dir(file), 0700)
		}

		_, err = os.Create(file)
		if err != nil {
			return err
		}
	}
	return nil
}
