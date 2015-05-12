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
	cockroachProtocol  = "tcp"
	allIPAddresses     = "0.0.0.0/0"
	firewallRuleName   = "cockroach-firewall"
	forwardingRuleName = "cockroach-forward-rule"
	healthCheckName    = "cockroach-health-check"
	backendServiceName = "cockroach-backend"
	urlMapName         = "cockroach-url-map"
	httpProxyName      = "cockroach-proxy"
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

// getFirewallRule looks for the cockroach firewall rule and returns it.
func (g *Google) getFirewallRule() (*compute.Firewall, error) {
	return g.computeService.Firewalls.Get(g.project, firewallRuleName).Do()
}

// createFirewallRule creates the cockroach firewall if it does not exist.
// It returns its resource link.
func (g *Google) createFirewallRule() (string, error) {
	if rule, err := g.getFirewallRule(); err == nil {
		log.Infof("found FirewallRule %s: %s", firewallRuleName, rule.SelfLink)
		return rule.SelfLink, nil
	}

	op, err := g.computeService.Firewalls.Insert(g.project,
		&compute.Firewall{
			Name: firewallRuleName,
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
		return "", err
	}
	if err = g.waitForOperation(op); err != nil {
		return "", err
	}
	log.Infof("created FirewallRule %s: %s", firewallRuleName, op.TargetLink)
	return op.TargetLink, nil
}

// getForwardingRule looks for the cockroach forwarding rule.
func (g *Google) getForwardingRule() (*compute.ForwardingRule, error) {
	return g.computeService.GlobalForwardingRules.Get(g.project, forwardingRuleName).Do()
}

// createForwardingRule creates the cockroach forwarding rule if it does not exist.
// Requires a resolvable target link. It should be a HTTP Proxy.
// Returns the forwarding rule resource link.
func (g *Google) createForwardingRule(targetLink string) (string, error) {
	if rule, err := g.getForwardingRule(); err == nil {
		log.Infof("found ForwardingRule %s: %s", forwardingRuleName, rule.SelfLink)
		return rule.SelfLink, nil
	}

	op, err := g.computeService.GlobalForwardingRules.Insert(g.project,
		&compute.ForwardingRule{
			Name:       forwardingRuleName,
			IPProtocol: cockroachProtocol,
			PortRange:  fmt.Sprintf("%d", g.context.Port),
			Target:     targetLink,
		}).Do()
	if err != nil {
		return "", err
	}
	if err = g.waitForOperation(op); err != nil {
		return "", err
	}

	log.Infof("created ForwardingRule %s: %s", forwardingRuleName, op.TargetLink)
	return op.TargetLink, nil
}

// getHealthCheck looks for the cockroach health check.
func (g *Google) getHealthCheck() (*compute.HttpHealthCheck, error) {
	return g.computeService.HttpHealthChecks.Get(g.project, healthCheckName).Do()
}

// createHealthCheck creates the cockroach health check if it does not exist.
// Returns its resource link.
func (g *Google) createHealthCheck() (string, error) {
	if check, err := g.getHealthCheck(); err == nil {
		log.Infof("found HealthCheck %s: %s", healthCheckName, check.SelfLink)
		return check.SelfLink, nil
	}

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
		return "", err
	}
	if err = g.waitForOperation(op); err != nil {
		return "", err
	}

	log.Infof("created HealthCheck %s: %s", healthCheckName, op.TargetLink)
	return op.TargetLink, nil
}

// getBackendService looks for the cockroach backend service.
func (g *Google) getBackendService() (*compute.BackendService, error) {
	return g.computeService.BackendServices.Get(g.project, backendServiceName).Do()
}

// createBackendService creates the cockroach backend service if it does not exist.
// Requires a resolvable health check and instance group.
// Returns the backend service resource link.
func (g *Google) createBackendService(healthCheckLink, instanceGroupLink string) (string, error) {
	if backend, err := g.getBackendService(); err == nil {
		log.Infof("found BackendService %s: %s", backendServiceName, backend.SelfLink)
		return backend.SelfLink, nil
	}

	op, err := g.computeService.BackendServices.Insert(g.project,
		&compute.BackendService{
			Name:         backendServiceName,
			HealthChecks: []string{healthCheckLink},
			Backends: []*compute.Backend{
				{Group: instanceGroupLink},
			},
		}).Do()
	if err != nil {
		return "", err
	}
	if err = g.waitForOperation(op); err != nil {
		return "", err
	}

	log.Infof("created BackendService %s: %s", backendServiceName, op.TargetLink)
	return op.TargetLink, nil
}

// getURLMap looks for the cockroach backend service.
func (g *Google) getURLMap() (*compute.UrlMap, error) {
	return g.computeService.UrlMaps.Get(g.project, urlMapName).Do()
}

// createURLMap creates the cockroach url map if it does not exist.
// Requires a resolvable backend service.
// Returns the url map resource link.
func (g *Google) createURLMap(backendServiceLink string) (string, error) {
	if urlMap, err := g.getURLMap(); err == nil {
		log.Infof("found URLMap %s: %s", urlMapName, urlMap.SelfLink)
		return urlMap.SelfLink, nil
	}

	op, err := g.computeService.UrlMaps.Insert(g.project,
		&compute.UrlMap{
			Name:           urlMapName,
			DefaultService: backendServiceLink,
		}).Do()
	if err != nil {
		return "", err
	}
	if err = g.waitForOperation(op); err != nil {
		return "", err
	}

	log.Infof("created URLMap %s: %s", urlMapName, op.TargetLink)
	return op.TargetLink, nil
}

// getHTTPProxy looks for the cockroach http proxy.
func (g *Google) getHTTPProxy() (*compute.TargetHttpProxy, error) {
	return g.computeService.TargetHttpProxies.Get(g.project, httpProxyName).Do()
}

// createHTTPProxy creates the cockroach http proxy if it does not exist.
// Requires a resolvable url map.
// Returns the http proxy resource link.
func (g *Google) createHTTPProxy(urlMapLink string) (string, error) {
	if proxy, err := g.getHTTPProxy(); err == nil {
		log.Infof("found HTTPProxy %s: %s", httpProxyName, proxy.SelfLink)
		return proxy.SelfLink, nil
	}

	op, err := g.computeService.TargetHttpProxies.Insert(g.project,
		&compute.TargetHttpProxy{
			Name:   httpProxyName,
			UrlMap: urlMapLink,
		}).Do()
	if err != nil {
		return "", err
	}
	if err = g.waitForOperation(op); err != nil {
		return "", err
	}

	log.Infof("create HTTPProxy %s: %s", httpProxyName, op.TargetLink)
	return op.TargetLink, nil
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
// We determine whether it's a zone or global operation by parsing its resource link.
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
