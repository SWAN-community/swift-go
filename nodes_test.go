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
	"log"
	"testing"
	"time"
)

// TestNodesHashOrder confirms that the hashes appear in the correct order.
func TestNodesHashOrder(t *testing.T) {
	ns, err := createNodes()
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	a := ns.hash[50]
	i := ns.getNodeIndexByHash(a.hash)
	if 50 != i {
		fmt.Println(err)
		t.Fail()
		return
	}
}

// TestNodesHomeNodeMultiNetwork validates that two instances of nodes
// collections return the same values for the same input data.
func TestNodesHomeNodeMultiNetwork(t *testing.T) {
	ns1, err := createNodes()
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	ns2, err := createNodes()
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	hn1, err := ns1.getHomeNode("212.36.33.158, 172.31.23.19", "127.0.0.1")
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	hn2, err := ns2.getHomeNode("212.36.33.158, 172.31.23.19", "127.0.0.1")
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	if hn1.domain != hn2.domain {
		fmt.Println(hn1.domain)
		fmt.Println(hn2.domain)
		t.Fail()
		return
	}
}

// TestNodesHomeNode confirms that similar input data produces different
// outputs.
func TestNodesHomeNode(t *testing.T) {
	ns, err := createNodes()
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	hn1, err := ns.getHomeNode("212.36.33.158, 172.31.23.19", "127.0.0.1")
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	hn2, err := ns.getHomeNode("109.249.187.121, 172.31.23.19", "127.0.0.1")
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	hn3, err := ns.getHomeNode("109.249.187.120, 172.31.23.19", "127.0.0.1")
	log.Println(hn1.domain)
	log.Println(hn2.domain)
	log.Println(hn3.domain)
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	if hn1.domain == hn2.domain ||
		hn2.domain == hn3.domain ||
		hn1.domain == hn3.domain {
		fmt.Println(err)
		t.Fail()
		return
	}
}

func createNodes() (*nodes, error) {
	ns := newNodes()
	for i := 0; i < 100; i++ {
		var n *node
		s, err := newSecret()
		if err != nil {
			return nil, err
		}
		n, err = newNode(
			"test",
			fmt.Sprintf("node%d", i),
			time.Now().UTC(),
			time.Now().UTC(),
			time.Now().UTC().AddDate(1, 0, 0),
			roleStorage,
			s.key)
		if err != nil {
			return nil, err
		}
		x, err := newSecret()
		if err != nil {
			return nil, err
		}
		n.addSecret(x)
		ns.all = append(ns.all, n)
		ns.dict[n.domain] = n
	}
	ns.order()
	return ns, nil
}
