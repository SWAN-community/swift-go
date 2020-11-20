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

func TestOperation(t *testing.T) {
	v, err := newVolatileTest()
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	r, err := NewBrowserRegexes()
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	k := []string{"key"}
	a := NewAccessSimple(k)
	s := NewServices(newConfigurationTest(), v, a, r)
	o1 := newOperation(s, nil)
	b, err := o1.asByteArray()
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	n, err := v.getNode("test-1.com")
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	o2, err := newOperationFromByteArray(s, n, b)
	if err != nil {
		fmt.Println(err)
		t.Fail()
		return
	}
	if o1.Title() != o2.Title() {
		fmt.Println(o1.Title())
		fmt.Println(o2.Title())
		t.Fail()
		return
	}
	if o1.Message() != o2.Message() {
		fmt.Println(o1.Message())
		fmt.Println(o2.Message())
		t.Fail()
		return
	}
	if o1.BackgroundColor() != o2.BackgroundColor() {
		fmt.Println(o1.BackgroundColor())
		fmt.Println(o2.BackgroundColor())
		t.Fail()
		return
	}
	if o1.MessageColor() != o2.MessageColor() {
		fmt.Println(o1.MessageColor())
		fmt.Println(o2.MessageColor())
		t.Fail()
		return
	}
	if o1.ProgressColor() != o2.ProgressColor() {
		fmt.Println(o1.ProgressColor())
		fmt.Println(o2.ProgressColor())
		t.Fail()
		return
	}
	if o1.timeStamp != o2.timeStamp {
		fmt.Println(o1.timeStamp)
		fmt.Println(o2.timeStamp)
		t.Fail()
		return
	}
	if o1.Message() != o2.Message() {
		fmt.Println(o1.Title())
		fmt.Println(o2.Title())
		t.Fail()
		return
	}
}
