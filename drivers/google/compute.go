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

	// The package is called "compute" but is in v1. Specify import name for clarify.
	compute "google.golang.org/api/compute/v1"

	"github.com/cockroachdb/cockroach/util/log"
)

const (
	firewallName       = "cockroach"
	cockroachProtocol  = "tcp"
	allIPAddresses     = "0.0.0.0/0"
	forwardingRuleName = "cockroach-lb-rule"
	healthCheckName    = "cockroach-lb-healthcheck"
	targetPoolName     = "cockroach-lb-targetpool"
	// TODO(marc): some of these should be pulled from cockroach/base/Context or similar.
	healthCheckPath  = "/_status/"
	computeURIPrefix = "https://www.googleapis.com/compute/v1/"
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

// Lookup forwarding rule. For now, we use "network forwarding" instead
// "HTTP/S load balancing" which is still beta.
func lookupForwardingRule(service *compute.Service, project, region string) (*compute.ForwardingRule, error) {
	return service.ForwardingRules.Get(project, region, forwardingRuleName).Do()
}

// Create health check, target pool, and forward rule. In that order.
func createForwardingRule(service *compute.Service, project, region string, cockroachPort int64) error {
	// Create HTTP Health checks.
	check, err := service.HttpHealthChecks.Insert(project,
		&compute.HttpHealthCheck{
			Name:               healthCheckName,
			Port:               cockroachPort,
			RequestPath:        healthCheckPath,
			CheckIntervalSec:   2,
			TimeoutSec:         1,
			HealthyThreshold:   2,
			UnhealthyThreshold: 2,
		}).Do()
	if err != nil {
		return err
	}
	log.Infof("created HealthCheck %s: %s", healthCheckName, check.TargetLink)

	// Create target pool.
	pool, err := service.TargetPools.Insert(project, region,
		&compute.TargetPool{
			Name:            targetPoolName,
			SessionAffinity: "NONE",
			HealthChecks:    []string{check.TargetLink},
		}).Do()
	if err != nil {
		return err
	}
	log.Infof("created TargetPool %s: %s", targetPoolName, pool.TargetLink)

	// Create forwarding rule.
	rule, err := service.ForwardingRules.Insert(project, region,
		&compute.ForwardingRule{
			Name:       forwardingRuleName,
			IPProtocol: cockroachProtocol,
			PortRange:  fmt.Sprintf("%d", cockroachPort),
			Target:     pool.TargetLink,
		}).Do()
	if err != nil {
		return err
	}
	log.Infof("created ForwardingRule %s: %s", forwardingRuleName, rule.TargetLink)

	return nil
}

// Add the given instance (full resource link) to the cockroach target pool.
func addTarget(service *compute.Service, project, region, instanceLink string) error {
	_, err := service.TargetPools.AddInstance(project, region, targetPoolName,
		&compute.TargetPoolsAddInstanceRequest{
			Instances: []*compute.InstanceReference{
				{
					Instance: instanceLink,
				},
			},
		}).Do()
	return err
}

// removes the given instance (full resource link) from the cockroach target pool.
func removeTarget(service *compute.Service, project, region, instanceLink string) error {
	_, err := service.TargetPools.RemoveInstance(project, region, targetPoolName,
		&compute.TargetPoolsRemoveInstanceRequest{
			Instances: []*compute.InstanceReference{
				{
					Instance: instanceLink,
				},
			},
		}).Do()
	return err
}
