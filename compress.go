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
	"compress/zlib"
	"fmt"
	"io"
)

// compress the byte array using the zlib compression routine.
func compress(b []byte) ([]byte, error) {
	var o bytes.Buffer
	z := zlib.NewWriter(&o)
	i, err := z.Write(b)
	if err != nil {
		return nil, err
	}
	z.Close()
	if i != len(b) {
		return nil, fmt.Errorf(
			"byte written '%d' does not match length '%d",
			i,
			len(b))
	}
	return o.Bytes(), nil
}

// decompress the byte array using the zlib compression routine.
func decompress(b []byte) ([]byte, error) {
	f := bytes.NewReader(b)
	z, err := zlib.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer z.Close()
	return io.ReadAll(z)
}
