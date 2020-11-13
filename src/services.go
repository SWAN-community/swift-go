/* ****************************************************************************
 * Copyright 2020 51 Degrees Mobile Experts Limited
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

// Services references all the information needed for every method.
type Services struct {
	config  Configuration   // Configuration used by the server.
	store   Store           // Instance of storage service for node data
	browser BrowserDetector // Service to provide browser warnings
}

// NewServices a set of services to use with Shared Web State. These provide
// defaults via the configuration parameter, and access to persistent storage
// via the store parameter.
func NewServices(
	config Configuration,
	store Store,
	browser BrowserDetector) *Services {
	var s Services
	s.config = config
	s.store = store
	s.browser = browser
	return &s
}

// Config returns the configuration service.
func (s *Services) Config() *Configuration { return &s.config }
