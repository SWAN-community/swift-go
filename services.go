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
	"net/http"
)

// Services references all the information needed for every method.
type Services struct {
	config  Configuration   // Configuration used by the server.
	store   storageService  // Instance of storage service for node data
	browser BrowserDetector // Service to provide browser warnings
	access  Access          // Instance of the access control interface
}

// NewServices a set of services to use with SWIFT. These provide defaults via
// the configuration parameter, and access to persistent storage via the store
// parameter.
func NewServices(
	config Configuration,
	store storageService,
	access Access,
	browser BrowserDetector) *Services {
	var s Services
	s.config = config
	s.store = store
	s.access = access
	s.browser = browser
	return &s
}

// Config returns the configuration service.
func (s *Services) Config() *Configuration { return &s.config }

// GetAccessNodeForHost returns the access node, if there is one, for the host
// name provided. If the host does not exist then an error is returned. If the
// host exists, but is not an access node then an error is returned.
// h is the internet domain of the SWIFT access node host
func (s *Services) GetAccessNodeForHost(h string) (*node, error) {
	return s.getNodeFromRequest(h, roleAccess)
}

// GetHomeNode returns the home node for the web browser associated with the
// access node processing the request. If the current request is not to an
// access node then an error will be returned.
func (s *Services) GetHomeNode(r *http.Request) (*node, error) {
	q := r.Form
	h, err := s.GetAccessNodeForHost(r.Host)
	if err != nil {
		return nil, err
	}
	n, err := s.store.getNodes(h.network)
	if err != nil {
		return nil, err
	}
	return n.getHomeNode(q.Get(xforwarededfor), q.Get(remoteAddr))
}

func (s *Services) getStorageNode(r *http.Request) (*node, error) {
	return s.getNodeFromRequest(r.Host, roleStorage)
}

func (s *Services) getNodeFromRequest(h string, q int) (*node, error) {

	// Get the node associated with the request.
	n := s.store.getNode(h)
	// Verify that a node was found.
	if n == nil {
		return nil, fmt.Errorf("No access node for '%s'", h)
	}

	// Verify that this node is the right type.
	if n.role != q {
		return nil, fmt.Errorf("Node '%s' incorrect type", n.domain)
	}

	return n, nil
}

// Returns true if the request is allowed to access the handler, otherwise
// false. Removes the accessKey parameter from the form to prevent it being
// used by other methods.  If false is returned then no further action is
// needed as the method will have responded to the request already.
func (s *Services) getAccessAllowed(
	w http.ResponseWriter,
	r *http.Request) bool {
	err := r.ParseForm()
	if err != nil {
		returnAPIError(s, w, err, http.StatusInternalServerError)
		return false
	}
	v, err := s.access.GetAllowed(r.FormValue("accessKey"))
	if v == false || err != nil {
		returnAPIError(
			s,
			w,
			fmt.Errorf("Access denied"),
			http.StatusNetworkAuthenticationRequired)
		return false
	}
	r.Form.Del("accessKey")
	return true
}
