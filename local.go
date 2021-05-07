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

type Local struct {
	timestamp   time.Time // The last time the maps were refreshed
	nodesFile   string    // Reference to the node table
	secretsFile string    // Reference to the table of node secrets
	common
}

type nodeItem struct {
	Network     string
	Domain      string
	Created     time.Time
	Expires     time.Time
	Role        int
	ScrambleKey string
}

type secretItem struct {
	Timestamp time.Time
	Key       string
}

func NewLocalStore(secretsFile string, nodesFile string) (*Local, error) {
	var l Local

	l.nodesFile = nodesFile
	l.secretsFile = secretsFile

	l.mutex = &sync.Mutex{}
	err := l.refresh()
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// GetNode takes a domain name and returns the associated node. If a node
// does not exist then nil is returned.
func (l *Local) getNode(domain string) (*Node, error) {
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

// SetNode inserts or updates the node.
func (l *Local) setNode(node *Node) error {
	err := l.setNodeSecrets(node)
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

	nis[node.network] = &nodeItem{
		Network:     node.network,
		Domain:      node.domain,
		Created:     node.created,
		Expires:     node.expires,
		Role:        node.role,
		ScrambleKey: node.scrambler.key,
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
			net.dict = make(map[string]*Node)
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

func (l *Local) addSecrets(ns map[string]*Node) error {
	sc := make(map[string][]*secret)

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
				ns[k].addSecret(i)
			}
		}
	}

	// Sort the secrets so the most recent is at the start of the array.
	for _, n := range ns {
		n.sortSecrets()
	}

	return nil
}

func (l *Local) fetchNodes() (map[string]*Node, error) {
	var err error
	ns := make(map[string]*Node)
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
			n.Expires,
			n.Role,
			n.ScrambleKey)
		if err != nil {
			return nil, err
		}
	}

	return ns, err
}

func (l *Local) setNodeSecrets(node *Node) error {
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

	for _, i := range node.secrets {
		sic[node.domain] = append(sic[node.domain], &secretItem{
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