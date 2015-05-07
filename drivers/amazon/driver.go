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
	"github.com/cockroachdb/cockroach-prod/docker"
	"github.com/cockroachdb/cockroach-prod/drivers"
	"github.com/cockroachdb/cockroach/util"
	"github.com/cockroachdb/cockroach/util/log"
)

const (
	dockerMachineDriverName = "amazonec2"
	amazonDataDir           = "/home/ubuntu/data"
	defaultZone             = "a"
)

// Amazon implements a driver for AWS.
// This is not synchronized: be careful.
type Amazon struct {
	context *base.Context
	region  string
	zone    string

	keyID string
	key   string

	vpcID string
}

// config contains the amazon-specific fields of the docker-machine config.
// Not all are specified, only those used here.
// Implements drivers.DriverConfig.
type config struct {
	InstanceID       string
	SecurityGroupID  string
	PrivateIPAddress string
	Zone             string

	// non docker-machine fields:
	LoadBalancerAddress string `json:"-"`
}

// DataDir returns the data directory.
func (cfg *config) DataDir() string {
	return amazonDataDir
}

// IPAddress returns the IP address we will listen on.
func (cfg *config) IPAddress() string {
	return cfg.PrivateIPAddress
}

// GossipAddress returns the address for the gossip network.
func (cfg *config) GossipAddress() string {
	return cfg.LoadBalancerAddress
}

// NewDriver returns an initialized Amazon driver.
// TODO(marc): we should keep initialized services (eg: elb, ec2).
func NewDriver(context *base.Context, region string) *Amazon {
	return &Amazon{
		context: context,
		region:  region,
		zone:    defaultZone,
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
		"--amazonec2-zone", a.zone,
	}
}

// PrintStatus prints the load balancer address to stdout.
// Do not call the "getOrInit*" methods here, we only want to look things up.
func (a *Amazon) PrintStatus() {
	fmt.Println("Region:", a.region)

	dnsName, err := FindCockroachELB(a.region)
	if err != nil {
		fmt.Println("Load balancer: problem:", err)
	} else if dnsName == "" {
		fmt.Println("Load balancer: not found (you need to initialize the cluster)")
	} else {
		fmt.Println("Load balancer:", dnsName)
	}

	securityGroupID, err := FindSecurityGroup(a.region)
	if err != nil {
		fmt.Println("Security group: problem:", err)
	} else if securityGroupID == "" {
		fmt.Println("Security group: not found (you need to initialize the cluster)")
	} else {
		fmt.Println("Security group:", securityGroupID)
	}
}

// GetNodeConfig takes a node name and reads its docker-machine config.
// The LoadBalancerAddress is looked up and filled in.
func (a *Amazon) GetNodeConfig(name string) (*drivers.HostConfig, error) {
	cfg := &drivers.HostConfig{
		Driver: &config{},
	}

	// Parse the config file.
	err := docker.GetHostConfig(name, cfg)
	if err != nil {
		return nil, err
	}

	// Add the load balancer address.
	dnsName, err := FindCockroachELB(a.region)
	if err != nil || dnsName == "" {
		return nil, util.Errorf("could not find load balancer: %v", err)
	}
	cfg.Driver.(*config).LoadBalancerAddress = dnsName

	return cfg, err
}

// AfterFirstNode runs any steps needed after the first node was created.
// This tweaks the security group to allow cockroach ports and creates
// the load balancer.
func (a *Amazon) AfterFirstNode() error {
	securityGroupID, err := FindSecurityGroup(a.region)
	if err != nil {
		return err
	}

	log.Info("adding security group rule")
	err = AddCockroachSecurityGroupIngress(a.region, a.context.Port, securityGroupID)
	if err != nil {
		return util.Errorf("failed to add security group rule: %v", err)
	}

	_, err = FindOrCreateLoadBalancer(a.region, a.context.Port, a.zone, securityGroupID)
	return err
}

// StartNode adds the node to the load balancer.
// ELB takes forever checking a stopped and started node,
// so we have to remove it at stopping time, and re-register it start time.
func (a *Amazon) StartNode(name string, cfg *drivers.HostConfig) error {
	log.Infof("adding node %s to load balancer", name)
	err := AddNodeToELB(a.region, cfg.Driver.(*config).InstanceID)
	if err != nil {
		return util.Errorf("failed to add node %s (%+v) to load balancer: %v", name, cfg, err)
	}
	return nil
}

// StopNode removes the node from the load balancer.
// ELB takes forever checking a stopped and started node,
// so we have to remove it at stopping time, and re-register it start time.
func (a *Amazon) StopNode(name string, cfg *drivers.HostConfig) error {
	log.Infof("removing node %s from load balancer", name)
	err := RemoveNodeFromELB(a.region, cfg.Driver.(*config).InstanceID)
	if err != nil {
		return util.Errorf("failed to remove node %s (%+v) from load balancer: %v", name, cfg, err)
	}
	return nil
}
