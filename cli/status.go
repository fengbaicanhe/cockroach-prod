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
	"fmt"
	"os"
	"os/exec"

	"github.com/cockroachdb/cockroach-prod/docker"
	"github.com/cockroachdb/cockroach/util/log"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "show status\n",
	Long: `
Show status.
`,
	Run: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) {
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
	c := exec.Command("docker-machine", "ls")
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()

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
