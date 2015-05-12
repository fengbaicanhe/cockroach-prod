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
	"strings"
	"time"

	// The package is called "compute" but is in v1. Specify import name for clarify.
	compute "google.golang.org/api/compute/v1"

	"github.com/cockroachdb/cockroach/util"
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
	healthCheckPath = "/_status/"
)

// computeOpError wraps a compute.OperationErrorErrors to implement error.
type computeOpError struct {
	compute.OperationErrorErrors
}

func (err computeOpError) Error() string {
	return err.Code
}

// Return the base path for global objects.
func globalBasePath(service *compute.Service, project string) string {
	return fmt.Sprintf("%s%s/global/", service.BasePath, project)
}

// Return the base path for region objects.
func regionBasePath(service *compute.Service, project, region string) string {
	return fmt.Sprintf("%s%s/regions/%s/", service.BasePath, project, region)
}

// Return the base path for zone objects.
func zoneBasePath(service *compute.Service, project, zone string) string {
	return fmt.Sprintf("%s%s/zones/%s/", service.BasePath, project, zone)
}

// Check whether the named project exists. Returns nil if it does.
func (g *Google) checkProjectExists() error {
	_, err := g.computeService.Projects.Get(g.project).Do()
	return err
}

// Lookup and return the instance details.
func (g *Google) getInstanceDetails(machine string) (*compute.Instance, error) {
	return g.computeService.Instances.Get(g.project, g.zone, machine).Do()
}

// Create the firewall rule. Noop if it exists.
func (g *Google) createFirewallRule() error {
	if _, err := g.computeService.Firewalls.Get(g.project, firewallName).Do(); err == nil {
		return nil
	}
	op, err := g.computeService.Firewalls.Insert(g.project,
		&compute.Firewall{
			Name: firewallName,
			Allowed: []*compute.FirewallAllowed{
				{
					IPProtocol: cockroachProtocol,
					Ports: []string{
						fmt.Sprintf("%d", g.context.Port),
					},
				},
			},
			SourceRanges: []string{
				allIPAddresses,
			},
		}).Do()
	if err != nil {
		return err
	}
	return g.waitForOperation(op)
}

// Lookup forwarding rule. For now, we use "network forwarding" instead
// "HTTP/S load balancing" which is still beta.
func (g *Google) lookupForwardingRule() (*compute.ForwardingRule, error) {
	return g.computeService.ForwardingRules.Get(g.project, g.region, forwardingRuleName).Do()
}

// Create health check, target pool, and forward rule. In that order.
func (g *Google) createForwardingRule() error {
	// Create HTTP Health checks.
	var healthCheckLink string
	if check, err := g.computeService.HttpHealthChecks.Get(g.project, healthCheckName).Do(); err == nil {
		healthCheckLink = check.SelfLink
		log.Infof("found HealthCheck %s: %s", healthCheckName, healthCheckLink)
	} else {
		op, err := g.computeService.HttpHealthChecks.Insert(g.project,
			&compute.HttpHealthCheck{
				Name:               healthCheckName,
				Port:               g.context.Port,
				RequestPath:        healthCheckPath,
				CheckIntervalSec:   2,
				TimeoutSec:         1,
				HealthyThreshold:   2,
				UnhealthyThreshold: 2,
			}).Do()
		if err != nil {
			return err
		}
		if err = g.waitForOperation(op); err != nil {
			return err
		}
		healthCheckLink = op.TargetLink
		log.Infof("created HealthCheck %s: %s", healthCheckName, healthCheckLink)
	}

	// Create target pool.
	var targetPoolLink string
	if pool, err := g.computeService.TargetPools.Get(g.project, g.region, targetPoolName).Do(); err == nil {
		targetPoolLink = pool.SelfLink
		log.Infof("found TargetPool %s: %s", targetPoolName, targetPoolLink)
	} else {
		op, err := g.computeService.TargetPools.Insert(g.project, g.region,
			&compute.TargetPool{
				Name:            targetPoolName,
				SessionAffinity: "NONE",
				HealthChecks:    []string{healthCheckLink},
			}).Do()
		if err != nil {
			return err
		}
		if err = g.waitForOperation(op); err != nil {
			return err
		}
		targetPoolLink = op.TargetLink
		log.Infof("created TargetPool %s: %s", targetPoolName, targetPoolLink)
	}

	// Create forwarding rule.
	var forwardingRuleLink string
	if rule, err := g.computeService.ForwardingRules.Get(g.project, g.region, forwardingRuleName).Do(); err == nil {
		forwardingRuleLink = rule.SelfLink
		log.Infof("found ForwardingRule %s: %s", forwardingRuleName, forwardingRuleLink)
	} else {
		op, err := g.computeService.ForwardingRules.Insert(g.project, g.region,
			&compute.ForwardingRule{
				Name:       forwardingRuleName,
				IPProtocol: cockroachProtocol,
				PortRange:  fmt.Sprintf("%d", g.context.Port),
				Target:     targetPoolLink,
			}).Do()
		if err != nil {
			return err
		}
		if err = g.waitForOperation(op); err != nil {
			return err
		}
		forwardingRuleLink = op.TargetLink
		log.Infof("created ForwardingRule %s: %s", forwardingRuleName, forwardingRuleLink)
	}

	return nil
}

