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
	"github.com/cockroachdb/cockroach-prod/drivers"
	"github.com/cockroachdb/cockroach/util"
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

// TODO(marc): this is the old config setup. This need to change after the merge.
type NodeInfo struct {
}

func (n *NodeInfo) DataDir() string {
	return ""
}

func (n *NodeInfo) IPAddress() string {
	return ""
}

func (n *NodeInfo) GossipAddress() string {
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

// GetNodeSettings takes a node name and unmarshalled json config
// and returns a filled NodeInfo.
func (g *Google) GetNodeSettings(name string, config interface{}) (drivers.NodeSettings, error) {
	return nil, util.Errorf("not implemented")
}

// ProcessFirstNode runs any steps needed after the first node was created.
func (g *Google) ProcessFirstNode(name string, config interface{}) error {
	return util.Errorf("not implemented")
}

// AddNode runs any steps needed to add a node (any node, not just the first one).
func (g *Google) AddNode(name string, config interface{}) error {
	return util.Errorf("not implemented")
}

// StartNode adds the node to the load balancer.
func (g *Google) StartNode(name string, config interface{}) error {
	return util.Errorf("not implemented")
}

// StopNode removes the node from the load balancer.
func (g *Google) StopNode(name string, config interface{}) error {
	return util.Errorf("not implemented")
}
