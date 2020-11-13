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
	"bytes"
	"errors"
	"time"
)

type result struct {
	Key     string    // The name of the key associated with the value
	Created time.Time // The UTC time that the value was created
	Expires time.Time // The UTC time that the value will expire
	Value   string    // The value as a byte array
}

type results struct {
	expires time.Time // The time after which the data can not be decrypted
	values  []*result // Array of values
}

func (r *results) isTimeStampValid() bool {
	return time.Now().UTC().Before(r.expires)
}

func encodeResults(r *results) ([]byte, error) {
	var b bytes.Buffer
	var err error
	err = writeTime(&b, r.expires)
	if err != nil {
		return nil, err
	}
	err = writeByte(&b, byte(len(r.values)))
	if err != nil {
		return nil, err
	}
	for _, e := range r.values {
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

func decodeResults(d []byte) (*results, error) {
	var err error
	var r results
	if d == nil {
		return nil, errors.New("Byte array empty")
	}
	b := bytes.NewBuffer(d)
	r.expires, err = readTime(b)
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
		r.values = append(r.values, &result{k, c, e, v})
	}
	return &r, nil
}
