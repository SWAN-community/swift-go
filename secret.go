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
	"encoding/base64"
	"encoding/json"
	"time"
)

type secret struct {
	timeStamp time.Time
	key       string
	crypto    *crypto
}

func newSecret() (*secret, error) {
	b, err := randomBytes(32)
	if err != nil {
		return nil, err
	}
	x, err := newCrypto(b)
	if err != nil {
		return nil, err
	}
	return &secret{time.Now(), base64.RawURLEncoding.EncodeToString(b), x}, nil
}

func newSecretFromKey(key string, timeStamp time.Time) (*secret, error) {
	b, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}
	x, err := newCrypto(b)
	if err != nil {
		return nil, err
	}
	return &secret{timeStamp, key, x}, nil
}

// TODO: use this to replace duplicate structs to mashal secrets to and from json
func (s *secret) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"timeStamp": s.timeStamp,
		"key":       s.key,
	})
}

// TODO: use this to replace duplicate structs to mashal secrets to and from json
func (s *secret) UnmarshalJSON(b []byte) error {
	var d map[string]interface{}
	err := json.Unmarshal(b, &d)
	if err != nil {
		return err
	}

	k := d["key"].(string)

	t, err := time.Parse(time.RFC3339Nano, d["timeStamp"].(string))
	if err != nil {
		return err
	}

	sp, err := newSecretFromKey(k, t)
	*s = *sp
	if err != nil {
		return err
	}
	return nil
}
