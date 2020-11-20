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
	"testing"
)

func TestStorageCommon(t *testing.T) {
	var err error
	var an *node
	var bn *node
	s, err := newVolatileTest()
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	an, err = s.getNode("test-1.com")
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	bn, err = s.getNode("test-1.com")
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	if an.created != bn.created {
		fmt.Printf(
			"Created a '%s' does not match b '%s'",
			an.created,
			bn.created)
		t.Fail()
	}
	if an.expires != bn.expires {
		fmt.Printf(
			"Expires a '%s' does not match b '%s'",
			an.expires,
			bn.expires)
		t.Fail()
	}
	if an.role != bn.role {
		fmt.Printf(
			"Role a '%d' does not match b '%d'",
			an.role,
			bn.role)
		t.Fail()
	}
	if an.domain != bn.domain {
		fmt.Printf(
			"Domain a '%s' does not match b '%s'",
			an.domain,
			bn.domain)
		t.Fail()
	}
}
