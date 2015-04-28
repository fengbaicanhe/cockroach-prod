// Copyright 2015 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.
//
// Author: Marc Berhault (marc@cockroachlabs.com)

package amazon

import (
	"io/ioutil"
	"os"
	"regexp"

	"github.com/cockroachdb/cockroach/util"
)

const (
	credentialsPath = "${HOME}/.aws/credentials"
)

var (
	keyIDRegexp = regexp.MustCompile(`\naws_access_key_id = ([^\n]+)\n`)
	keyRegexp   = regexp.MustCompile(`\naws_secret_access_key = ([^\n]+)\n`)
)

// LoadAWSCredentials looks for the .aws/credentials file, parses it, and returns the
// key-id and secret-key.
func LoadAWSCredentials() (string, string, error) {
	credPath := os.ExpandEnv(credentialsPath)
	contents, err := ioutil.ReadFile(credPath)
	if err != nil {
		return "", "", err
	}

	match := keyIDRegexp.FindSubmatch(contents)
	if match == nil || len(match) != 2 {
		return "", "", util.Errorf("could not extract aws_access_key_id from %s", credPath)
	}
	keyID := match[1]

	match = keyRegexp.FindSubmatch(contents)
	if match == nil || len(match) != 2 {
		return "", "", util.Errorf("could not extract aws_secret_access_key from %s", credPath)
	}
	key := match[1]

	return string(keyID), string(key), nil
}

// ParseDockerMachineConfig takes an unmarshalled docker-machine json config and
// extracts interesting fields from a docker machine amazon driver config.
// TODO(marc): we should probably pull in docker machine here.
func ParseDockerMachineConfig(config interface{}) (*NodeInfo, error) {
	machineConfig, ok := config.(map[string]interface{})
	if !ok {
		return nil, util.Errorf("unable to parse json machine config: %v", config)
	}
	driverConfig, ok := machineConfig["Driver"]
	if !ok {
		return nil, util.Errorf("missing key \"Driver\" in json config: %v", config)
	}
	awsConfig, ok := driverConfig.(map[string]interface{})
	if !ok {
		return nil, util.Errorf("unable to parse json driver config: %v", driverConfig)
	}

	instanceID, ok := awsConfig["InstanceId"]
	if !ok {
		return nil, util.Errorf("missing key \"InstanceId\" in driver config: %v", awsConfig)
	}

	securityGroupID, ok := awsConfig["SecurityGroupId"]
	if !ok {
		return nil, util.Errorf("missing key \"SecurityGroupId\" in driver config: %v", awsConfig)
	}

	ipAddress, ok := awsConfig["PrivateIPAddress"]
	if !ok {
		return nil, util.Errorf("missing key \"PrivateIPAddress\" in driver config: %v", awsConfig)
	}

	zone, ok := awsConfig["Zone"]
	if !ok {
		return nil, util.Errorf("missing key \"Zone\" in driver config: %v", awsConfig)
	}

	return &NodeInfo{
		instanceID:       instanceID.(string),
		securityGroupID:  securityGroupID.(string),
		privateIPAddress: ipAddress.(string),
		zone:             zone.(string),
	}, nil
}
