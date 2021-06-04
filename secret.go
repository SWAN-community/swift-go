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

// MarshalJSON marshals a secret to JSON without having to expose the fields in
// the secret struct. This is achieved by converting a secret to a map.
func (s *secret) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"timeStamp": s.timeStamp,
		"key":       s.key,
	})
}

// UnmarshalJSON called by json.Unmarshall unmarshals a secret from JSON and
// turns it into a new secret. As the secret is marshalled to JSON by converting
// it to a map, the unmarshalling from JSON needs to handle the type of each
// field correctly.
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
