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
	"fmt"
	"log"
	"os"
)

const (
	nodesTableName        = "swiftnodes"   // Table name for nodes
	secretsTableName      = "swiftsecrets" // Table name for secrets
	domainFieldName       = "Domain"       // The domain of the node
	networkFieldName      = "Network"      // The network of the node
	roleFieldName         = "role"         // The role of the node
	startsFieldName       = "starts"       // When the node begins operation
	expiresFieldName      = "expires"      // When the node expires
	scramblerKeyFieldName = "ScramblerKey" // Used to scramble table and key names
)

// Store interface for persistent data shared across instances operated.
type Store interface {

	// getName returns the human readable name for the store
	getName() string

	// getNode return details of node if store knows about node
	getNode(domain string) (*node, error)

	// getNodes returns nodes
	getNodes(network string) (*nodes, error)

	// getReadonly returns true if the store does not support inserts and updates.
	getReadOnly() bool
	// iterateNodes call the callback for every node
	// n is the node
	// s is the state for the function
	iterateNodes(callback func(n *node, s interface{}) error, s interface{}) error

	// setNode inserts or updates the node if the store supports inserts and
	// updates
	setNode(n *node) error
}

// NewStore returns a work implementation of the Store interface for the
// configuration supplied.
func NewStore(swiftConfig Configuration) []Store {
	var swiftStores []Store

	azureAccountName, azureAccountKey, gcpProject, swiftSecrets, swiftNodes, awsLoadConfig :=
		os.Getenv("AZURE_STORAGE_ACCOUNT"),
		os.Getenv("AZURE_STORAGE_ACCESS_KEY"),
		os.Getenv("GCP_PROJECT"),
		os.Getenv("SWIFT_SECRETS_FILE"),
		os.Getenv("SWIFT_NODES_FILE"),
		os.Getenv("AWS_SDK_LOAD_CONFIG")
	if len(azureAccountName) > 0 || len(azureAccountKey) > 0 {
		log.Printf("SWIFT: Using Azure Table Storage")
		if len(azureAccountName) == 0 || len(azureAccountKey) == 0 {
			panic(errors.New("Either the AZURE_STORAGE_ACCOUNT or " +
				"AZURE_STORAGE_ACCESS_KEY environment variable is not set"))
		}
		swiftStore, err := NewAzure(azureAccountName, azureAccountKey)
		if err != nil {
			panic(err)
		}
		swiftStores = append(swiftStores, swiftStore)
	}
	if len(gcpProject) > 0 {
		log.Printf("SWIFT: Using Google Firebase")
		swiftStore, err := NewFirebase(gcpProject)
		if err != nil {
			panic(err)
		}
		swiftStores = append(swiftStores, swiftStore)
	}
	if len(swiftSecrets) > 0 ||
		len(swiftNodes) > 0 {
		log.Printf("SWIFT: Using local storage")
		if len(swiftSecrets) == 0 {
			panic(errors.New("The SWIFT_SECRETS_FILE environment variable " +
				"is not set"))
		}
		if len(swiftNodes) == 0 {
			panic(errors.New("The SWIFT_NODES_FILE environment variable " +
				"is not set"))
		}
		swiftStore, err := NewLocalStore(swiftSecrets, swiftNodes)
		if err != nil {
			panic(err)
		}
		swiftStores = append(swiftStores, swiftStore)
	}
	if len(awsLoadConfig) > 0 {
		log.Printf("SWIFT: Using AWS DynamoDB")
		swiftStore, err := NewAWS()
		if err != nil {
			panic(err)
		}
		swiftStores = append(swiftStores, swiftStore)
	}

	if len(swiftStores) == 0 {
		panic(fmt.Errorf("SWIFT: no store has been configured. " +
			"Provide details for store by specifying one or more sets of " +
			"environment variables\r\n: " +
			"(1) Azure Storage account details 'AZURE_STORAGE_ACCOUNT' & 'AZURE_STORAGE_ACCESS_KEY'\r\n" +
			"(2) GCP project in 'GCP_PROJECT' \r\n" +
			"(3) Local storage file paths in 'SWIFT_SECRETS_FILE' & 'SWIFT_NODES_FILE'\r\n" +
			"(4) AWS Dynamo DB by setting 'AWS_SDK_LOAD_CONFIG' to true\r\n" +
			"Refer to https://github.com/SWAN-community/swift-go/blob/main/README.md " +
			"for specifics on setting up each storage solution"))
	}

	return swiftStores
}
