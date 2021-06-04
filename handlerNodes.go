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
	"net/http"
	"time"
)

// NodeView is a struct containing the node fields to display in the nodes
// swiftNodesTemplate
type NodeView struct {
	Network  string    // The name of the network the node belongs to
	Domain   string    // The domain name associated with the node
	Created  time.Time // The time that the node first came online
	Starts   time.Time // The time that the node will begin operation
	Expires  time.Time // The time that the node will retire from the network
	Role     int       // The role the node has in the network
	Accessed time.Time // The time the node was last accessed
	Alive    bool      // True if the node is reachable via a HTTP request
}

// NodeViews is a struct which contains an array of NodeView which is used
// to display a list of nodes using the swiftNodesTemplate
type NodeViews struct {
	Nodes []NodeView
}

// Get the NodeView
func (nv *NodeViews) NodeViewItems() []NodeView {
	return nv.Nodes
}

// HandlerNodes is a handler that returns a list of all the known nodes, each
// node is converted into a NodeView item which is then used to populate an HTML
// template.
func HandlerNodes(s *Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var nvs NodeViews

		ns, err := s.store.getAllNodes()
		if err != nil {
			returnAPIError(s, w, err, http.StatusInternalServerError)
		}

		for _, n := range ns {
			nv := NodeView{
				Network:  n.network,
				Domain:   n.domain,
				Created:  n.created,
				Starts:   n.starts,
				Expires:  n.expires,
				Role:     n.role,
				Accessed: n.accessed,
				Alive:    n.alive,
			}
			nvs.Nodes = append(nvs.Nodes, nv)
		}

		sendHTMLTemplate(s, w, swiftNodesTemplate, &nvs)
	}
}
