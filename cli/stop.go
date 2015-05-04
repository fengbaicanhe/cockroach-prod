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

var stopCmd = &commander.Command{
	UsageLine: "stop",
	Short:     "stop all nodes\n",
	Long: `
Stop all nodes. This stops the actual cloud instances.
`,
	Run:  runStop,
	Flag: *flag.CommandLine,
}

func runStop(cmd *commander.Command, args []string) {
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

	// TODO(marc): only get nodes in state "Running".
	nodes, err := docker.ListCockroachNodes()
	if err != nil {
		log.Errorf("failed to get list of existing cockroach nodes: %v", err)
		return
	}
	if len(nodes) == 0 {
		log.Errorf("no existing cockroach nodes detected, this means there is probably no existing cluster")
		return
	}

	for _, nodeName := range nodes {
		// Lookup node info.
		nodeConfig, err := docker.GetMachineConfig(nodeName)
		if err != nil {
			log.Errorf("could not get node config for %s: %v", nodeName, err)
			return
		}

		// Do "stop node" logic.
		err = driver.StopNode(nodeName, nodeConfig)
		if err != nil {
			log.Errorf("could not run StopNode steps for %s: %v", nodeName, err)
			return
		}

		// Stop the machine.
		err = docker.StopMachine(nodeName)
		if err != nil {
			log.Errorf("could not stop machine %s: %v", nodeName, err)
		}
	}
}