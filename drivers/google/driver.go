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

	// The package names are "compute", "resourceviews", etc... but we specify them
	// for clarify.
	compute "google.golang.org/api/compute/v1"
	resourceviews "google.golang.org/api/resourceviews/v1beta2"

	"github.com/cockroachdb/cockroach-prod/base"
	"github.com/cockroachdb/cockroach-prod/docker"
	"github.com/cockroachdb/cockroach-prod/drivers"
	"github.com/cockroachdb/cockroach/util"
	"github.com/cockroachdb/cockroach/util/log"
)

const (
	dockerMachineDriverName = "google"
	googleDataDir           = "/home/docker-user/data"
	// for now, we always run in zone <region>-a.
	// TODO(marc): allow custom value.
	zoneSuffix = "-a"
)

// Google implements a driver for Google Compute Engine.
type Google struct {
	context *base.Context
	region  string
	project string
	zone    string

	// Created at Init() time.
	// Most methods using the compute service can be found in compute.go
	computeService *compute.Service
	// Methods for instance groups (the API still calls them "resource views")
	// are in instance_groups.go
	instanceGroupsService *resourceviews.Service
}

// config contains the google-specific fields of the docker-machine config.
// Not all are specified, only those used here.
// Implements drivers.DriverConfig.
type config struct {
	MachineName string
	Zone        string

	// Fields not saved by docker-machine. We look them up.
	internalIPAddress     string
	link                  string
	forwardingRuleAddress string
}

// DataDir returns the data directory.
func (cfg *config) DataDir() string {
	return googleDataDir
}

// IPAddress returns the IP address we will listen on.
func (cfg *config) IPAddress() string {
	return cfg.internalIPAddress
}

// GossipAddress returns the address for the gossip network.
func (cfg *config) GossipAddress() string {
	return cfg.forwardingRuleAddress
}

// NewDriver returns an initialized Google driver.
func NewDriver(context *base.Context, region string) *Google {
	return &Google{
		context: context,
		region:  region,
		project: context.GCEProject,
		zone:    region + zoneSuffix,
	}
}

// Context returns the base context.
func (g *Google) Context() *base.Context {
	return g.context
}

// DockerMachineDriver returns the name of the docker-machine driver.
func (g *Google) DockerMachineDriver() string {
	return dockerMachineDriverName
}

// Init creates and and initializes the compute client.
func (g *Google) Init() error {
	// Initialize auth: we re-use the code from docker-machine.
	oauthClient, err := newOauthClient(g.context.GCETokenPath)
	if err != nil {
		return util.Errorf("could not get OAuth client: %v", err)
	}

	cSvc, err := compute.New(oauthClient)
	if err != nil {
		return util.Errorf("could not get Compute service: %v", err)
	}
	g.computeService = cSvc

	ivSvc, err := resourceviews.New(oauthClient)
	if err != nil {
		return util.Errorf("could not get Compute service: %v", err)
	}
	g.instanceGroupsService = ivSvc

	if err = g.checkProjectExists(); err != nil {
		return util.Errorf("invalid project %q: %v", g.project, err)
	}

	log.Infof("validated project name: %q", g.project)
	return nil
}

// DockerMachineCreateArgs returns the list of driver-specific arguments
// to pass to 'docker-machine create'
// TODO(marc): there are many other flags, see 'docker-machine help create'
func (g *Google) DockerMachineCreateArgs() []string {
	return []string{
		"--google-project", g.project,
		"--google-auth-token", g.context.GCETokenPath,
	}
}

// PrintStatus prints the load balancer address to stdout.
func (g *Google) PrintStatus() {
	rule, err := g.getForwardingRule()
	if err != nil {
		fmt.Println("Forwarding Rule: not found:", err)
		return
	}
	fmt.Printf("Forwarding Rule: %s:%d\n", rule.IPAddress, g.context.Port)
}

// GetNodeConfig takes a node name and reads its docker-machine config.
func (g *Google) GetNodeConfig(name string) (*drivers.HostConfig, error) {
	cfg := &drivers.HostConfig{
		Driver: &config{},
	}

	// Parse the config file.
	err := docker.GetHostConfig(name, cfg)
	if err != nil {
		return nil, err
	}

	// We need the just-parsed driver config.
	driverCfg := cfg.Driver.(*config)

	// Lookup the instance, there are a few fields docker-machine does not save.
	instance, err := g.getInstanceDetails(driverCfg.MachineName)
	if err != nil {
		return nil, err
	}

	driverCfg.internalIPAddress = instance.NetworkInterfaces[0].NetworkIP
	driverCfg.link = instance.SelfLink

	// Lookup the forwarding rule.
	rule, err := g.getForwardingRule()
	if err != nil {
		return nil, err
	}
	cfg.Driver.(*config).forwardingRuleAddress = rule.IPAddress

	return cfg, err
}

// AfterFirstNode runs any steps needed after the first node was created.
// The cloud compute HTTP load balancer setup is really convoluted:
// https://cloud.google.com/compute/docs/load-balancing/http/#fundamentals
// Things we create (children must be created before their parents):
// - firewall rule
// - global forwarding rule
//   - HTTP proxy
//     - URL map
//       - backend service
//         - health check
//         - instance group
func (g *Google) AfterFirstNode() error {
	log.Info("creating firewall rule")
	_, err := g.createFirewallRule()
	if err != nil {
		return util.Errorf("failed to create firewall rule: %v", err)
	}

	log.Info("creating instance group")
	instanceGroupLink, err := g.createInstanceGroup()
	if err != nil {
		return util.Errorf("failed to create instance group: %v", err)
	}

	log.Info("creating health check")
	healthCheckLink, err := g.createHealthCheck()
	if err != nil {
		return util.Errorf("failed to create health check: %v", err)
	}

	log.Info("creating backend service")
	backendServiceLink, err := g.createBackendService(healthCheckLink, instanceGroupLink)
	if err != nil {
		return util.Errorf("failed to create backend service: %v", err)
	}

	log.Info("creating URL map")
	urlMapLink, err := g.createURLMap(backendServiceLink)
	if err != nil {
		return util.Errorf("failed to create URL map: %v", err)
	}

	log.Info("creating HTTP proxy")
	httpProxyLink, err := g.createHTTPProxy(urlMapLink)
	if err != nil {
		return util.Errorf("failed to create HTTP proxy: %v", err)
	}

	log.Info("creating forwarding rule")
	_, err = g.createForwardingRule(httpProxyLink)
	if err != nil {
		return util.Errorf("failed to create forwarding rule: %v", err)
	}
	return nil
}

// StartNode adds the node to the load balancer.
func (g *Google) StartNode(name string, cfg *drivers.HostConfig) error {
	log.Infof("Adding node %s to instance group", name)
	return g.addInstanceToGroup(cfg.Driver.(*config).link)
}

// StopNode removes the node from the load balancer.
func (g *Google) StopNode(name string, cfg *drivers.HostConfig) error {
	log.Infof("Removing node %s from instance group", name)
	return g.removeInstanceFromGroup(cfg.Driver.(*config).link)
}
