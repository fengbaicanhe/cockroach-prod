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

	"github.com/cockroachdb/cockroach-prod/base"
	"github.com/cockroachdb/cockroach-prod/docker"
	"github.com/cockroachdb/cockroach-prod/drivers"
	"github.com/cockroachdb/cockroach/util"
	"github.com/docker/machine/log"
)

const (
	dockerMachineDriverName = "google"
)

// Google implements a driver for Google Compute Engine.
type Google struct {
	context *base.Context
	region  string
	project string
}

// config contains the google-specific fields of the docker-machine config.
// Not all are specified, only those used here.
// Implements drivers.DriverConfig.
type config struct {
}

// DataDir returns the data directory.
func (cfg *config) DataDir() string {
	return ""
}

// IPAddress returns the IP address we will listen on.
func (cfg *config) IPAddress() string {
	return ""
}

// GossipAddress returns the address for the gossip network.
func (cfg *config) GossipAddress() string {
	return ""
}

// NewDriver returns an initialized Google driver.
func NewDriver(context *base.Context, region string) *Google {
	return &Google{
		context: context,
		region:  region,
		project: context.GCEProject,
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

// Init creates and compute client.
func (g *Google) Init() error {
	// Initialize auth: we re-use the code from docker-machine.
	svc, err := newGCEService(g.context.GCETokenPath)
	if err != nil {
		return util.Errorf("could not get OAuth token: %v", err)
	}

	_, err = svc.Projects.Get(g.project).Do()
	if err != nil {
		return util.Errorf("invalid project %q: %v", g.project, err)
	}

	log.Infof("validated project name: %q", g.project)
	// Return unimplemented for now so we don't proceed.
	return util.Errorf("not implemented")
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
	fmt.Printf("Nothing yet")
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

	return cfg, err
}

// AfterFirstNode runs any steps needed after the first node was created.
func (g *Google) AfterFirstNode() error {
	return util.Errorf("not implemented")
}

// AddNode runs any steps needed to add a node (any node, not just the first one).
func (g *Google) AddNode(name string, cfg *drivers.HostConfig) error {
	return util.Errorf("not implemented")
}

// StartNode adds the node to the load balancer.
func (g *Google) StartNode(name string, cfg *drivers.HostConfig) error {
	return util.Errorf("not implemented")
}

// StopNode removes the node from the load balancer.
func (g *Google) StopNode(name string, cfg *drivers.HostConfig) error {
	return util.Errorf("not implemented")
}
