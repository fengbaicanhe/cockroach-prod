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
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/elb"
	"github.com/cockroachdb/cockroach/util"
	"github.com/cockroachdb/cockroach/util/log"
)

const (
	cockroachELBName    = "cockroach-db"
	awsELBNotFoundError = "LoadBalancerNotFound"
)

// FindCockroachELB looks for the cockroach ELB in the given region
// and returns its external DNS name if found.
// If not found, err=nil and dnsName="".
func FindCockroachELB(region string) (string, error) {
	elbService := elb.New(&aws.Config{Region: region})
	elbs, err := elbService.DescribeLoadBalancers(&elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{
			aws.String(cockroachELBName),
		},
	})

	if IsAWSErrorCode(err, awsELBNotFoundError) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	if len(elbs.LoadBalancerDescriptions) == 0 {
		return "", nil
	}
	if len(elbs.LoadBalancerDescriptions) > 1 {
		return "", util.Errorf("found %d ELBs named %s", len(elbs.LoadBalancerDescriptions), cockroachELBName)
	}

	return *elbs.LoadBalancerDescriptions[0].DNSName, nil
}

// CreateCockroachELB creates a new load balancer in the given region.
// It uses the nodeInfo and cockroachPort to fill in the request.
// Returns the external DNS name of the created load balancer.
// We cannot specify health check parameters at creation time, but AWS
// uses the following defaults:
// target: TCP:instance_port
// timeout: 5s
// interval: 30s
// thresholds: unhealthy:2, heathy:10
// TODO(marc): we should call ConfigureHealthCheck
func CreateCockroachELB(region string, cockroachPort int64, zone string, securityGroupID string) (string, error) {
	elbService := elb.New(&aws.Config{Region: region})
	resp, err := elbService.CreateLoadBalancer(&elb.CreateLoadBalancerInput{
		LoadBalancerName: aws.String(cockroachELBName),
		SecurityGroups:   []*string{aws.String(securityGroupID)},
		Listeners: []*elb.Listener{
			{
				InstancePort:     aws.Long(cockroachPort),
				InstanceProtocol: aws.String(cockroachProtocol),
				LoadBalancerPort: aws.Long(cockroachPort),
				Protocol:         aws.String(cockroachProtocol),
			},
		},
		AvailabilityZones: []*string{
			aws.String(region + zone),
		},
	})
	return *resp.DNSName, err
}

// FindOrCreateLoadBalancer looks for the cockroach load balancer
// and creates it if it does not exist.
// Returns the external DNS name of the load balancer.
func FindOrCreateLoadBalancer(region string, cockroachPort int64, zone string,
	securityGroupID string) (string, error) {
	log.Infof("looking for load balancer")
	dnsName, err := FindCockroachELB(region)
	if err != nil {
		return "", util.Errorf("failed to lookup existing load balancer: %v", err)
	}

	if dnsName != "" {
		log.Info("found load balancer")
		return dnsName, nil
	}

	log.Infof("no existing load balancer, creating one")
	dnsName, err = CreateCockroachELB(region, cockroachPort, zone, securityGroupID)
	if err != nil {
		return "", util.Errorf("failed to create load balancer: %v", err)
	}
	log.Info("created load balancer")
	return dnsName, nil
}

// AddNodeToELB adds the specified node to the cockroach load balancer.
// This can only succeed if the cockroach ELB exists.
func AddNodeToELB(region string, instanceID string) error {
	elbService := elb.New(&aws.Config{Region: region})
	_, err := elbService.RegisterInstancesWithLoadBalancer(&elb.RegisterInstancesWithLoadBalancerInput{
		LoadBalancerName: aws.String(cockroachELBName),
		Instances:        []*elb.Instance{{InstanceID: aws.String(instanceID)}},
	})
	return err
}

// RemoveNodeFromELB removes the specified node from the cockroach load balancer.
// This can only succeed if the cockroach ELB exists.
func RemoveNodeFromELB(region string, instanceID string) error {
	elbService := elb.New(&aws.Config{Region: region})
	_, err := elbService.DeregisterInstancesFromLoadBalancer(&elb.DeregisterInstancesFromLoadBalancerInput{
		LoadBalancerName: aws.String(cockroachELBName),
		Instances:        []*elb.Instance{{InstanceID: aws.String(instanceID)}},
	})
	return err
}
