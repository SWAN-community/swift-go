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
	"testing"
	"time"
)

func TestNodesHashOrder(t *testing.T) {
	ns := newNodes()
	for i := 0; i < 100; i++ {
		var n *node
		s, err := newSecret()
		if err != nil {
			fmt.Println(err)
			t.Fail()
			return
		}
		n, err = newNode(
			"test",
			fmt.Sprintf("node%d", i),
			time.Now().UTC(),
			time.Now().UTC().AddDate(1, 0, 0),
			roleStorage,
			s.key)
		if err != nil {
			fmt.Println(err)
			t.Fail()
			return
		}
		x, err := newSecret()
		if err != nil {
			fmt.Println(err)
			t.Fail()
			return
		}
		n.addSecret(x)
		ns.all = append(ns.all, n)
		ns.dict[n.domain] = n
	}
	ns.order()
	a := ns.hash[50]
	i := ns.getNodeIndexByHash(a.hash)
	if 50 != i {
		t.Fail()
		return
	}
}
