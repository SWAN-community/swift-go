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
	"net/http"
	"time"
)

// Register contains HTML template data used to register a node with the network
type Register struct {
	Services      *Services
	Store         string
	Domain        string
	Network       string
	Starts        time.Time
	Expires       time.Time
	Role          int
	Error         string
	NetworkError  string
	ExpiresError  string
	StoreError    string
	RoleError     string
	ReadOnly      bool
	DisplayErrors bool
	request       *http.Request
}

// ExpiresString returns the expires date as a string
func (r *Register) ExpiresString() string {
	return r.Expires.Format("2006-01-02")
}

// StartsString returns the start date as a string
func (r *Register) StartsString() string {
	return r.Starts.Format("2006-01-02")
}

// Language returns the language code associated with the web page.
func (r *Register) Language() string {
	v := r.request.Header.Get("accept-language")
	if v != "" {
		return languageRegex.FindString(v)
	}
	return ""
}

// BackgroundColor returns the background color associated with the service config
func (r *Register) BackgroundColor() string {
	return r.Services.config.BackgroundColor
}

// MessageColor returns the message color associated with the service config
func (r *Register) MessageColor() string {
	return r.Services.config.MessageColor
}
