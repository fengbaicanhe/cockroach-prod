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

package cli

import (
	"strconv"

	"github.com/cockroachdb/cockroach-prod/docker"
	"github.com/cockroachdb/cockroach/util"
	"github.com/cockroachdb/cockroach/util/log"
	"github.com/spf13/cobra"
)

var addNodesCmd = &cobra.Command{
	Use:   "add-nodes N",
	Short: "add new nodes\n",
	Long: `
Add N new nodes to an existing cluster
`,
	Run: runAddNodes,
}

func runAddNodes(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmd.Usage()
		return
	}
	numNodes, err := strconv.Atoi(args[0])
	if err != nil || numNodes < 1 {
		log.Errorf("argument %s must be an integer > 0", args)
		return
	}

	for i := 1; i <= numNodes; i++ {
		log.Infof("adding node %d of %d", i, numNodes)
		err := AddOneNode()
		if err != nil {
			log.Errorf("problem adding node: %v", err)
			return
		}
	}
}

// AddOneNode is a helper to add a single node. Called repeatedly.
func AddOneNode() error {
	driver, err := NewDriver(Context)
	if err != nil {
		return util.Errorf("could not create driver: %v", err)
	}

	err = driver.Init()
	if err != nil {
		return util.Errorf("failed to initialized driver: %v", err)
	}

	nodes, err := docker.ListCockroachNodes()
	if err != nil {
		return util.Errorf("failed to get list of existing cockroach nodes: %v", err)
	}
	if len(nodes) == 0 {
		return util.Errorf("no existing cockroach nodes detected, this means there is probably no existing cluster")
	}

	largestIndex, err := docker.GetLargestNodeIndex(nodes)
	if err != nil {
		return util.Errorf("problem parsing existing node list: %v", err)
	}

	nodeName := docker.MakeNodeName(largestIndex + 1)

	// Create node.
	err = docker.CreateMachine(driver, nodeName)
	if err != nil {
		return util.Errorf("could not create machine %s: %v", nodeName, err)
	}

	// Lookup node info.
	nodeConfig, err := driver.GetNodeConfig(nodeName)
	if err != nil {
		return util.Errorf("could not get node config for %s: %v", nodeName, err)
	}

	// Do "new node" logic.
	err = driver.AddNode(nodeName, nodeConfig)
	if err != nil {
		return util.Errorf("could not run AddNode steps for %s: %v", nodeName, err)
	}

	// Start the cockroach node.
	err = docker.RunDockerStart(driver, nodeName, nodeConfig)
	if err != nil {
		return util.Errorf("could not initialize first cockroach node %s: %v", nodeName, err)
	}
	return nil
}
