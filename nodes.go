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
	"math/rand"
	"regexp"
	"sort"
)

type nodes struct {
	all    []*Node          // All the nodes in a random order
	active []*Node          // Active nodes ordered by creation time
	hash   []*Node          // Active storage nodes ordered by hash value
	dict   map[string]*Node // All the nodes keyed on domain name
}

func newNodes() *nodes {
	var ns nodes
	ns.all = []*Node{}
	ns.active = []*Node{}
	ns.hash = []*Node{}
	ns.dict = make(map[string]*Node)
	return &ns
}

func (ns *nodes) getRandomNode(condition func(n *Node) bool) *Node {
	indexes := make([]int, len(ns.active))
	for i := 0; i < len(ns.active); i++ {
		indexes[i] = i
	}
	for i := range indexes {
		j := rand.Intn(i + 1)
		indexes[i], indexes[j] = indexes[j], indexes[i]
	}
	for _, i := range indexes {
		if condition(ns.active[indexes[i]]) {
			return ns.active[indexes[i]]
		}
	}
	return nil
}

// Get the hash of the remote address for the request by removing the port if
// present and using the domain or IP address.
func getRemoteAddrHash(xff string, ra string) uint64 {
	var a uint64
	d := getRemoteAddr(xff, ra)
	if len(d) > 0 {
		a = getHash(d)
	}
	return a
}

var regexClientIP, _ = regexp.Compile("[\\d\\.]+|\\[[^\\]]+\\]")

// GetIP gets a requests IP address by reading off the forwarded-for header
// (for proxies) and falls back to use the remote address.
func getRemoteAddr(xff string, ra string) string {
	if xff != "" {
		b := regexClientIP.FindString(xff)
		if b != "" {
			return b
		}
		return xff
	}
	if ra != "" {
		b := regexClientIP.FindString(ra)
		if b != "" {
			return b
		}
		return ra
	}
	return ""
}

// Find the node that has a hash value closest to that of the remote IP address.
func (ns *nodes) getHomeNode(xff string, ra string) (*Node, error) {
	i := ns.getNodeIndexByHash(getRemoteAddrHash(xff, ra))
	if i < 0 || i >= len(ns.hash) {
		return nil, fmt.Errorf(
			"None of the '%d' available nodes were identified as a home node "+
				"for remote address '%s'",
			len(ns.hash),
			getRemoteAddr(xff, ra))
	}
	return ns.hash[i], nil
}

func (ns *nodes) getNodeIndexByHash(h uint64) int {
	m := 0
	l := 0
	u := len(ns.hash) - 1
	for l <= u {
		m = (l + u) / 2
		if ns.hash[m].hash < h {
			l = m + 1
		} else if ns.hash[m].hash > h {
			u = m - 1
		} else {
			break
		}
	}
	return m
}

func (ns *nodes) order() {
	ns.active = getActiveOrdered(ns.all)
	ns.hash = getHashOrdered(ns.active)
}

func getHashOrdered(active []*Node) []*Node {
	h := make([]*Node, 0, len(active))
	for _, n := range active {
		if n.role == roleStorage {
			h = append(h, n)
		}
	}
	sort.Slice(h, func(i, j int) bool {
		return h[i].hash < h[j].hash
	})
	return h
}

func getActiveOrdered(all []*Node) []*Node {
	a := make([]*Node, 0, len(all))
	for _, n := range all {
		if n.isActive() {
			a = append(a, n)
		}
	}
	sort.Slice(a, func(i, j int) bool {
		return a[i].created.Before(a[j].created)
	})
	return a
}
