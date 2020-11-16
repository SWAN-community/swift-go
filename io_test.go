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

func TestIoTime(t *testing.T) {
	d := time.Now().UTC()
	var b bytes.Buffer
	err := writeDate(&b, d)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	i := b.Bytes()
	c := bytes.NewBuffer(i)
	r, err := readDate(c)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	testCompareDate(t, r, d)
}

func testCompareDate(t *testing.T, a time.Time, b time.Time) {
	if a.Year() != b.Year() {
		fmt.Printf("Year %d != %d", a.Year(), b.Year())
		t.Fail()
	}
	if a.Month() != b.Month() {
		fmt.Printf("Month %d != %d", a.Month(), b.Month())
		t.Fail()
	}
	if a.Day() != b.Day() {
		fmt.Printf("Day %d != %d", a.Day(), b.Day())
		t.Fail()
	}
}
