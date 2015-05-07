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

package google

import (
	"fmt"

	"google.golang.org/api/compute/v1"
)

const (
	firewallName      = "cockroach"
	cockroachProtocol = "tcp"
	allIPAddresses    = "0.0.0.0/0"
)

// Check whether the named project exists. Returns nil if it does.
func checkProjectExists(service *compute.Service, project string) error {
	_, err := service.Projects.Get(project).Do()
	return err
}

// Lookup and return the instance details.
func getInstanceDetails(service *compute.Service, project, zone, machine string) (*compute.Instance, error) {
	return service.Instances.Get(project, zone, machine).Do()
}

// Create firewall rule.
// The API is happy inserting an existing one, so don't check for existence first.
func createFirewallRule(service *compute.Service, project string, cockroachPort int64) error {
	_, err := service.Firewalls.Insert(project,
		&compute.Firewall{
			Name: firewallName,
			Allowed: []*compute.FirewallAllowed{
				{
					IPProtocol: cockroachProtocol,
					Ports: []string{
						fmt.Sprintf("%d", cockroachPort),
					},
				},
			},
			SourceRanges: []string{
				allIPAddresses,
			},
		}).Do()
	return err
}
