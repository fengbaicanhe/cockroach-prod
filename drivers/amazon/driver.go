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
	"fmt"

	"github.com/cockroachdb/cockroach-prod/base"
	"github.com/cockroachdb/cockroach-prod/drivers"
	"github.com/cockroachdb/cockroach/util"
	"github.com/cockroachdb/cockroach/util/log"
)

const (
	dockerMachineDriverName = "amazonec2"
	amazonDataDir           = "/home/ubuntu/data"
	cockroachProtocol       = "tcp"
)

// Amazon implements a driver for AWS.
type Amazon struct {
	context *base.Context
	region  string

	keyID string
	key   string

	vpcID string
}

type NodeInfo struct {
	instanceID          string
	securityGroupID     string
	privateIPAddress    string
	zone                string
	loadBalancerAddress string
}

func (n *NodeInfo) DataDir() string {
	return amazonDataDir
}

func (n *NodeInfo) IPAddress() string {
	return n.privateIPAddress
}

func (n *NodeInfo) GossipAddress() string {
	return n.loadBalancerAddress
}

// NewDriver returns an initialized Amazon driver.
// TODO(marc): we should keep initialized services (eg: elb, ec2).
func NewDriver(context *base.Context, region string) *Amazon {
	return &Amazon{
		context: context,
		region:  region,
	}
}

// Context returns the base context.
func (a *Amazon) Context() *base.Context {
	return a.context
}

// DockerMachineDriver returns the name of the docker-machine driver.
func (a *Amazon) DockerMachineDriver() string {
	return dockerMachineDriverName
}

// Init looks for AWS credentials.
func (a *Amazon) Init() error {
	var err error
	a.keyID, a.key, err = LoadAWSCredentials()
	if err != nil {
		return util.Errorf("unable to load AWS credentials: %v", err)
	}
	log.Infof("loaded AWS key: %s", a.keyID)

	// Find default VPC.
	a.vpcID, err = FindDefaultVPC(a.region)
	if err != nil {
		return util.Errorf("could not find default VPC ID in region %s: %v", a.region, err)
	}
	log.Infof("found default VPC id: %s", a.vpcID)

	return nil
}

// DockerMachineCreateArgs returns the list of driver-specific arguments
// to pass to 'docker-machine create'
// TODO(marc): there are many other flags, see 'docker-machine help create'
func (a *Amazon) DockerMachineCreateArgs() []string {
	return []string{
		"--amazonec2-access-key", a.keyID,
		"--amazonec2-secret-key", a.key,
		"--amazonec2-region", a.region,
		"--amazonec2-vpc-id", a.vpcID,
	}
}

// PrintStatus prints the load balancer address to stdout.
func (a *Amazon) PrintStatus() {
	log.Infof("looking for load balancer")
	elbDesc, err := FindCockroachELB(a.region)
	if err != nil || elbDesc == nil {
		log.Infof("could not find load balancer: %v", err)
		return
	}

	fmt.Printf("Load balancer: %s\n", *elbDesc.DNSName)
}

// GetNodeSettings takes a node name and unmarshalled json config
// and returns a filled NodeInfo.
func (a *Amazon) GetNodeSettings(name string, config interface{}) (drivers.NodeSettings, error) {
	settings, err := ParseDockerMachineConfig(config)
	if err != nil {
		return nil, err
	}

	elbDesc, err := FindCockroachELB(a.region)
	if err != nil || elbDesc == nil {
		return nil, util.Errorf("could not find load balancer: %v", err)
	}
	settings.loadBalancerAddress = *elbDesc.DNSName
	return settings, err
}

// ProcessFirstNode runs any steps needed after the first node was created.
// This tweaks the security group to allow cockroach ports and creates
// the load balancer.
func (a *Amazon) ProcessFirstNode(name string, config interface{}) error {
	nodeInfo, err := ParseDockerMachineConfig(config)
	if err != nil {
		return err
	}

	log.Infof("adding security group rule for node: %+v", nodeInfo)
	err = AddCockroachSecurityGroupIngress(a.region, a.context.Port, nodeInfo)
	if err != nil {
		return util.Errorf("failed to add security group rule for node %+v: %v", nodeInfo, err)
	}

	log.Infof("looking for load balancer")
	elbDesc, err := FindCockroachELB(a.region)
	if err != nil {
		return util.Errorf("failed to lookup existing load balancer for node %+v: %v", nodeInfo, err)
	}

	if elbDesc != nil {
		log.Info("found load balancer")
		return nil
	}

	log.Infof("no existing load balancer, creating one")
	err = CreateCockroachELB(a.region, a.context.Port, nodeInfo)
	if err != nil {
		return util.Errorf("failed to create load balancer for node %+v: %v", nodeInfo, err)
	}
	log.Info("created load balancer")
	return nil
}

// AddNode runs any steps needed to add a node (any node, not just the first one).
// This just adds the node to the load balancer, so for now, call StartNode.
func (a *Amazon) AddNode(name string, config interface{}) error {
	return a.StartNode(name, config)
}

// StartNode adds the node to the load balancer.
// ELB takes forever checking a stopped and started node,
// so we have to remove it at stopping time, and re-register it start time.
func (a *Amazon) StartNode(name string, config interface{}) error {
	nodeInfo, err := ParseDockerMachineConfig(config)
	if err != nil {
		return err
	}

	log.Infof("adding node %+v to load balancer", nodeInfo)
	err = AddNodeToELB(a.region, nodeInfo)
	if err != nil {
		return util.Errorf("failed to add node %+v to load balancer: %v", nodeInfo, err)
	}
	return nil
}

// StopNode removes the node from the load balancer.
// ELB takes forever checking a stopped and started node,
// so we have to remove it at stopping time, and re-register it start time.
func (a *Amazon) StopNode(name string, config interface{}) error {
	nodeInfo, err := ParseDockerMachineConfig(config)
	if err != nil {
		return err
	}

	log.Infof("removing node %+v from load balancer", nodeInfo)
	err = RemoveNodeFromELB(a.region, nodeInfo)
	if err != nil {
		return util.Errorf("failed to remove node %+v from load balancer: %v", nodeInfo, err)
	}
	return nil
}
