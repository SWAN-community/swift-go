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

	"github.com/SWAN-community/common-go"
)

// Character used to separate state elements.
const resultSeparator = "\r"

// Results from a storage operation.
type Results struct {
	HTML              // Include the common HTML UI members.
	expires time.Time // The time after which the data can not be decrypted
	pairs   []*Pair   // Array of key value pairs
	state   []string  // Optional state information
}

// Pairs common.Readonly accessor to the results's key value pairs.
func (r *Results) Pairs() []*Pair { return r.pairs }

// State common.Readonly accessor to the results's state array.
func (r *Results) State() []string { return r.state }

// Get returns the result for the key provided, or nil if the key does not
// exist.
func (r *Results) Get(key string) *Pair {
	for _, r := range r.pairs {
		if key == r.key {
			return r
		}
	}
	return nil
}

// Map returns the results as a map, keyed on the pair key.
func (r *Results) Map() map[string]*Pair {
	p := make(map[string]*Pair)
	for _, r := range r.pairs {
		p[r.key] = r
	}
	return p
}

// IsTimeStampValid returns true if the time stamp of the result is valid.
func (r *Results) IsTimeStampValid() bool {
	return time.Now().UTC().Before(r.expires)
}

// DecodeResults turns a byte array into a results data structure.
func DecodeResults(d []byte) (*Results, error) {
	var err error
	var r Results
	if d == nil {
		return nil, errors.New("Byte array empty")
	}
	b := bytes.NewBuffer(d)
	r.expires, err = common.ReadTime(b)
	if err != nil {
		return nil, err
	}
	s, err := common.ReadString(b)
	if err != nil {
		return nil, err
	}
	r.state = strings.Split(s, resultSeparator)
	err = r.HTML.set(b)
	if err != nil {
		return nil, err
	}
	n, err := common.ReadByte(b)
	if err != nil {
		return nil, err
	}
	for i := byte(0); i < n; i++ {
		k, err := common.ReadString(b)
		if err != nil {
			return nil, err
		}
		c, err := common.ReadDate(b)
		if err != nil {
			return nil, err
		}
		e, err := common.ReadDate(b)
		if err != nil {
			return nil, err
		}
		v, err := common.ReadByteArrayArray(b)
		if err != nil {
			return nil, err
		}
		r.pairs = append(r.pairs, &Pair{k, c, e, v})
	}
	return &r, nil
}

func encodeResults(r *Results) ([]byte, error) {
	var b bytes.Buffer
	var err error
	err = common.WriteTime(&b, r.expires)
	if err != nil {
		return nil, err
	}
	err = common.WriteString(&b, strings.Join(r.state, resultSeparator))
	if err != nil {
		return nil, err
	}
	err = r.HTML.write(&b)
	if err != nil {
		return nil, err
	}
	err = common.WriteByte(&b, byte(len(r.pairs)))
	if err != nil {
		return nil, err
	}
	for _, e := range r.pairs {
		err = common.WriteString(&b, e.key)
		if err != nil {
			return nil, err
		}
		err = common.WriteDate(&b, e.created)
		if err != nil {
			return nil, err
		}
		err = common.WriteDate(&b, e.expires)
		if err != nil {
			return nil, err
		}
		err = common.WriteByteArrayArray(&b, e.values)
		if err != nil {
			return nil, err
		}
	}
	return b.Bytes(), nil
}
