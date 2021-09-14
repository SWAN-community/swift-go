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
)

// A test secret of 32 bytes
var testSecret = []byte("passphrasewhichneedstobe32bytes!")

func TestCryptoGood(t *testing.T) {
	testCryptoString(t, "Share Web State")
}

func TestCryptoEmpty(t *testing.T) {
	testCryptoString(t, "")
}

func TestCryptoCorrupt(t *testing.T) {
	x, err := newCrypto(testSecret)
	if err != nil {
		t.Fail()
	}
	c, err := x.encrypt([]byte("corrupt"))
	if err != nil {
		t.Fail()
	}
	c = append(c, []byte{0}...)
	_, err = x.decrypt(c)
	if err == nil {
		t.Fail()
	}
	fmt.Println(err)
}

func testCryptoString(t *testing.T, s string) {
	i := []byte(s)
	o, err := testCryptoByteArray(i)
	if err != nil {
		fmt.Println(err)
		t.Fail()
	} else {
		if bytes.Compare(i, o) != 0 {
			fmt.Println(string(i))
			fmt.Println(string(o))
			t.Fail()
		}
	}
}

func testCryptoByteArray(i []byte) ([]byte, error) {
	x, err := newCrypto(testSecret)
	if err != nil {
		return nil, err
	}
	c, err := x.encrypt(i)
	if err != nil {
		return nil, err
	}
	return x.decrypt(c)
}
