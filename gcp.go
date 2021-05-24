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
	"context"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/iterator"
)

// Firebase is a implementation of owid.Store for GCP's Firebase.
type Firebase struct {
	name      string
	timestamp time.Time         // The last time the maps were refreshed
	client    *firestore.Client // Firebase app
	common
}

// NewFirebase creates a new instance of the Firebase structure
func NewFirebase(name string, project string) (*Firebase, error) {
	var f Firebase
	f.name = name
	ctx := context.Background()
	conf := &firebase.Config{ProjectID: project}
	app, err := firebase.NewApp(ctx, conf)

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	f.client = client

	f.mutex = &sync.Mutex{}
	err = f.refresh()
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (f *Firebase) getName() string {
	return f.name
}

func (f *Firebase) getReadOnly() bool {
	return false
}

func (f *Firebase) getNode(domain string) (*node, error) {
	n, err := f.common.getNode(domain)
	if err != nil {
		return nil, err
	}
	if n == nil {
		err = f.refresh()
		if err != nil {
			return nil, err
		}
		n, err = f.common.getNode(domain)
	}
	return n, err
}

func (f *Firebase) getNodes(network string) (*nodes, error) {
	ns, err := f.common.getNodes(network)
	if err != nil {
		return nil, err
	}
	if ns == nil {
		err = f.refresh()
		if err != nil {
			return nil, err
		}
		ns, err = f.common.getNodes(network)
	}
	return ns, err
}

// getAllNodes refreshes internal data and returns all nodes.
func (f *Firebase) getAllNodes() ([]*node, error) {
	err := f.refresh()
	if err != nil {
		return nil, err
	}
	return f.common.getAllNodes()
}

func (f *Firebase) iterateNodes(
	callback func(n *node, s interface{}) error,
	s interface{}) error {
	for _, n := range f.nodes {
		err := callback(n, s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *Firebase) setNode(n *node) error {
	err := f.setNodeSecrets(n)
	if err != nil {
		return err
	}
	ctx := context.Background()
	item := NodeItem{
		n.network,
		n.domain,
		n.created,
		n.expires.Unix(),
		n.role,
		n.scrambler.key}
	_, err2 := f.client.Collection(nodesTableName).Doc(n.domain).Set(ctx, item)
	return err2
}

func (f *Firebase) refresh() error {
	nets := make(map[string]*nodes)

	// Fetch the nodes and then add the secrets.
	ns, err := f.fetchNodes()
	if err != nil {
		return err
	}
	err = f.addSecrets(ns)
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
	f.mutex.Lock()
	f.nodes = ns
	f.networks = nets
	f.mutex.Unlock()

	return nil
}

func (f *Firebase) addSecrets(ns map[string]*node) error {
	ctx := context.Background()
	iter := f.client.Collection(secretsTableName).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var item SecretItem
		doc.DataTo(&item)
		s, err := newSecretFromKey(item.ScramblerKey, item.TimeStamp)
		if err != nil {
			return err
		}
		if ns[item.Domain] != nil {
			ns[item.Domain].addSecret(s)
		}
	}

	// Sort the secrets so the most recent is at the start of the array.
	for _, n := range ns {
		n.sortSecrets()
	}

	return nil
}

func (f *Firebase) fetchNodes() (map[string]*node, error) {
	ns := make(map[string]*node)
	ctx := context.Background()

	iter := f.client.Collection(nodesTableName).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var item NodeItem
		doc.DataTo(&item)
		ns[item.Domain], err = newNode(
			item.Network,
			item.Domain,
			item.Created,
			time.Unix(item.Expires, 0).UTC(),
			item.Role,
			item.ScramblerKey)
		if err != nil {
			return nil, err
		}
	}
	return ns, nil
}

func (f *Firebase) setNodeSecrets(n *node) error {
	ctx := context.Background()
	for _, s := range n.secrets {

		item := SecretItem{
			n.domain,
			s.timeStamp,
			n.expires.Unix(),
			s.key}

		_, _, err := f.client.Collection(secretsTableName).Add(ctx, item)
		if err != nil {
			return err
		}
	}
	return nil
}
