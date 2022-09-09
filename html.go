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

	"github.com/SWAN-community/common-go"
)

// Constants for the bits in operation.flags where the constant name corresponds
// to the public method of operation.
const (
	flagDisplayUserInterface  = iota
	flagPostMessageOnComplete = iota
	flagUseHomeNode           = iota
	flagJavaScript            = iota
)

// HTML parameters that control the function and display of the user interface.
type HTML struct {
	Title           string // Window title
	Message         string // Message to display
	BackgroundColor string // Background color of the window
	MessageColor    string // Color of the message text
	ProgressColor   string // Color of the progress line
	flags           byte
}

// DisplayUserInterface true if a UI should be displayed during the storage
// operation, otherwise false.
func (h *HTML) DisplayUserInterface() bool {
	return h.hasBit(flagDisplayUserInterface)
}

// DisplayUserInterfaceAsString returns the flag as string either "true" or
// "false".
func (h *HTML) DisplayUserInterfaceAsString() string {
	if h.DisplayUserInterface() {
		return "true"
	}
	return "false"
}

// SetDisplayUserInterface sets the flag to true or false.
func (h *HTML) SetDisplayUserInterface(v bool) {
	if v {
		h.setBit(flagDisplayUserInterface)
	} else {
		h.clearBit(flagDisplayUserInterface)
	}
}

// PostMessageOnComplete true if at the end of the operation the resulting data
// should be returned to the parent using JavaScript postMessage, otherwise
// false.
// parent.postMessage("swan","[Encrypted SWAN data]");
func (h *HTML) PostMessageOnComplete() bool {
	return h.hasBit(flagPostMessageOnComplete)
}

// PostMessageOnCompleteAsString returns the flag as string either "true" or
// "false".
func (h *HTML) PostMessageOnCompleteAsString() string {
	if h.PostMessageOnComplete() {
		return "true"
	}
	return "false"
}

// SetPostMessageOnComplete sets the flag to true or false.
func (h *HTML) SetPostMessageOnComplete(v bool) {
	if v {
		h.setBit(flagPostMessageOnComplete)
	} else {
		h.clearBit(flagPostMessageOnComplete)
	}
}

// UseHomeNode true if the home node can be used if it contains current data.
// False if the SWAN network should be consulted irrespective of the state of
// data held on the home node.
func (h *HTML) UseHomeNode() bool {
	return h.hasBit(flagUseHomeNode)
}

// UseHomeNodeAsString returns the flag as a string. Either "true" or "false".
func (h *HTML) UseHomeNodeAsString() string {
	if h.PostMessageOnComplete() {
		return "true"
	}
	return "false"
}

// SetUseHomeNode sets the flag to true or false.
func (h *HTML) SetUseHomeNode(v bool) {
	if v {
		h.setBit(flagUseHomeNode)
	} else {
		h.clearBit(flagUseHomeNode)
	}
}

// JavaScript true if the response for storage operations should be JavaScript
// include that will continue the operation. This feature requires cookies to be
// sent for DOM inserted JavaScript elements.
func (h *HTML) JavaScript() bool {
	return h.hasBit(flagJavaScript)
}

// UseJavaScriptAsString returns the flag as a string. Either "true" or "false".
func (h *HTML) UseJavaScriptAsString() string {
	if h.JavaScript() {
		return "true"
	}
	return "false"
}

// SetJavaScript sets the flag to true or false.
func (h *HTML) SetJavaScript(v bool) {
	if v {
		h.setBit(flagJavaScript)
	} else {
		h.clearBit(flagJavaScript)
	}
}

func (h *HTML) setBit(pos uint8) byte {
	h.flags |= (1 << pos)
	return h.flags
}

func (h *HTML) clearBit(pos uint8) byte {
	h.flags &= ^(1 << pos)
	return h.flags
}

func (h *HTML) hasBit(pos uint8) bool {
	val := h.flags & (1 << pos)
	return (val > 0)
}

func (h *HTML) write(b *bytes.Buffer) error {
	var err error
	err = common.WriteString(b, h.Title)
	if err != nil {
		return err
	}
	err = common.WriteString(b, h.Message)
	if err != nil {
		return err
	}
	err = common.WriteString(b, h.BackgroundColor)
	if err != nil {
		return err
	}
	err = common.WriteString(b, h.MessageColor)
	if err != nil {
		return err
	}
	err = common.WriteString(b, h.ProgressColor)
	if err != nil {
		return err
	}
	err = common.WriteByte(b, h.flags)
	if err != nil {
		return err
	}
	return nil
}

func (h *HTML) set(b *bytes.Buffer) error {
	var err error
	h.Title, err = common.ReadString(b)
	if err != nil {
		return err
	}
	h.Message, err = common.ReadString(b)
	if err != nil {
		return err
	}
	h.BackgroundColor, err = common.ReadString(b)
	if err != nil {
		return err
	}
	h.MessageColor, err = common.ReadString(b)
	if err != nil {
		return err
	}
	h.ProgressColor, err = common.ReadString(b)
	if err != nil {
		return err
	}
	h.flags, err = common.ReadByte(b)
	if err != nil {
		return err
	}
	return nil
}
