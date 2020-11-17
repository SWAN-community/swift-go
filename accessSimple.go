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

// AccessSimple is a implementation of swift.Access for testing where a list
// of keys returns true, and all others return false.
type AccessSimple struct {
	validKeys []string // A list of the keys that are valid.
}

func NewAccessSimple(validKeys []string) *AccessSimple {
	var a AccessSimple
	a.validKeys = validKeys
	return &a
}

func (a *AccessSimple) GetAllowed(accessKey string) (bool, error) {
	// TODO: Change the method to use the list of a hash set.
	return true, nil
}
