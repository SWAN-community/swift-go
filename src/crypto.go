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
	"compress/zlib"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
)

// crypto structure containing AES ciphers.
type crypto struct {
	gcm cipher.AEAD
}

// NewCrypto creates a new instance of the security structure used to encrypt
// and decrypt data using rotating shared secret keys.
func newCrypto(key []byte) (*crypto, error) {
	var x crypto
	i, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	x.gcm, err = cipher.NewGCM(i)
	if err != nil {
		return nil, err
	}
	return &x, nil
}

func (x *crypto) decrypt(b []byte) ([]byte, error) {
	nonceSize := x.gcm.NonceSize()
	if len(b) < nonceSize {
		return nil, fmt.Errorf(
			"Data length '%d' shorter than nonce '%d'",
			len(b),
			nonceSize)
	}
	nonce, c := b[:nonceSize], b[nonceSize:]
	d, err := x.gcm.Open(nil, nonce, c, nil)
	if err != nil {
		return nil, err
	}
	return d, err
}

func (x *crypto) decryptAndDecompress(b []byte) ([]byte, error) {
	d, err := x.decrypt(b)
	if err != nil {
		return nil, err
	}
	return decompress(d)
}

func (x *crypto) encryptWithNonce(b []byte, n []byte) []byte {

	// Seal encrypts and authenticates plaintext, authenticates the
	// additional data and appends the result to dst, returning the updated
	// slice. The nonce must be NonceSize() bytes long and unique for all
	// time, for a given key.
	return x.gcm.Seal(n, n, b, nil)
}

func (x *crypto) compressAndEncrypt(b []byte) ([]byte, error) {

	// Compress the data before encrypting it.
	c, err := compress(b)
	if err != nil {
		return nil, err
	}

	// Create nonce with a cryptographically secure random sequence. Nonce
	// should never be repeated.
	n, err := randomBytes(x.gcm.NonceSize())
	if err != nil {
		return nil, err
	}
	return x.encryptWithNonce(c, n), nil
}

func randomBytes(l int) ([]byte, error) {
	r := make([]byte, l)
	_, err := io.ReadFull(rand.Reader, r)
	return r, err
}

func compress(b []byte) ([]byte, error) {
	var o bytes.Buffer
	z := zlib.NewWriter(&o)
	i, err := z.Write(b)
	if err != nil {
		return nil, err
	}
	if i != len(b) {
		return nil, fmt.Errorf(
			"Byte written '%d' does not match length '%d",
			i,
			len(b))
	}
	z.Close()
	return o.Bytes(), nil
}

func decompress(b []byte) ([]byte, error) {
	f := bytes.NewReader(b)
	z, err := zlib.NewReader(f)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(z)
}
