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
	s := NewServices(newConfigurationTest(), v, r)
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
	if o1.title != o2.title {
		fmt.Println(o1.title)
		fmt.Println(o2.title)
		t.Fail()
		return
	}
	if o1.message != o2.message {
		fmt.Println(o1.message)
		fmt.Println(o2.message)
		t.Fail()
		return
	}
	if o1.backgroundColor != o2.backgroundColor {
		fmt.Println(o1.backgroundColor)
		fmt.Println(o2.backgroundColor)
		t.Fail()
		return
	}
	if o1.messageColor != o2.messageColor {
		fmt.Println(o1.messageColor)
		fmt.Println(o2.messageColor)
		t.Fail()
		return
	}
	if o1.progressColor != o2.progressColor {
		fmt.Println(o1.progressColor)
		fmt.Println(o2.progressColor)
		t.Fail()
		return
	}
	if o1.timeStamp != o2.timeStamp {
		fmt.Println(o1.timeStamp)
		fmt.Println(o2.timeStamp)
		t.Fail()
		return
	}
	if o1.message != o2.message {
		fmt.Println(o1.title)
		fmt.Println(o2.title)
		t.Fail()
		return
	}
}
