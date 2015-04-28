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

// The AWS library needs existing credentials.
// See "Configuring Credentials" at: https://github.com/awslabs/aws-sdk-go
package cli

import (
	"flag"
	"fmt"
	"os"

	"code.google.com/p/go-commander"

	"github.com/cockroachdb/cockroach-prod/docker"
	"github.com/cockroachdb/cockroach/util/log"
	"github.com/ghemawat/stream"
)

var statusCmd = &commander.Command{
	UsageLine: "status",
	Short:     "show status\n",
	Long: `
Show status.
`,
	Run:  runStatus,
	Flag: *flag.CommandLine,
}

func runStatus(cmd *commander.Command, args []string) {
	// Check dependencies first.
	if err := docker.CheckDockerMachine(); err != nil {
		log.Errorf("docker-machine is not properly installed: %v", err)
		return
	}
	log.Info("docker-machine binary found")

	if err := docker.CheckDocker(); err != nil {
		log.Errorf("docker is not properly installed: %v", err)
		return
	}
	log.Info("docker binary found")

	// Print docker-machine status.
	fmt.Println("######## docker-machine ########")
	err := stream.Run(
		stream.Command("docker-machine", "ls"),
		stream.WriteLines(os.Stdout),
	)
	if err != nil {
		log.Error(err)
	}

	// Get driver-specific status.
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

	fmt.Printf("\n######### %s ########\n", driver.DockerMachineDriver())
	driver.PrintStatus()
}
