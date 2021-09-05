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

import "fmt"

// Volatile localstorage implementation
type Volatile struct {
	name     string
	readOnly bool
	common
}

func newVolatile(name string, readOnly bool, ns []*node) *Volatile {
	var v Volatile
	v.name = name
	v.readOnly = readOnly
	v.init(ns)
	return &v
}

func (v *Volatile) getName() string {
	return v.name
}

func (v *Volatile) getNode(domain string) (*node, error) {
	return v.common.getNode(domain)
}

func (v *Volatile) getNodes(network string) (*nodes, error) {
	return v.common.getNodes(network)
}

func (v *Volatile) getReadOnly() bool {
	return v.readOnly
}

func (v *Volatile) iterateNodes(
	callback func(n *node, s interface{}) error,
	s interface{}) error {
	for _, n := range v.common.nodes {
		err := callback(n, s)
		if err != nil {
			return err
		}
	}
	return nil
}

func (v *Volatile) setNode(n *node) error {
	if v.readOnly {
		return fmt.Errorf("store '%s' is read only", v.name)
	}

	var net *nodes
	v.nodes[n.domain] = n
	net = v.networks[n.network]
	if net == nil {
		net = newNodes()
		v.networks[n.network] = net
	}
	net.dict[n.domain] = n
	net.all = append(net.all, n)
	return nil
}
