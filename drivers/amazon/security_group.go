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
	"github.com/awslabs/aws-sdk-go/service/ec2"
)

const (
	allIPAddresses                = "0.0.0.0/0"
	awsSecurityRuleDuplicateError = "InvalidPermission.Duplicate"
)

// AddCockroachSecurityGroupIngress takes in a nodeInfo and
// adds the cockroach port ingress rules to the security group.
// The To and From ports are set to 'cockroachPort'.
// Duplicates are technically errors according to the AWS API, but we check for
// the duplicate error code and return ok.
func AddCockroachSecurityGroupIngress(region string, cockroachPort int64, node *NodeInfo) error {
	ec2Service := ec2.New(&aws.Config{Region: region})

	_, err := ec2Service.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		CIDRIP:     aws.String(allIPAddresses),
		FromPort:   aws.Long(cockroachPort),
		ToPort:     aws.Long(cockroachPort),
		IPProtocol: aws.String(cockroachProtocol),
		GroupID:    aws.String(node.securityGroupID),
	})

	if err != nil {
		if awserr := aws.Error(err); awserr != nil && awserr.Code == awsSecurityRuleDuplicateError {
			return nil
		}
	}

	return err
}
