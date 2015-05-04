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

package drivers

import "github.com/cockroachdb/cockroach-prod/base"

// Driver is the interface for all drivers.
type Driver interface {
	// Context returns the base context.
	Context() *base.Context

	// DockerMachineDriver returns the name of the docker-machine driver.
	DockerMachineDriver() string

	// Init is called when creating the driver. This will typically
	// setup credentials.
	Init() error

	// DockerMachineCreateArgs returns the list of driver-specific arguments
	// to pass to 'docker-machine create'
	DockerMachineCreateArgs() []string

	// PrintStatus asks the driver to print some basic status to stdout.
	PrintStatus()

	// GetNodeSettings takes a node name and unmarshalled json config
	// and returns a filled in driver.NodeSettings.
	GetNodeSettings(name string, config interface{}) (NodeSettings, error)

	// ProcessFirstNode runs any steps needed after the first node was created.
	// This takes the unmarshalled json config and node name.
	ProcessFirstNode(name string, config interface{}) error

	// AddNode runs any steps needed for new nodes (not just the first one).
	// This takes the unmarshalled json config and node name.
	AddNode(name string, config interface{}) error

	// StartNode runs any steps needed when starting an existing node.
	// This takes the unmarshalled json config and node name.
	StartNode(name string, config interface{}) error

	// StopNode runs any steps needed when stopping a node.
	// This takes the unmarshalled json config and node name.
	StopNode(name string, config interface{}) error
}

// NodeSettings is a set of node parameters needed outside the driver.
type NodeSettings interface {
	// DataDir is the directory used as the data directory.
	DataDir() string
	// IPAddress is the node address cockroach should bind to.
	IPAddress() string
	// GossipAddress is the address to reach the gossip network.
	GossipAddress() string
}
