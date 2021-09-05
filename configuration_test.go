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

import "testing"

func TestLocalConfigurationSettings(t *testing.T) {
	c := NewConfig("appsettings.test.local")
	if c.SwiftFile == "" {
		t.Error("SWIFT file not set")
		return
	}
}

func TestLocalConfigurationEnvironment(t *testing.T) {
	e := "TEST ENV SWIFT FILE"
	t.Setenv("SWIFT_FILE", e)
	c := NewConfig("appsettings.test.none")
	if c.SwiftFile != e {
		t.Error("SWIFT file not expected value")
		return
	}
}

func TestAwsConfigurationSettings(t *testing.T) {
	c := NewConfig("appsettings.test.aws")
	if c.AwsEnabled == false {
		t.Error("AWS Enabled not set")
		return
	}
}

func TestAwsConfigurationEnvironment(t *testing.T) {
	t.Setenv("AWS_ENABLED", "true")
	c := NewConfig("appsettings.test.none")
	if c.AwsEnabled != true {
		t.Error("AWS Enabled not expected value")
		return
	}
}

func TestGcpConfigurationSettings(t *testing.T) {
	c := NewConfig("appsettings.test.gcp")
	if c.GcpProject == "" {
		t.Error("GCP Project not set")
		return
	}
}

func TestGcpConfigurationEnvironment(t *testing.T) {
	e := "PROJECT NAME"
	t.Setenv("GCP_PROJECT", e)
	c := NewConfig("appsettings.test.none")
	if c.GcpProject != e {
		t.Error("GCP Project not expected value")
		return
	}
}

func TestAzureConfigurationSettings(t *testing.T) {
	c := NewConfig("appsettings.test.azure")
	if c.AzureStorageAccount == "" || c.AzureStorageAccessKey == "" {
		t.Error("Azure not set")
		return
	}
}

func TestAzureConfigurationEnvironment(t *testing.T) {
	ea := "ACCOUNT"
	ek := "KEY"
	t.Setenv("AZURE_STORAGE_ACCOUNT", ea)
	t.Setenv("AZURE_STORAGE_ACCESS_KEY", ek)
	c := NewConfig("appsettings.test.none")
	if c.AzureStorageAccount != ea || c.AzureStorageAccessKey != ek {
		t.Error("Azure not expected value")
		return
	}
}

func TestConfigurationAll(t *testing.T) {
	c := NewConfig("appsettings.test.all")
	assert(t, c.AlivePollingSeconds == 2, "AlivePollingSeconds")
	assert(t, c.HomeNodeTimeout == 86400, "HomeNodeTimeout")
	assert(t, c.MaxStores == 100, "MaxStores")
	assert(t, c.NodeCount == 10, "NodeCount")
	assert(t, c.StorageManagerRefreshMinutes == 1, "StorageManagerRefreshMinutes")
	assert(t, c.StorageOperationTimeout == 30, "StorageOperationTimeout")
	assert(t, c.AwsEnabled == false, "AwsEnabled")
	assert(t, c.AzureStorageAccessKey == "", "AzureStorageAccessKey")
	assert(t, c.AzureStorageAccount == "", "AzureStorageAccount")
	assert(t, c.BackgroundColor == "#f5f5f5", "BackgroundColor")
	assert(t, c.GcpProject == "", "GcpProject")
	assert(t, c.Message == "Test Message", "Message")
	assert(t, c.MessageColor == "darkslategray", "MessageColor")
	assert(t, c.ProgressColor == "darkgreen", "ProgressColor")
	assert(t, c.Scheme == "https", "Scheme")
	assert(t, c.SwiftFile == ".swan/swiftnodes-production.json", "SwiftFile")
	assert(t, c.Title == "Test Title", "Title")
	assert(t, c.Debug == false, "Debug")
	c.Validate()
}

func assert(t *testing.T, v bool, m string) {
	if v == false {
		t.Fatal(m)
	}
}

// newConfigurationTest creates a test configuration instance for testing.
func newConfigurationTest() Configuration {
	var c Configuration
	c.Message = "Test Message"
	c.Title = "Test Title"
	c.BackgroundColor = "white"
	c.MessageColor = "black"
	c.ProgressColor = "blue"
	c.Scheme = "https"
	c.Debug = true
	c.MaxStores = 10
	c.StorageManagerRefreshMinutes = 10
	return c
}
