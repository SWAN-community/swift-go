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
	"errors"
	"strings"
	"time"
)

// Character used to separate state elements.
const resultSeparator = "\r"

// Result from a storage operation.
type Result struct {
	Key     string    // The name of the key associated with the value
	Created time.Time // The UTC time that the value was created
	Expires time.Time // The UTC time that the value will expire
	Value   string    // The value as a byte array
}

// Results from a storage operation.
type Results struct {
	Expires time.Time // The time after which the data can not be decrypted
	Values  []*Result // Array of values
	HTML              // Include the common HTML UI members.
	State   []string  // Optional state information
}

// Get returns the result for the key provided, or nil if the key does not
// exist.
func (r *Results) Get(key string) *Result {
	for _, r := range r.Values {
		if key == r.Key {
			return r
		}
	}
	return nil
}

// IsTimeStampValid returns true if the time stamp of the result is valid.
func (r *Results) IsTimeStampValid() bool {
	return time.Now().UTC().Before(r.Expires)
}

// DecodeResults turns a byte array into a results data structure.
func DecodeResults(d []byte) (*Results, error) {
	var err error
	var r Results
	if d == nil {
		return nil, errors.New("Byte array empty")
	}
	b := bytes.NewBuffer(d)
	r.Expires, err = readTime(b)
	if err != nil {
		return nil, err
	}
	s, err := readString(b)
	if err != nil {
		return nil, err
	}
	r.State = strings.Split(s, resultSeparator)
	err = r.HTML.set(b)
	if err != nil {
		return nil, err
	}
	n, err := readByte(b)
	if err != nil {
		return nil, err
	}
	for i := byte(0); i < n; i++ {
		k, err := readString(b)
		if err != nil {
			return nil, err
		}
		c, err := readDate(b)
		if err != nil {
			return nil, err
		}
		e, err := readDate(b)
		if err != nil {
			return nil, err
		}
		v, err := readString(b)
		if err != nil {
			return nil, err
		}
		r.Values = append(r.Values, &Result{k, c, e, v})
	}
	return &r, nil
}

func encodeResults(r *Results) ([]byte, error) {
	var b bytes.Buffer
	var err error
	err = writeTime(&b, r.Expires)
	if err != nil {
		return nil, err
	}
	err = writeString(&b, strings.Join(r.State, resultSeparator))
	if err != nil {
		return nil, err
	}
	err = r.HTML.write(&b)
	if err != nil {
		return nil, err
	}
	err = writeByte(&b, byte(len(r.Values)))
	if err != nil {
		return nil, err
	}
	for _, e := range r.Values {
		err = writeString(&b, e.Key)
		if err != nil {
			return nil, err
		}
		err = writeDate(&b, e.Created)
		if err != nil {
			return nil, err
		}
		err = writeDate(&b, e.Expires)
		if err != nil {
			return nil, err
		}
		err = writeString(&b, e.Value)
		if err != nil {
			return nil, err
		}
	}
	return b.Bytes(), nil
}
