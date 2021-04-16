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
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

const (
	conflictInvalid = iota // Used to ensure the byte has been initialized
	conflictOldest  = iota
	conflictNewest  = iota
	conflictAdd     = iota
)

// An empty pair referenced in the resolveConflict method if both parameters are
// null.
var emptyValue pair

// Pair from a storage operation.
type Pair struct {
	key     string    // The name of the key associated with the value
	created time.Time // The UTC time that the value was created
	expires time.Time // The UTC time that the value will expire
	values  [][]byte  // The values as byte arrays
}

// pair used internally and adds more information for the operation.
type pair struct {
	Pair
	conflict        byte      // Flag for conflict resolution
	cookieWriteTime time.Time // Last time the cookie was written to
}

// Key readonly accessor to the pair's key.
func (p *Pair) Key() string { return p.key }

// Created readonly accessor to the pair's created time.
func (p *Pair) Created() time.Time { return p.created }

// Expires readonly accessor to the pair's expiry time.
func (p *Pair) Expires() time.Time { return p.expires }

// Value readonly accessor to the pair's value.
func (p *Pair) Values() [][]byte { return p.values }

// Value returns the value as string. Used with HTML templates or JSON
// serialization.
func (p *Pair) Value() string {
	var s = make([]string, len(p.values))
	for i, v := range p.values {
		s[i] = base64.RawStdEncoding.EncodeToString(v)
	}
	return strings.Join(s, "\r\n")
}

// Conflict returns conflict policy as a string. Used with HTML templates.
func (p *pair) Conflict() string {
	switch p.conflict {
	case conflictInvalid:
		return "invalid"
	case conflictNewest:
		return "newest"
	case conflictOldest:
		return "oldest"
	case conflictAdd:
		return "add"
	}
	return ""
}

func (p *pair) setFromBuffer(b *bytes.Buffer) error {
	var err error
	p.key, err = readString(b)
	if err != nil {
		return err
	}
	p.conflict, err = readByte(b)
	if err != nil {
		return err
	}
	p.created, err = readTime(b)
	if err != nil {
		return err
	}
	p.expires, err = readDate(b)
	if err != nil {
		return err
	}
	p.values, err = readByteArrayArray(b)
	if err != nil {
		return err
	}
	return nil
}

func (p *pair) writeToBuffer(b *bytes.Buffer) error {
	err := writeString(b, p.key)
	if err != nil {
		return err
	}
	err = writeByte(b, p.conflict)
	if err != nil {
		return err
	}
	err = writeTime(b, p.created)
	if err != nil {
		return err
	}
	err = writeDate(b, p.expires)
	if err != nil {
		return err
	}
	err = writeByteArrayArray(b, p.values)
	if err != nil {
		return err
	}
	return nil
}

func (p *pair) present() bool {
	return p.created.IsZero() == false
}

func (p *pair) isValid() bool {
	return p.expires.After(time.Now().UTC())
}

// isEmpty treats any pair without any values as empty. A pair with values, but
// those values are empty byte array is not considered any empty value.
func (p *pair) isEmpty() bool {
	return p.values == nil || len(p.values) == 0
}

// equals returns true if the key and all values match exactly, otherwise false.
func (p *pair) equals(o *pair) bool {

	// Check that the keys are equal.
	if p.key != o.key {
		return false
	}

	// Check that the number of values are the same.
	if len(p.values) != len(o.values) {
		return false
	}

	// Check that the values are all identical.
	for i := 0; i < len(p.values); i++ {
		if bytes.Equal(p.values[i], o.values[i]) == false {
			return false
		}
	}
	return true
}

// Performs a distinct merge of the values in the two pairs. Duplicates are
// removed.
func mergeValues(o *pair, c *pair) [][]byte {

	// Make an array of values that has sufficient capacity to support all
	// the values.
	v := make([][]byte, 0, len(o.values)+len(c.values))

	// Add the values from pair o. Assumes that o does not contain any
	// duplicates.
	for _, a := range o.values {
		v = append(v, a)
	}

	// Add any values from pair c that are not in v.
	for _, a := range c.values {
		f := false
		for _, b := range v {
			if bytes.Equal(a, b) {
				f = true
				break
			}
		}
		if f == false {
			v = append(v, a)
		}
	}
	return v
}

func mergePairs(o *pair, c *pair) *pair {
	if valuesEqual(o.values, c.values) == false {
		var n pair
		n.conflict = conflictAdd
		n.created = time.Now().UTC()
		if o.expires.After(c.expires) {
			n.expires = o.expires
		} else {
			n.expires = c.expires
		}
		n.key = o.key
		n.values = mergeValues(o, c)
		return &n
	}
	return c
}

func valuesEqual(a [][]byte, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if bytes.Equal(a[i], b[i]) == false {
			return false
		}
	}
	return true
}

func resolveConflictOldest(o *pair, c *pair) *pair {
	if o.created.Before(c.created) {
		return o
	}
	if c.created.Before(o.created) {
		return c
	}
	if o.cookieWriteTime.After(c.cookieWriteTime) {
		return o
	}
	return c
}

func resolveConflictNewest(o *pair, c *pair) *pair {
	if o.created.After(c.created) {
		return o
	}
	if c.created.After(o.created) {
		return c
	}
	if o.cookieWriteTime.After(c.cookieWriteTime) {
		return o
	}
	return c
}

// Where there are two pairs for the same key determine which one should be used
// for the next operation in the storage operation.
// o is the pair from the storage operation
// c is the pair stored in a cookie for the current node
func resolveConflict(o *pair, c *pair) (*pair, error) {
	var p *pair
	if o == nil && c == nil {
		// Neither has any information.
		p = &emptyValue
	} else if o != nil && c == nil {
		// o is the only valid pair.
		p = o
	} else if o == nil && c != nil {
		// c is the only valid pair.
		p = c
	} else {
		// Resolve any conflict using o's conflict flag.
		switch o.conflict {
		case conflictInvalid:
			return nil, fmt.Errorf("Conflict flag is not initialized")
		case conflictNewest:
			p = resolveConflictNewest(o, c)
			break
		case conflictOldest:
			p = resolveConflictOldest(o, c)
			break
		case conflictAdd:
			p = mergePairs(o, c)
			break
		default:
			p = o
			break
		}
	}
	return p, nil
}
