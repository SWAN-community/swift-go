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
	"time"
)

const (
	conflictOldest = iota
	conflictNewest = iota
)

// An empty pair referenced in the resolveConflict method if both parameters are
// null.
var emptyValue pair

type pair struct {
	key             string    // The name of the key associated with the value
	created         time.Time // The UTC time that the value was created
	expires         time.Time // The UTC time that the value will expire
	value           string    // The value as a string
	conflict        byte      // Flag for conflict resolution
	cookieWriteTime time.Time // Last time the cookie was written to
}

// Key returns the key as a string. Used with HTML templates.
func (p *pair) Key() string { return p.key }

// Value returns the value as string. Used with HTML templates.
func (p *pair) Value() string { return p.value }

// Created returns the date and pair was created. Used with HTML templates.
func (p *pair) Created() time.Time { return p.created }

// Expires returns the date and pair will expire. Used with HTML templates.
func (p *pair) Expires() time.Time { return p.expires }

// Conflict returns conflict policy as a string. Used with HTML templates.
func (p *pair) Conflict() string {
	switch p.conflict {
	case conflictNewest:
		return "newest"
	case conflictOldest:
		return "oldest"
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
	p.value, err = readString(b)
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
	err = writeString(b, p.value)
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
func resolveConflict(o *pair, c *pair) *pair {
	var p *pair
	if o == nil && c == nil {
		// Neither has any information.
		p = &emptyValue
	} else if o != nil && c == nil {
		// A is the only valid pair.
		p = o
	} else if o == nil && c != nil {
		// B is the only valid pair.
		p = c
	} else {
		// Resolve any conflict using A's conflict flag.
		switch o.conflict {
		case conflictNewest:
			p = resolveConflictNewest(o, c)
		case conflictOldest:
			p = resolveConflictOldest(o, c)
		default:
			p = o
		}
	}
	return p
}
