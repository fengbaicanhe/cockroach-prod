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
	"github.com/cockroachdb/cockroach/util"
)

// FindDefaultVPC looks for the default VPC in a given region
// and returns its ID if found.
func FindDefaultVPC(region string) (string, error) {
	ec2Service := ec2.New(&aws.Config{Region: region})

	// Call the DescribeInstances Operation
	resp, err := ec2Service.DescribeVPCs(&ec2.DescribeVPCsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("isDefault"),
				Values: []*string{aws.String("true")},
			},
		},
	})
	if err != nil {
		return "", err
	}

	if len(resp.VPCs) == 0 {
		return "", util.Errorf("no default VPC found in region %s", region)
	}
	if len(resp.VPCs) > 1 {
		return "", util.Errorf("found %d default VPCs in region %s", len(resp.VPCs), region)
	}

	return *resp.VPCs[0].VPCID, nil
}
