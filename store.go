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
	"errors"
	"log"
	"os"
)

const (
	nodesTableName        = "swiftnodes"   // Table name for nodes
	secretsTableName      = "swiftsecrets" // Table name for secrets
	domainFieldName       = "Domain"       // The domain of the node
	networkFieldName      = "Network"      // The network of the node
	roleFieldName         = "role"         // The role of the node
	expiresFieldName      = "expires"      // When the node expires
	scramblerKeyFieldName = "ScramblerKey" // Used to scramble table and key names
)

// Store interface for persistent data shared across instances operated.
type Store interface {

	// GetNode takes a domain name and returns the associated node. If a node
	// does not exist then nil is returned.
	getNode(domain string) (*Node, error)

	// GetNodes returns all the nodes associated with a network.
	getNodes(network string) (*nodes, error)

	// SetNode inserts or updates the node.
	setNode(node *Node) error
}

// NewStore returns a work implementation of the Store interface for the
// configuration supplied.
func NewStore(swiftConfig Configuration) Store {
	var swiftStore Store
	var err error

	azureAccountName, azureAccountKey, gcpProject :=
		os.Getenv("AZURE_STORAGE_ACCOUNT"),
		os.Getenv("AZURE_STORAGE_ACCESS_KEY"),
		os.Getenv("GCP_PROJECT")
	if len(azureAccountName) > 0 || len(azureAccountKey) > 0 {
		log.Printf("SWIFT: Using Azure Table Storage")
		if len(azureAccountName) == 0 || len(azureAccountKey) == 0 {
			panic(errors.New("Either the AZURE_STORAGE_ACCOUNT or " +
				"AZURE_STORAGE_ACCESS_KEY environment variable is not set."))
		}
		swiftStore, err = NewAzure(
			azureAccountName,
			azureAccountKey)
		if err != nil {
			panic(err)
		}
	} else if len(gcpProject) > 0 {
		log.Printf("SWIFT: Using Google Firebase")
		swiftStore, err = NewFirebase(gcpProject)
		if err != nil {
			panic(err)
		}
	} else {
		log.Printf("SWIFT: Using AWS DynamoDB")
		swiftStore, err = NewAWS()
		if err != nil {
			panic(err)
		}
	}

	if swiftStore == nil {
		panic(errors.New("SWIFT: store not configured, have you set AWS " +
			" OR Azure Storage Table credentials?"))
	}

	return swiftStore
}
