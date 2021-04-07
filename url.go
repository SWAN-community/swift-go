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
	"fmt"
	"net/url"
)

// SetURL take the value of key s, validates the value is a URL, and the sets
// the value of key d to the validated value. If the value is not a URL then
// an error is returned.
func SetURL(sourceKey string, destKey string, values *url.Values) error {
	u, err := validateURL(sourceKey, values.Get(sourceKey))
	if err != nil {
		return err
	}
	values.Set(destKey, u.String())
	return nil
}

// ValidateURL confirms that the parameter is a valid URL and then returns the
// URL ready for use with SWIFT if valid. The method checks that the SWIFT
// encrypted data can be appended to the end of the string as an identifiable
// segment if there is now query string in the URL. An error is returned if the
// URL is not validate for use with SWIFT.
func ValidateURL(name string, value string) (*url.URL, error) {
	if value == "" {
		return nil, fmt.Errorf("%s must be a valid URL", name)
	}
	u, err := url.Parse(value)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf(
			"%s '%s' must use http or https scheme", name, value)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("%s '%s' must include a host", name, value)
	}
	return u, nil
}
