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
	"log"
	"time"

	"github.com/SWAN-community/config-go"
)

// Configuration maps to the appsettings.json settings file.
type Configuration struct {
	config.Base `mapstructure:",squash"`
	// If a local file with SWIFT node information is to be used the path to the
	// file.
	SwiftFile string `mapstructure:"swiftFile"`
	// The number of seconds between polling operations for alive checks. This
	// is supplement to the passive check so if a node has not been accessed for
	// more than this then it is eligible for polling.
	AlivePollingSeconds int `mapstructure:"alivePollingSeconds"`
	// The number of seconds from creation of an operation that it is valid for.
	// Used to prevent repeated processing of the same operation.
	StorageOperationTimeout int `mapstructure:"storageOperationTimeout"`
	// The number of minutes between refreshes of the storage manager.
	StorageManagerRefreshMinutes int `mapstructure:"storageManagerRefreshMinutes"`
	// The maximum number of Store instances that can be referenced by a storage
	// manager.
	MaxStores int `mapstructure:"maxStores"`
	// The length of time in seconds values stored in SWIFT nodes can be relied
	// upon to be current. Used by the home node to determine if it should
	// consult other nodes in the network before returning it's current values.
	HomeNodeTimeout int `mapstructure:"homeNodeTimeout"`
	// The default message to display in the user interface if one is not
	// provided by the requestor of the storage operation.
	Message string `mapstructure:"message"`
	// The title of the web page to use in the user interface if one is not
	// provided by the requestor of the storage operation.
	Title string `mapstructure:"title"`
	// The background color of the web page to use in the user interface if one
	// is not provided by the requestor of the storage operation.
	BackgroundColor string `mapstructure:"backgroundColor"`
	// The message color to use in the user interface if one is not provided by
	// the requestor of the storage operation.
	MessageColor string `mapstructure:"messageColor"`
	// The progress circle color to use in the user interface if one is not
	// provided by the requestor of the storage operation.
	ProgressColor string `mapstructure:"progressColor"`
	// The HTTP scheme to use (HTTP for development and HTTPS for production).
	Scheme string `mapstructure:"scheme"`
	// The number of nodes to consult when accessing the SWIFT network.
	NodeCount byte `mapstructure:"nodeCount"`
	// True to enable debug logging and user interfaces.
	Debug bool `mapstructure:"debug"`
}

// HomeNodeTimeoutDuration the home node timeout as a time.Duration
func (c *Configuration) HomeNodeTimeoutDuration() time.Duration {
	return time.Duration(c.HomeNodeTimeout) * time.Second
}

// StorageOperationTimeoutDuration the storage operation timeout as a
// time.Duration
func (c *Configuration) StorageOperationTimeoutDuration() time.Duration {
	return time.Duration(c.StorageOperationTimeout) * time.Second
}

// NewConfig creates a new instance of configuration from the file provided.
func NewConfig(file string) Configuration {
	var c Configuration
	err := config.LoadConfig([]string{"."}, file, &c)
	if err != nil {
		fmt.Println(err.Error())
	}
	return c
}

// Validate confirms that the configuration is usable.
func (c *Configuration) Validate() error {
	var err error
	log.Printf("SWIFT:Debug Mode: %t\n", c.Debug)
	if err == nil {
		if c.Message != "" {
			log.Printf("SWIFT:Message: %s\n", c.Message)
		} else {
			err = fmt.Errorf("SWIFT Message missing in config")
		}
	}
	if err == nil {
		if c.Title != "" {
			log.Printf("SWIFT:Title: %s\n", c.Title)
		} else {
			err = fmt.Errorf("SWIFT Title missing in config")
		}
	}
	if err == nil {
		if c.BackgroundColor != "" {
			log.Printf("SWIFT:BackgroundColor: %s\n", c.BackgroundColor)
		} else {
			err = fmt.Errorf("SWIFT BackgroundColor missing in config")
		}
	}
	if err == nil {
		if c.MessageColor != "" {
			log.Printf("SWIFT:MessageColor: %s\n", c.MessageColor)
		} else {
			err = fmt.Errorf("SWIFT MessageColor missing in config")
		}
	}
	if err == nil {
		if c.ProgressColor != "" {
			log.Printf("SWIFT:ProgressColor: %s\n", c.ProgressColor)
		} else {
			err = fmt.Errorf("SWIFT ProgressColor missing in config")
		}
	}
	if err == nil {
		if c.Scheme != "" {
			log.Printf("SWIFT:Scheme: %s\n", c.Scheme)
		} else {
			err = fmt.Errorf("SWIFT Scheme missing in config")
		}
	}
	if err == nil {
		if !(c.Scheme == "http" || c.Scheme == "https") {
			err = fmt.Errorf("SWIFT Scheme invalid (https or http)")
		}
	}
	if err == nil {
		if c.NodeCount <= 0 {
			err = fmt.Errorf("SWIFT NodeCount must be greater than 0")
		} else {
			log.Printf("SWIFT:NodeCount: %d\n", c.NodeCount)
		}
	}
	if err == nil {
		if c.StorageOperationTimeout <= 0 {
			err = fmt.Errorf("SWIFT storageOperationTimeout must be greater than 0")
		} else {
			log.Printf("SWIFT:StorageOperationTimeout: %d\n", c.StorageOperationTimeout)
		}
	}
	if err == nil {
		if c.HomeNodeTimeout <= 0 {
			err = fmt.Errorf("SWIFT HomeNodeTimeout must be greater than 0")
		}
	}
	if err == nil {
		if c.AlivePollingSeconds < 0 {
			err = fmt.Errorf("SWIFT AlivePollingSeconds must 0 or positive")
		} else {
			log.Printf("SWIFT:AlivePollingSeconds: %d\n", c.AlivePollingSeconds)
		}
	}
	if err == nil {
		if c.StorageManagerRefreshMinutes <= 0 {
			err = fmt.Errorf("SWIFT StorageManagerRefreshMinutes must be greater than 0")
		} else {
			log.Printf("SWIFT:StorageManagerRefreshMinutes: %d\n", c.StorageManagerRefreshMinutes)
		}
	}
	return err
}
