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
	network   string    // The name of the network the node belongs to
	domain    string    // The domain name associated with the node
	hash      uint64    // Number used to relate client IPs to node
	created   time.Time // The time that the node first came online
	starts    time.Time // The time that the node will begin operation
	expires   time.Time // The time that the node will retire from the network
	role      int       // The role the node has in the network
	secrets   []*secret // All the secrets associated with the node
	scrambler *secret   // Secret used to scramble data with fixed nonce
	nonce     []byte    // Fixed nonce used with the scrambler
	accessed  time.Time // The time the node was last accessed
	alive     bool      // True if the node is reachable via a HTTP request
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

func newNode(
	network string,
	domain string,
	created time.Time,
	starts time.Time,
	expires time.Time,
	role int,
	scrambleKey string) (*node, error) {
	s, err := newSecretFromKey(scrambleKey, created)
	if err != nil {
		return nil, err
	}
	n := node{
		network,
		domain,
		getHash(domain),
		created,
		starts,
		expires,
		role,
		make([]*secret, 0),
		s,
		makeNonce(s, []byte(domain)),
		time.Time{},
		false}
	return &n, nil
}

func makeNonce(s *secret, d []byte) []byte {
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

func (n *node) isActive() bool {
	return n.expires.After(time.Now().UTC()) && len(n.secrets) > 0
}

func (n *node) unscramble(s string) (string, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	d, err := n.scrambler.crypto.decrypt(b)
	if err != nil {
		return "", err
	}
	return string(d), err
}

func (n *node) scrambleByteArray(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(
		n.scrambler.crypto.encryptWithNonce(b, n.nonce))
}

func (n *node) scramble(s string) string {
	return n.scrambleByteArray([]byte(s))
}

func (n *node) encrypt(d []byte) ([]byte, error) {
	s, err := n.getSecret()
	if err != nil {
		return nil, err
	}
	return s.crypto.compressAndEncrypt(d)
}

// Decrypt takes the byte array and decrypts the results ready to be used by the
// swift.DecodeResults method.
// d encrypted byte array
func (n *node) Decrypt(d []byte) ([]byte, error) {
	var err error
	for _, s := range n.secrets {
		b, err := s.crypto.decryptAndDecompress(d)
		if err == nil {
			return b, nil
		}
	}
	return nil, err
}

// DecryptAndDecode takes the byte array, decrypts it and decodes it into a Results
// structure checking that the time stamp is valid.
func (n *node) DecryptAndDecode(d []byte) (*Results, error) {

	// Decrypt the byte array using the node.
	b, err := n.Decrypt(d)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, fmt.Errorf("Could not decrypt byte array")
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
		"network":   n.network,
		"domain":    n.domain,
		"created":   n.created,
		"starts":    n.starts,
		"expires":   n.expires,
		"role":      n.role,
		"secrets":   n.secrets,
		"scrambler": n.scrambler.key,
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
	d, err := n.Decrypt(v)
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
	if n == nil {
		fmt.Println("Null node")
	}
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
