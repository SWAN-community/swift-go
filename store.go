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

const (
	nodesTableName        = "swiftnodes"   // Table name for nodes
	secretsTableName      = "swiftsecrets" // Table name for secrets
	domainFieldName       = "domain"       // The domain of the node
	networkFieldName      = "network"      // The network of the node
	roleFieldName         = "role"         // The role of the node
	expiresFieldName      = "expires"      // When the node expires
	scramblerKeyFieldName = "scrambler"    // Used to scramble table and key names
)

// Store interface for persistent data shared across instances operated.
type Store interface {

	// GetAccessNode returns the access node for the network.
	GetAccessNode(network string) (string, error)

	// GetNode takes a domain name and returns the associated node. If a node
	// does not exist then nil is returned.
	getNode(domain string) (*node, error)

	// GetNodes returns all the nodes associated with a network.
	getNodes(network string) (*nodes, error)

	// SetNode inserts or updates the node.
	setNode(node *node) error
}
