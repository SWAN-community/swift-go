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

type storageService struct {
	config Configuration   // Swift configuration
	store  *storageManager // Storage manager reference
	stores []Store         // List of stores that the service is initialized with
	ticker *time.Ticker    // Ticker reference
	mutex  *sync.Mutex     // mutex used to lock storage manager when updating
}

// NewStorageService
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

func (svc *storageService) getNode(domain string) *node {
	return svc.store.getNode(domain)
}

func (svc *storageService) getNodes(network string) (*nodes, error) {
	return svc.store.getNodes(network)
}

func (svc *storageService) getAllNodes() ([]*node, error) {
	return svc.store.getAllNodes()
}

func (svc *storageService) setNodes(ns ...*node) error {
	return svc.store.setNodes(ns...)
}
