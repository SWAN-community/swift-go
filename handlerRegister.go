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
	"strconv"
	"time"
)

// HandlerRegister takes a Services pointer and returns a HTTP handler used to
// register a domain as an Access Node or a Storage Node. Does not work after
// the domain has been registered in the storage service.
func HandlerRegister(s *Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		var d Register
		d.StoreNames = s.store.GetStoreNames()
		d.Store = ""
		d.request = r
		d.Services = s
		d.Domain = r.Host
		d.Starts = time.Now().UTC().AddDate(0, 0, 1)
		d.Network = ""
		d.Expires = time.Now().UTC().AddDate(0, 3, 0)
		d.Role = roleStorage
		d.Secret = true
		d.Scramble = true
		d.CookieDomain = r.Host

		// Check that the domain has not already been registered.
		n := s.store.getNode(r.Host)
		if n != nil {
			return
		}

		// Get any values from the form.
		err = r.ParseForm()
		if err != nil {
			returnServerError(s, w, err)
			return
		}
		d.DisplayErrors = len(r.Form) > 0

		// Get the store information
		d.Store = r.FormValue("store")

		// Get the network information.
		d.Network = r.FormValue("network")
		if len(d.Network) <= 3 {
			d.NetworkError = "Network must be longer than 3 characters"
		} else if len(d.Network) > 20 {
			d.NetworkError = "Network can not be longer than 20 characters"
		}

		// Get the role information.
		if r.FormValue("role") != "" {
			d.Role, err = strconv.Atoi(r.FormValue("role"))
			if err != nil {
				d.RoleError = err.Error()
			} else if d.Role != roleAccess &&
				d.Role != roleStorage &&
				d.Role != roleShare {
				d.RoleError = fmt.Sprintf("Role '%d' invalid", d.Role)
			}
		}

		// Get the node expiry information.
		if r.FormValue("expires") != "" {
			d.Expires, err = time.Parse("2006-01-02", r.FormValue("expires"))
			if err != nil {
				d.ExpiresError = err.Error()
			} else if d.Expires.Before(time.Now().UTC()) {
				d.ExpiresError = "Expiry date must be in the future"
			}
		}

		// Get the node starts information.
		if r.FormValue("starts") != "" {
			d.Starts, err = time.Parse("2006-01-02T15:04", r.FormValue("starts"))
			if err != nil {
				d.StartsError = err.Error()
			}
		}

		// Get the secrets, scramble and cookie domain.
		if r.FormValue("cookieDomain") != "" {
			d.CookieDomain = r.FormValue("cookieDomain")
		}
		d.Secret = r.FormValue("secret") == "true" ||
			r.FormValue("secret") == "yes" ||
			r.FormValue("secret") == "1"
		d.Scramble = r.FormValue("scramble") == "true" ||
			r.FormValue("scramble") == "yes" ||
			r.FormValue("scramble") == "1"

		// If the form data is valid then store the new node.
		if d.ExpiresError == "" &&
			d.RoleError == "" &&
			d.NetworkError == "" {
			storeNode(s, &d)
		}

		// Return the HTML page.
		sendHTMLTemplate(s, w, registerTemplate, &d)
	}
}

func storeNode(s *Services, d *Register) {

	// Create a new scrambler for this new node.
	scramblerKey := ""
	if d.Scramble {
		scrambler, err := newSecret()
		if err != nil {
			d.Error = err.Error()
			return
		}
		scramblerKey = scrambler.key
	}

	// Create the new node ready to have it's secret added and stored.
	n, err := newNode(
		d.Network,
		d.Domain,
		time.Now().UTC(),
		d.Starts.UTC(),
		d.Expires,
		d.Role,
		scramblerKey,
		d.CookieDomain)
	if err != nil {
		d.Error = err.Error()
		return
	}

	// Add the first secret to the node if secrets are to be used.
	if d.Secret {
		x, err := newSecret()
		if err != nil {
			d.Error = err.Error()
			return
		}
		n.addSecret(x)
	} else {
		n.secrets = []*secret{}
	}

	// Store the node and it successful mark the registration process as
	// complete.
	err = s.store.setNodes(d.Store, n)
	if err != nil {
		d.StoreError = err.Error()
	} else {
		d.ReadOnly = true
	}
}
