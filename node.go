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
	"encoding/json"
	"fmt"
	"hash/crc64"
	"hash/fnv"
	"math/rand"
	"net/http"
	"sort"
	"time"
)

// Table used to initialize hash functions.
var nodeHashTable = crc64.MakeTable(crc64.ECMA)

const (
	roleAccess  = iota // The node responds to server initiated access requests
	roleStorage = iota // The node can be used for storage operations
	roleShare   = iota // The node responds to share requests
)

// node is a SWIFT storage node associated with a network and a domain name.
type node struct {
	network      string    // The name of the network the node belongs to
	domain       string    // The domain name associated with the node
	hash         uint64    // Number used to relate client IPs to node
	created      time.Time // The time that the node first came online
	starts       time.Time // The time that the node will begin operation
	expires      time.Time // The time that the node will retire from the network
	role         int       // The role the node has in the network
	secrets      []*secret // All the secrets associated with the node
	scrambler    *secret   // Secret used to scramble data with fixed nonce
	nonce        []byte    // Fixed nonce used with the scrambler
	accessed     time.Time // The time the node was last accessed
	alive        bool      // True if the node is reachable via a HTTP request
	cookieDomain string    // The domain to use for cookies
}

// Domain returns the internet domain associated with the Node.
func (n *node) Domain() string { return n.domain }

// Network returns the network names associated with the Node.
func (n *node) Network() string { return n.network }

func getHash(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return rand.New(rand.NewSource(int64(h.Sum64()))).Uint64()
}

func (n *node) getScramblerKey() string {
	if n.scrambler != nil {
		return n.scrambler.key
	}
	return ""
}

// supportsCrypto returns true if the node can encrypt and decrypt data.
func (n *node) supportsCrypto() bool { return len(n.secrets) > 0 }

func newNode(
	network string,
	domain string,
	created time.Time,
	starts time.Time,
	expires time.Time,
	role int,
	scrambleKey string,
	cookieDomain string) (*node, error) {
	scrambler, err := makeScrambler(created, scrambleKey)
	if err != nil {
		return nil, err
	}
	n := node{
		network:      network,
		domain:       domain,
		hash:         getHash(domain),
		created:      created,
		starts:       starts,
		expires:      expires,
		role:         role,
		secrets:      make([]*secret, 0),
		scrambler:    scrambler,
		nonce:        makeNonce(scrambler, []byte(domain)),
		accessed:     time.Time{},
		alive:        false,
		cookieDomain: cookieDomain}
	return &n, nil
}

// makeScrambler If a scramble key is provided then make the scrambler,
// otherwise return nil to indicate the node will not scramble the table name
// to form the first fragment of the storage path.
func makeScrambler(created time.Time, scrambleKey string) (*secret, error) {
	if scrambleKey != "" {
		s, err := newSecretFromKey(scrambleKey, created)
		if err != nil {
			return nil, err
		}
		return s, nil
	}
	return nil, nil
}

// makeNonce If a secret is provided then returns nonce for the secret,
// otherwise an empty array.
func makeNonce(s *secret, d []byte) []byte {
	if s != nil {
		n := make([]byte, s.crypto.gcm.NonceSize())
		c := 0
		for i := 0; i < len(n); i++ {
			n[i] = d[c]
			c++
			if c >= len(d) {
				c = 0
			}
		}
		return n
	}
	return []byte{}
}

// isActive returns true if the node has not expired.
func (n *node) isActive() bool {
	return n.expires.After(time.Now().UTC())
}

// unscramble if the node has been configured with a scrambler then the input
// string should be a base 64 encoded string created by the scramble method
// previously. If no scrambler is used with the node then the input is the same
// as the output.
func (n *node) unscramble(s string) (string, error) {
	if n.scrambler != nil {
		b, err := base64.RawURLEncoding.DecodeString(s)
		if err != nil {
			return "", err
		}
		d, err := n.scrambler.crypto.decrypt(b)
		if err != nil {
			return "", err
		}
		return string(d), nil
	}
	return s, nil
}

// scramble the input string if there is a scrambler used with the node. If no
// scrambler is used with the node then the input is the same as the output.
func (n *node) scramble(s string) string {
	if n.scrambler != nil {
		return base64.RawURLEncoding.EncodeToString(
			n.scrambler.crypto.encryptWithNonce([]byte(s), n.nonce))
	}
	return s
}

// encrypt the byte array with the most recent secret that the now has. Returns
// an error if no secrets are available or the encryption fails.
func (n *node) encrypt(d []byte) ([]byte, error) {
	s, err := n.getSecret()
	if err != nil {
		return nil, err
	}
	return s.crypto.encrypt(d)
}

