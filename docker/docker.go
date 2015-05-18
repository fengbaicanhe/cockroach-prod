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

package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cockroachdb/cockroach-prod/drivers"
	"github.com/cockroachdb/cockroach/util"
	"github.com/cockroachdb/cockroach/util/log"
)

const (
	dockerVersionStringPrefix = "Docker version "
)

// CheckDocker verifies that docker-machine is installed and runnable.
func CheckDocker() error {
	cmd := exec.Command("docker", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	if !strings.HasPrefix(string(out), dockerVersionStringPrefix) {
		return util.Errorf("bad output %s for docker -v, expected string prefix %q",
			out, dockerVersionStringPrefix)
	}
	return nil
}

// RunDockerInit initializes the first node.
func RunDockerInit(driver drivers.Driver, nodeName string, settings *drivers.HostConfig) error {
	dockerArgs, err := GetDockerFlags(nodeName)
	if err != nil {
		return err
	}
	args := dockerArgs
	args = append(args,
		"run",
		"--rm",
		"-v", fmt.Sprintf("%s:/data", settings.Driver.DataDir()),
		"cockroachdb/cockroach",
		"init",
		"--stores=ssd=/data",
	)
	log.Infof("running: docker %s", strings.Join(args, " "))
	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunDockerStart starts the cockroach binary.
func RunDockerStart(driver drivers.Driver, nodeName string, settings *drivers.HostConfig) error {
	dockerArgs, err := GetDockerFlags(nodeName)
	if err != nil {
		return err
	}
	port := driver.Context().Port
	args := dockerArgs
	args = append(args,
		"run",
		"-d",
		"-v", fmt.Sprintf("%s:/data", settings.Driver.DataDir()),
		"-p", fmt.Sprintf("%d:%d", port, port),
		"--net", "host",
		"cockroachdb/cockroach",
		"start",
		"--insecure",
		"--stores=ssd=/data",
		// --addr must be an address reachable by other nodes.
		fmt.Sprintf("--addr=%s:%d", settings.Driver.IPAddress(), port),
		// TODO(marc): remove localhost once we serve /_status/ before
		// joining the gossip network.
		fmt.Sprintf("--gossip=localhost:%d,http-lb=%s:%d", port, settings.Driver.GossipAddress(), port),
	)
	log.Infof("running: docker %s", strings.Join(args, " "))
	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
