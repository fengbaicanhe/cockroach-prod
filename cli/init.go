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
	"flag"

	"code.google.com/p/go-commander"
	"github.com/cockroachdb/cockroach-prod/docker"
	"github.com/cockroachdb/cockroach/util/log"
)

var initCmd = &commander.Command{
	UsageLine: "init",
	Short:     "initialize a cockroach cluster",
	Long: `
Initialize a cockroach cluster. This initializes and starts the first node.
`,
	Run:  runInit,
	Flag: *flag.CommandLine,
}

func runInit(cmd *commander.Command, args []string) {
	driver, err := NewDriver(Context)
	if err != nil {
		log.Errorf("could not create driver: %v", err)
		return
	}

	err = driver.Init()
	if err != nil {
		log.Errorf("failed to initialized driver: %v", err)
		return
	}

	nodes, err := docker.ListCockroachNodes()
	if err != nil {
		log.Errorf("failed to get list of existing cockroach nodes: %v", err)
		return
	}
	if len(nodes) != 0 {
		log.Errorf("init called but docker-machine has %d existing cockroach nodes: %v", len(nodes), nodes)
		return
	}

	nodeName := docker.MakeNodeName(0)
	// Create first node.
	err = docker.CreateMachine(driver, nodeName)
	if err != nil {
		log.Errorf("could not create machine %s: %v", nodeName, err)
		return
	}

	// Lookup node info.
	nodeConfig, err := docker.GetMachineConfig(nodeName)
	if err != nil {
		log.Errorf("could not get node config for %s: %v", nodeName, err)
		return
	}

	// Run driver steps after first-node creation.
	err = driver.ProcessFirstNode(nodeName, nodeConfig)
	if err != nil {
		log.Errorf("could not run ProcessFirstNode steps for %s: %v", nodeName, err)
		return
	}

	// Do "new node" logic.
	err = driver.AddNode(nodeName, nodeConfig)
	if err != nil {
		log.Errorf("could not run AddNode steps for %s: %v", nodeName, err)
		return
	}

	// Initialize cockroach node.
	nodeDriverSettings, err := driver.GetNodeSettings(nodeName, nodeConfig)
	if err != nil {
		log.Errorf("could not determine node settings for %s: %v", nodeName, err)
		return
	}

	err = docker.RunDockerInit(Context, nodeName, nodeDriverSettings)
	if err != nil {
		log.Errorf("could not initialize first cockroach node %s: %v", nodeName, err)
		return
	}

	// Start the cockroach node.
	err = docker.RunDockerStart(Context, nodeName, nodeDriverSettings)
	if err != nil {
		log.Errorf("could not initialize first cockroach node %s: %v", nodeName, err)
	}
}
