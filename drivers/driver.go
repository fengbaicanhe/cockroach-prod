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

// HostConfig describes the docker-machine host config.
// Driver is the driver-specific config.
// We only specify the fields we currently use. For the full list, see:
// https://github.com/docker/machine/blob/master/libmachine/host.go
type HostConfig struct {
	DriverName string
	Driver     HostDriverConfig
}

// HostDriverConfig describes the docker-machine host driver configs.
// It is implemented by each driver.
type HostDriverConfig interface {
	// DataDir is the directory used as the data directory.
	DataDir() string
	// IPAddress is the node address cockroach should bind to.
	IPAddress() string
	// GossipAddress is the address to reach the gossip network.
	GossipAddress() string
}

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

	// GetNodeConfig takes a node name and reads its docker-machine config.
	GetNodeConfig(name string) (*HostConfig, error)

	// AfterFirstNode runs any steps needed after the first node was created.
	AfterFirstNode() error

	// AddNode runs any steps needed for new nodes (not just the first one).
	AddNode(name string, config *HostConfig) error

	// StartNode runs any steps needed when starting an existing node.
	StartNode(name string, config *HostConfig) error

	// StopNode runs any steps needed when stopping a node.
	StopNode(name string, config *HostConfig) error
}
