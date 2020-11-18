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

import "bytes"

// HTML parameters that control the display of the user interface.
type HTML struct {
	Title           string // Window title
	Message         string // Message to display
	BackgroundColor string // Background color of the window
	MessageColor    string // Color of the message text
	ProgressColor   string // Color of the progress line
}

func (h *HTML) write(b *bytes.Buffer) error {
	var err error
	err = writeString(b, h.Title)
	if err != nil {
		return err
	}
	err = writeString(b, h.Message)
	if err != nil {
		return err
	}
	err = writeString(b, h.BackgroundColor)
	if err != nil {
		return err
	}
	err = writeString(b, h.MessageColor)
	if err != nil {
		return err
	}
	err = writeString(b, h.ProgressColor)
	if err != nil {
		return err
	}
	return nil
}

func (h *HTML) set(b *bytes.Buffer) error {
	var err error
	h.Title, err = readString(b)
	if err != nil {
		return err
	}
	h.Message, err = readString(b)
	if err != nil {
		return err
	}
	h.BackgroundColor, err = readString(b)
	if err != nil {
		return err
	}
	h.MessageColor, err = readString(b)
	if err != nil {
		return err
	}
	h.ProgressColor, err = readString(b)
	if err != nil {
		return err
	}
	return nil
}