// decrypt the byte array b using the secrets available to the node returning
// the decrypted byte array.
//
// b encrypted byte array
func (n *node) decrypt(b []byte) ([]byte, error) {
	for _, s := range n.secrets {
		d, err := s.crypto.decrypt(b)
		if err != nil {
			return nil, err
		}
		if d != nil {
			return d, nil
		}
	}
	return nil, fmt.Errorf("no secrets available to decrypt byte array")
}

// encode takes the byte array, compresses it and if there are secrets for the
// node encrypts it ready to be used with HTTP responses.
//
// b byte array to encode
func (n *node) encode(b []byte) ([]byte, error) {
	e, err := compress(b)
	if err != nil {
		return nil, err
	}
	if n.supportsCrypto() {
		e, err = n.encrypt(e)
		if err != nil {
			return nil, err
		}
	}
	return e, nil
}

// decode decrypts the byte array b if the node supports crypto and then
// decompresses the result before returning it.
//
// b byte array to be decoded.
func (n *node) decode(b []byte) ([]byte, error) {
	var err error
	if n.supportsCrypto() {
		b, err = n.decrypt(b)
		if err != nil {
			return nil, err
		}
	}
	return decompress(b)
}

// DecodeAsResults takes the byte array, decodes it into a Results structure
// checking that the time stamp is valid.
func (n *node) DecodeAsResults(d []byte) (*Results, error) {

	// Decrypt the byte array using the node.
	b, err := n.decode(d)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("could not decrypt byte array")
	}

	// Decode the byte array to become a results array.
	r, err := DecodeResults(b)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// MarshalJSON marshals a node to JSON without having to expose the fields in
// the node struct. This is achieved by converting a node to a map.
func (n *node) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"network":      n.network,
		"domain":       n.domain,
		"created":      n.created,
		"starts":       n.starts,
		"expires":      n.expires,
		"role":         n.role,
		"secrets":      n.secrets,
		"scrambler":    n.getScramblerKey(),
		"cookieDomain": n.cookieDomain,
	})
}

// UnmarshalJSON called by json.Unmarshall unmarshals a node from JSON and turns
// it into a new node. As the node is marshalled to JSON by converting it to a
// map, the unmarshalling from JSON needs to handle the type of each field
// correctly.
func (n *node) UnmarshalJSON(b []byte) error {
	var d map[string]interface{}
	err := json.Unmarshal(b, &d)
	if err != nil {
		return err
	}

	created, err := time.Parse(time.RFC3339Nano, d["created"].(string))
	if err != nil {
		return err
	}

	starts, err := time.Parse(time.RFC3339Nano, d["starts"].(string))
	if err != nil {
		return err
	}

	expires, err := time.Parse(time.RFC3339Nano, d["expires"].(string))
	if err != nil {
		return err
	}

	role := int(d["role"].(float64))

	np, err := newNode(
		d["network"].(string),
		d["domain"].(string),
		created,
		starts,
		expires,
		role,
		d["scrambler"].(string),
		d["cookieDomain"].(string),
	)
	secrets := d["secrets"].([]interface{})

	for _, secret := range secrets {
		s := secret.(map[string]interface{})

		k := s["key"].(string)

		t, err := time.Parse(time.RFC3339Nano, s["timeStamp"].(string))
		if err != nil {
			return err
		}

		sec, err := newSecretFromKey(k, t)
		if err != nil {
			return err
		}

		np.secrets = append(n.secrets, sec)
	}

	*n = *np
	if err != nil {
		return err
	}
	return nil
}

func (n *node) getValueFromCookie(c *http.Cookie) (*pair, error) {
	var p pair
	v, err := base64.StdEncoding.DecodeString(c.Value)
	if err != nil {
		return nil, err
	}
	d, err := n.decode(v)
	if err != nil {
		return nil, err
	}
	if len(d) == 0 {
		return nil, fmt.Errorf("value for cookie '%s' zero length", c.Name)
	}
	b := bytes.NewBuffer(d)
	p.cookieWriteTime, err = readTime(b)
	if err != nil {
		return nil, fmt.Errorf("time for cookie '%s' invalid", c.Name)
	}
	err = p.setFromBuffer(b)
	if err != nil {
		return nil, fmt.Errorf(
			"Value for cookie '%s' error '%s'",
			c.Name,
			err.Error())
	}
	return &p, nil
}

func (n *node) addSecret(secret *secret) {
	n.secrets = append(n.secrets, secret)
}

func (n *node) getSecret() (*secret, error) {
	if len(n.secrets) > 0 {
		return n.secrets[0], nil
	}
	return nil, fmt.Errorf("no secrets for node '%s'", n.domain)
}

func (n *node) sortSecrets() {
	sort.Slice(n.secrets, func(i, j int) bool {
		return n.secrets[i].timeStamp.Sub(n.secrets[j].timeStamp) < 0
	})
}
