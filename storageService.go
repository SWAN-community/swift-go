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
	"sync"
	"time"
)

// storageService type is a background service which periodically refreshes the
// referenced storage manager. The storageService maintains a reference to the
// originally configured stores, these are used to initialize a new storage
// manager on each refresh.
type storageService struct {
	config Configuration   // Swift configuration
	store  *storageManager // Storage manager reference
	stores []Store         // List of stores that the service is initialized with
	ticker *time.Ticker    // Ticker reference
	mutex  *sync.Mutex     // mutex used to lock storage manager when updating
}

// NewStorageService creates a new instance of storageService and creates the
// initial instance of storageManager, a go routine is then started which
// will periodically refresh the storageManager reference with a new instance.
func NewStorageService(c Configuration, sts ...Store) storageService {
	var svc storageService
	var err error
	svc.config = c
	svc.stores = sts
	svc.mutex = &sync.Mutex{}

	svc.mutex.Lock()
	svc.store, err = newStorageManager(c, sts...)
	if err != nil {
		panic(err)
	}
	svc.mutex.Unlock()

	// start background goroutine to continuously refresh the store.
	go svc.startStorageService()

	return svc
}

// startStorageService creates a new ticker which, every time it executes,
// creates a new instance of storageManager and updates the reference in the
// storageService.
func (svc *storageService) startStorageService() {
	if svc.config.StorageManagerRefreshMinutes <= 0 {
		panic(fmt.Errorf("configuration for 'storageManagerRefreshMinutes' " +
			"is not set correctly, a positive value must be supplied"))
	}

	d := time.Duration(svc.config.StorageManagerRefreshMinutes) * time.Minute

	svc.ticker = time.NewTicker(d)
	defer svc.ticker.Stop()

	for _ = range svc.ticker.C {
		newStore, err := newStorageManager(svc.config, svc.stores...)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		if newStore != nil {
			svc.mutex.Lock()
			svc.store = newStore
			svc.mutex.Unlock()
		}
	}
}

// getNode abstracts calls to storageManager.getNode
func (svc *storageService) getNode(domain string) *node {
	return svc.store.getNode(domain)
}

// getNodes abstracts calls to storageManager.getNodes
func (svc *storageService) getNodes(network string) (*nodes, error) {
	return svc.store.getNodes(network)
}

// getAllNodes abstracts calls to storageManager.getAllNodes
func (svc *storageService) getAllNodes() ([]*node, error) {
	return svc.store.getAllNodes()
}

// setNodes abstracts calls to storageManager.setNodes
func (svc *storageService) setNodes(store string, ns ...*node) error {
	return svc.store.setNodes(store, ns...)
}

// GetStoreNames returns an array of names of all the writeable stores
func (svc *storageService) GetStoreNames() []string {
	var storeNames []string
	for _, s := range svc.stores {
		if !s.getReadOnly() {
			storeNames = append(storeNames, s.getName())
		}
	}
	return storeNames
}

// SetNode takes a register object and creates a new node, returns boolean
// for if successful or not and another boolean if this is an update operation.
func (s *storageService) SetNode(d *Register) (bool, bool) {

	// check if this is an update operation
	isUpdate := false
	if s.getNode(d.Domain) != nil {
		isUpdate = true
	}

	// Create a new scrambler for this new node.
	scrambler, err := newSecret()
	if err != nil {
		d.Error = err.Error()
		return false, isUpdate
	}

	// Create the new node ready to have it's secret added and stored.
	n, err := newNode(
		d.Network,
		d.Domain,
		time.Now().UTC(),
		d.Starts,
		d.Expires,
		d.Role,
		scrambler.key)
	if err != nil {
		d.Error = err.Error()
		return false, isUpdate
	}

	// Add the first secret to the node.
	x, err := newSecret()
	if err != nil {
		d.Error = err.Error()
		return false, isUpdate
	}
	n.addSecret(x)

	// Store the node and it successful mark the registration process as
	// complete.
	err = s.setNodes(d.Store, n)
	if err != nil {
		d.StoreError = err.Error()
	} else {
		d.ReadOnly = true
	}
	return true, isUpdate
}
