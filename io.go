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
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// The base year for all dates encoded with the io time methods.
var ioDateBase = time.Date(2020, time.Month(1), 1, 0, 0, 0, 0, time.UTC)

func readString(b *bytes.Buffer) (string, error) {
	s, err := b.ReadBytes(0)
	if err == nil {
		return string(s[0 : len(s)-1]), err
	}
	return "", err
}

func readByteArrayArray(b *bytes.Buffer) ([][]byte, error) {
	c, err := readUint16(b)
	if err != nil {
		return nil, err
	}
	v := make([][]byte, c, c)
	for i := uint16(0); i < c; i++ {
		v[i], err = readByteArray(b)
		if err != nil {
			return nil, err
		}
	}
	return v, nil
}

func readByteArray(b *bytes.Buffer) ([]byte, error) {
	l, err := readUint16(b)
	if err != nil {
		return nil, err
	}
	return b.Next(int(l)), err
}

func writeByteArrayArray(b *bytes.Buffer, v [][]byte) error {
	err := writeUint16(b, uint16(len(v)))
	if err != nil {
		return err
	}
	for _, i := range v {
		err = writeByteArray(b, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeByteArray(b *bytes.Buffer, v []byte) error {
	err := writeUint16(b, uint16(len(v)))
	if err != nil {
		return err
	}
	l, err := b.Write(v)
	if err == nil {
		if l != len(v) {
			return fmt.Errorf(
				"Mismatched lengths '%d' and '%d'",
				l,
				len(v))
		}
	}
	return err
}

func readTime(b *bytes.Buffer) (time.Time, error) {
	var t time.Time
	d, err := readByteArray(b)
	if err == nil {
		t.GobDecode(d)
	}
	return t, err
}

func writeTime(b *bytes.Buffer, t time.Time) error {
	d, err := t.GobEncode()
	if err != nil {
		return err
	}
	return writeByteArray(b, d)
}

func readDate(b *bytes.Buffer) (time.Time, error) {
	h, err := b.ReadByte()
	if err != nil {
		return time.Time{}, err
	}
	l, err := b.ReadByte()
	if err != nil {
		return time.Time{}, err
	}
	d := int(h)<<8 | int(l)
	return ioDateBase.Add(time.Duration(d) * time.Hour * 24), nil
}

func writeDate(b *bytes.Buffer, t time.Time) error {
	i := int(t.Sub(ioDateBase).Hours() / 24)
	err := writeByte(b, byte(i>>8))
	if err != nil {
		return err
	}
	return writeByte(b, byte(i&0x00FF))
}

func readByte(b *bytes.Buffer) (byte, error) {
	d := b.Next(1)
	if len(d) != 1 {
		return 0, fmt.Errorf("'%d' bytes incorrect for Byte", len(d))
	}
	return d[0], nil
}

func writeByte(b *bytes.Buffer, i byte) error {
	return b.WriteByte(i)
}

func readUint64(b *bytes.Buffer) (uint64, error) {
	d := b.Next(8)
	if len(d) != 8 {
		return 0, fmt.Errorf("'%d' bytes incorrect for Uint64", len(d))
	}
	return binary.LittleEndian.Uint64(d), nil
}

func writeUint64(b *bytes.Buffer, i uint64) error {
	v := make([]byte, 8)
	binary.LittleEndian.PutUint64(v, i)
	l, err := b.Write(v)
	if err == nil {
		if l != len(v) {
			return fmt.Errorf(
				"Mismatched lengths '%d' and '%d'",
				l,
				len(v))
		}
	}
	return err
}

func readUint16(b *bytes.Buffer) (uint16, error) {
	d := b.Next(2)
	if len(d) != 2 {
		return 0, fmt.Errorf("'%d' bytes incorrect for Uint16", len(d))
	}
	return binary.LittleEndian.Uint16(d), nil
}

func writeUint16(b *bytes.Buffer, i uint16) error {
	v := make([]byte, 2)
	binary.LittleEndian.PutUint16(v, i)
	l, err := b.Write(v)
	if err == nil {
		if l != len(v) {
			return fmt.Errorf(
				"Mismatched lengths '%d' and '%d'",
				l,
				len(v))
		}
	}
	return err
}

func readUint32(b *bytes.Buffer) (uint32, error) {
	d := b.Next(4)
	if len(d) != 4 {
		return 0, fmt.Errorf("'%d' bytes incorrect for Uint32", len(d))
	}
	return binary.LittleEndian.Uint32(d), nil
}

func writeUint32(b *bytes.Buffer, i uint32) error {
	v := make([]byte, 4)
	binary.LittleEndian.PutUint32(v, i)
	l, err := b.Write(v)
	if err == nil {
		if l != len(v) {
			return fmt.Errorf(
				"Mismatched lengths '%d' and '%d'",
				l,
				len(v))
		}
	}
	return err
}

func writeString(b *bytes.Buffer, s string) error {
	l, err := b.WriteString(s)
	if err == nil {

		// Validate the number of bytes written matches the number of bytes in
		// the string.
		if l != len(s) {
			return fmt.Errorf(
				"Mismatched lengths '%d' and '%d'",
				l,
				len(s))
		}

		// Write the null terminator.
		b.WriteByte(0)
	}
	return err
}

func writeFloat32(b *bytes.Buffer, f float32) error {
	return writeUint32(b, math.Float32bits(f))
}

func readFloat32(b *bytes.Buffer) (float32, error) {
	f, err := readUint32(b)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(f), nil
}
