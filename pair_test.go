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
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestPair(t *testing.T) {
	var a pair
	var b pair
	a.key = "Test"
	a.created = time.Now().UTC()
	a.expires = time.Now().UTC()
	a.value = "Hello World"
	a.conflict = conflictNewest
	var out bytes.Buffer
	err := a.writeToBuffer(&out)
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	in := bytes.NewBuffer(out.Bytes())
	err = b.setFromBuffer(in)
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	if a.conflict != b.conflict {
		fmt.Println(a.conflict)
		fmt.Println(b.conflict)
		t.Fail()
	}
	if string(a.key) != string(b.key) {
		fmt.Println(string(a.key))
		fmt.Println(string(b.key))
		t.Fail()
	}
	if string(a.value) != string(b.value) {
		fmt.Println(string(a.value))
		fmt.Println(string(b.value))
		t.Fail()
	}
	testCompareDate(t, a.created, b.created)
	testCompareDate(t, a.expires, b.expires)
}