// Add the given instance (full resource link) to the cockroach target pool.
func (g *Google) addTarget(instanceLink string) error {
	op, err := g.computeService.TargetPools.AddInstance(g.project, g.region, targetPoolName,
		&compute.TargetPoolsAddInstanceRequest{
			Instances: []*compute.InstanceReference{
				{
					Instance: instanceLink,
				},
			},
		}).Do()
	if err != nil {
		return err
	}
	return g.waitForOperation(op)
}

// removes the given instance (full resource link) from the cockroach target pool.
func (g *Google) removeTarget(instanceLink string) error {
	op, err := g.computeService.TargetPools.RemoveInstance(g.project, g.region, targetPoolName,
		&compute.TargetPoolsRemoveInstanceRequest{
			Instances: []*compute.InstanceReference{
				{
					Instance: instanceLink,
				},
			},
		}).Do()
	if err != nil {
		return err
	}
	return g.waitForOperation(op)
}

func errorFromOperationError(opError *compute.OperationError) error {
	if opError == nil {
		return nil
	}
	if len(opError.Errors) == 0 {
		return nil
	}
	return computeOpError{*opError.Errors[0]}
}

// Repeatedly poll the given operation until its status is DONE, then return its Error.
// We determine whether it's a zone or global operation by parsing its self-link.
// TODO(marc): give up after a while.
func (g *Google) waitForOperation(op *compute.Operation) error {
	// Early out for finished ops.
	if op.Status == "DONE" {
		if log.V(1) {
			log.Infof("Operation %s %s: DONE, err=%v", op.OperationType, op.TargetLink,
				errorFromOperationError(op.Error))
		}
		return errorFromOperationError(op.Error)
	}

	var isGlobal, isRegion bool
	if strings.HasPrefix(op.SelfLink, globalBasePath(g.computeService, g.project)) {
		isGlobal = true
	} else if strings.HasPrefix(op.SelfLink, regionBasePath(g.computeService, g.project, g.region)) {
		isRegion = true
	} else if strings.HasPrefix(op.SelfLink, zoneBasePath(g.computeService, g.project, g.zone)) {
	} else {
		log.Fatalf("unsupported operation (expect global, region, or zone): %+v", op)
	}

	for {
		var liveOp *compute.Operation
		var err error
		if isGlobal {
			liveOp, err = g.computeService.GlobalOperations.Get(g.project, op.Name).Do()
		} else if isRegion {
			liveOp, err = g.computeService.RegionOperations.Get(g.project, g.region, op.Name).Do()
		} else {
			liveOp, err = g.computeService.ZoneOperations.Get(g.project, g.zone, op.Name).Do()
		}
		// This usually indicates a bad operation object.
		if err != nil {
			return util.Errorf("could not lookup operation %+v: %s", op, err)
		}
		if log.V(1) {
			log.Infof("Operation %s %s: %s, err=%v", liveOp.OperationType, liveOp.TargetLink,
				liveOp.Status, errorFromOperationError(liveOp.Error))
		}
		if liveOp.Status == "DONE" {
			return errorFromOperationError(liveOp.Error)
		}
		time.Sleep(time.Second)
	}

	return nil
}
