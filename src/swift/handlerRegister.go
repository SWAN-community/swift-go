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

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func handlerRegister(s *Services) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var d Register
		d.Services = s
		d.Domain = r.Host
		d.Network = ""
		d.Expires = time.Now().UTC().AddDate(0, 3, 0)
		d.Role = roleStorage

		// Check that the domain has not already been registered.
		n, err := s.store.getNode(r.Host)
		if err != nil {
			returnError(s, w, err)
			return
		}
		if n != nil {
			return
		}

		// Get any values from the form.
		err = r.ParseForm()
		if err != nil {
			returnError(s, w, err)
			return
		}
		d.DisplayErrors = len(r.Form) > 0

		// Get the network information.
		d.Network = r.FormValue("network")
		if len(d.Network) <= 5 {
			d.NetworkError = "Network must be longer than 5 characters"
		} else if len(d.Network) > 20 {
			d.NetworkError = "Network can not be longer than 20 characters"
		}

		// Get the role information.
		if r.FormValue("role") != "" {
			d.Role, err = strconv.Atoi(r.FormValue("role"))
			if err != nil {
				d.RoleError = err.Error()
			} else if d.Role != roleAccess && d.Role != roleStorage {
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

		// If the form data is valid then store the new node.
		if d.ExpiresError == "" &&
			d.RoleError == "" &&
			d.NetworkError == "" {
			storeNode(s, &d)
		}

		// Return the HTML page.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		err = registerTemplate.Execute(w, &d)
		if err != nil {
			returnError(s, w, err)
		}
	}
}

func storeNode(s *Services, d *Register) {

	// Create a new scrambler for this new node.
	scrambler, err := newSecret()
	if err != nil {
		d.Error = err.Error()
		return
	}

	// Create the new node ready to have it's secret added and stored.
	n, err := newNode(
		d.Network,
		d.Domain,
		time.Now().UTC(),
		d.Expires,
		d.Role,
		scrambler.key)
	if err != nil {
		d.Error = err.Error()
		return
	}

	// Add the first secret to the node.
	x, err := newSecret()
	if err != nil {
		d.Error = err.Error()
		return
	}
	n.addSecret(x)

	// Store the node and it successful mark the registration process as
	// complete.
	err = s.store.setNode(n)
	if err != nil {
		d.Error = err.Error()
	} else {
		d.ReadOnly = true
	}
}
