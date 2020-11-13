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
	"net/http"
	"regexp"
)

type browserRegex struct {
	regex *regexp.Regexp
	html  string
}

// BrowserRegexes is a concrete implementation of the interface
// sws.BrowserDetector
type BrowserRegexes struct {
	browsers []*browserRegex
}

// NewBrowserRegexes creates a new implementation of sws.BrowserDetector
// configured with default regular expressions and messages.
func NewBrowserRegexes() (*BrowserRegexes, error) {
	var b BrowserRegexes
	safari, err := createRegexHTML("Safari", "<p>Apple, the company that control this web browsers, have adopted a policy that hurts our ability to pay for the services we provide to you for free. Find our more <a href=''>here</a>.</p>")
	if err != nil {
		return nil, err
	}
	b.browsers = append(b.browsers, safari)
	return &b, nil
}

// GetWarningHTML returns the warning text for the browser if the User-Agent
// HTTP matches a regular expression.
func (b *BrowserRegexes) GetWarningHTML(r *http.Request) string {
	for _, e := range b.browsers {
		if e.regex.MatchString(r.UserAgent()) {
			return e.html
		}
	}
	return ""
}

func createRegexHTML(r string, h string) (*browserRegex, error) {
	var err error
	var b browserRegex
	b.regex, err = regexp.Compile("p([a-z]+)ch")
	if err != nil {
		return nil, err
	}
	b.html = h
	return &b, nil
}
