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
	timestamp time.Time         // The last time the maps were refreshed
	client    *firestore.Client // Firebase app
	common
}

// NewFirebase creates a new instance of the Firebase structure
func NewFirebase(project string) (*Firebase, error) {
	var f Firebase

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

func (a *Firebase) getNode(domain string) (*Node, error) {
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

func (a *Firebase) getNodes(network string) (*nodes, error) {
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

func (f *Firebase) setNode(node *Node) error {
	err := f.setNodeSecrets(node)
	if err != nil {
		return err
	}
	ctx := context.Background()
	item := NodeItem{
		node.network,
		node.domain,
		node.created,
		node.expires.Unix(),
		node.role,
		node.scrambler.key}
	_, err2 := f.client.Collection(nodesTableName).Doc(node.domain).Set(ctx, item)
	return err2
}

func (a *Firebase) refresh() error {
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
	a.mutex.Lock()
	a.nodes = ns
	a.networks = nets
	a.mutex.Unlock()

	return nil
}

func (f *Firebase) addSecrets(ns map[string]*Node) error {
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

func (f *Firebase) fetchNodes() (map[string]*Node, error) {
	ns := make(map[string]*Node)
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

func (f *Firebase) setNodeSecrets(node *Node) error {
	ctx := context.Background()
	for _, s := range node.secrets {

		item := SecretItem{
			node.domain,
			s.timeStamp,
			node.expires.Unix(),
			s.key}

		_, _, err := f.client.Collection(secretsTableName).Add(ctx, item)
		if err != nil {
			return err
		}
	}
	return nil
}
