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
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Configuration maps to the appsettings.json settings file.
type Configuration struct {
	BundleTimeout   time.Duration `json:"bundleTimeout"`
	CookieTimeout   time.Duration `json:"cookieTimeout"`
	Message         string        `json:"message"`
	Title           string        `json:"title"`
	BackgroundColor string        `json:"backgroundColor"`
	MessageColor    string        `json:"messageColor"`
	ProgressColor   string        `json:"progressColor"`
	Scheme          string        `json:"scheme"`
	NodeCount       byte          `json:"nodeCount"`
	AzureAccessKey  string        `json:"azureAccessKey"`
	AzureAccount    string        `json:"azureAccount"`
	AWSAccessKey    string        `json:"awsAccessKey"`
	AWSAccount      string        `json:"awsAccount"`
	Debug           bool          `json:"debug"`
}

// NewConfig creates a new instance of configuration from the file provided.
func NewConfig(file string) Configuration {
	var c Configuration
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&c)
	return c
}

// Validate confirms that the configuration is usable.
func (c *Configuration) Validate() error {
	var err error
	fmt.Printf("Debug Mode: %t\n", c.Debug)
	if err == nil {
		if c.Message != "" {
			fmt.Printf("Message: %s\n", c.Message)
		} else {
			err = fmt.Errorf("Message missing in config")
		}
	}
	if err == nil {
		if c.Title != "" {
			fmt.Printf("Title: %s\n", c.Title)
		} else {
			err = fmt.Errorf("Title missing in config")
		}
	}
	if err == nil {
		if c.BackgroundColor != "" {
			fmt.Printf("BackgroundColor: %s\n", c.BackgroundColor)
		} else {
			err = fmt.Errorf("BackgroundColor missing in config")
		}
	}
	if err == nil {
		if c.MessageColor != "" {
			fmt.Printf("MessageColor: %s\n", c.MessageColor)
		} else {
			err = fmt.Errorf("MessageColor missing in config")
		}
	}
	if err == nil {
		if c.ProgressColor != "" {
			fmt.Printf("ProgressColor: %s\n", c.ProgressColor)
		} else {
			err = fmt.Errorf("ProgressColor missing in config")
		}
	}
	return err
}
