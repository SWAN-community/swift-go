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
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

const (
	azureTimeout = 2
)

// Azure is a implementation of sws.Store for Microsoft's Azure table storage.
type Azure struct {
	name         string
	timestamp    time.Time      // The last time the maps were refreshed
	nodesTable   *storage.Table // Reference to the node table
	secretsTable *storage.Table // Reference to the table of node secrets
	common
}

// NewAzure creates a new client for accessing table storage with the
// credentials supplied.
func NewAzure(name string, account string, accessKey string) (*Azure, error) {
	var a Azure
	a.name = name
	c, err := storage.NewBasicClient(account, accessKey)
	if err != nil {
		return nil, err
	}
	ts := c.GetTableService()
	a.mutex = &sync.Mutex{}
	a.nodesTable = ts.GetTableReference(nodesTableName)
	err = azureCreateTable(a.nodesTable)
	if err != nil {
		return nil, err
	}
	a.secretsTable = ts.GetTableReference(secretsTableName)
	err = azureCreateTable(a.secretsTable)
	if err != nil {
		return nil, err
	}
	err = a.refresh()
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (a *Azure) getName() string {
	return a.name
}

func (a *Azure) getReadOnly() bool {
	return false
}

func (a *Azure) getNode(domain string) (*node, error) {
	n, err := a.common.getNode(domain)
	if err != nil {
		return nil, err
	}
	if n == nil {
		err = a.refresh()
		if err != nil {
			return nil, err
		}
		n, err = a.common.getNode(domain)
	}
	return n, err
}

func (a *Azure) getNodes(network string) (*nodes, error) {
	ns, err := a.common.getNodes(network)
	if err != nil {
		return nil, err
	}
	if ns == nil {
		err = a.refresh()
		if err != nil {
			return nil, err
		}
		ns, err = a.common.getNodes(network)
	}
	return ns, err
}

// getAllNodes refreshes internal data and returns all nodes.
func (a *Azure) getAllNodes() ([]*node, error) {
	err := a.refresh()
	if err != nil {
		return nil, err
	}
	return a.common.getAllNodes()
}

// iterateNodes calls the callback function for each node
func (a *Azure) iterateNodes(
	callback func(n *node, s interface{}) error,
	s interface{}) error {
	for _, n := range a.nodes {
		err := callback(n, s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Azure) setNode(n *node) error {
	err := a.setNodeSecrets(n)
	if err != nil {
		return err
	}
	e := a.nodesTable.GetEntityReference(n.network, n.domain)
	e.Properties = make(map[string]interface{})
	e.Properties[expiresFieldName] = n.expires
	e.Properties[roleFieldName] = n.role
	e.Properties[scramblerKeyFieldName] = n.scrambler.key
	return e.Insert(storage.FullMetadata, nil)
}

func azureCreateTable(t *storage.Table) error {
	err := t.Create(azureTimeout, storage.FullMetadata, nil)
	if err != nil {
		switch e := err.(type) {
		case storage.AzureStorageServiceError:
			if e.Code != "TableAlreadyExists" {
				return err
			}
		default:
			return err
		}
	}
	return nil
}

func (a *Azure) refresh() error {
	nets := make(map[string]*nodes)

	// Fetch the nodes and then add the secrets.
	ns, err := a.fetchNodes()
	if err != nil {
		return err
	}
	err = a.addSecrets(ns)
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
	a.mutex.Lock()
	a.nodes = ns
	a.networks = nets
	a.mutex.Unlock()

	return nil
}

func (a *Azure) addSecrets(ns map[string]*node) error {

	// Fetch all the records from the secrets table in Azure.
	e, err := a.secretsTable.QueryEntities(
		azureTimeout,
		storage.FullMetadata,
		nil)
	if err != nil {
		return err
	}

	// Iterate over the secrets adding them to nodes.
	for _, i := range e.Entities {
		s, err := newSecretFromKey(i.RowKey, i.TimeStamp)
		if err != nil {
			return err
		}
		if ns[i.PartitionKey] != nil {
			ns[i.PartitionKey].addSecret(s)
		}
	}

	// Sort the secrets so the most recent is at the start of the array.
	for _, n := range ns {
		n.sortSecrets()
	}

	return nil
}

func (a *Azure) fetchNodes() (map[string]*node, error) {
	var err error
	ns := make(map[string]*node)

	// Fetch all the records from the nodes table in Azure.
	e, err := a.nodesTable.QueryEntities(
		azureTimeout,
		storage.FullMetadata,
		nil)
	if err != nil {
		return nil, err
	}

	// Iterate over the records creating nodes and adding them to the networks
	// map.
	for _, i := range e.Entities {
		ns[i.RowKey], err = newNode(
			i.PartitionKey,
			i.RowKey,
			i.TimeStamp,
			i.Properties[startsFieldName].(time.Time),
			i.Properties[expiresFieldName].(time.Time),
			int(i.Properties[roleFieldName].(float64)),
			i.Properties[scramblerKeyFieldName].(string))
		if err != nil {
			return nil, err
		}
	}

	return ns, err
}

func (a *Azure) setNodeSecrets(n *node) error {
	for _, s := range n.secrets {
		e := a.secretsTable.GetEntityReference(n.domain, s.key)
		e.TimeStamp = s.timeStamp
		err := e.Insert(storage.FullMetadata, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
