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

	"github.com/cockroachdb/cockroach-prod/base"
	"github.com/cockroachdb/cockroach-prod/drivers"
	"github.com/cockroachdb/cockroach/util"
	"github.com/cockroachdb/cockroach/util/log"
	"github.com/ghemawat/stream"
)

const (
	dockerVersionStringPrefix = "Docker version "
)

// CheckDocker verifies that docker-machine is installed and runnable.
func CheckDocker() error {
	out, err := stream.Contents(stream.Command("docker", "-v"))
	if err != nil {
		return err
	}
	if out == nil || len(out) != 1 {
		return util.Errorf("bad output %v for docker -v, expected string prefix %q",
			out, dockerVersionStringPrefix)
	}
	if !strings.HasPrefix(out[0], dockerVersionStringPrefix) {
		return util.Errorf("bad output %v for docker -v, expected string prefix %q",
			out, dockerVersionStringPrefix)
	}
	return nil
}

// RunDockerInit initializes the first node.
func RunDockerInit(context *base.Context, nodeName string, settings drivers.NodeSettings) error {
	dockerArgs, err := GetDockerFlags(nodeName)
	if err != nil {
		return err
	}
	args := dockerArgs
	args = append(args,
		"run",
		"--rm",
		"-v", fmt.Sprintf("%s:/data", settings.DataDir()),
		"cockroachdb/cockroach",
		"init",
		"-insecure",
		"-stores", "ssd=/data",
	)
	log.Infof("running: docker %s", strings.Join(args, " "))
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunDockerStart starts the cockroach binary.
func RunDockerStart(context *base.Context, nodeName string, settings drivers.NodeSettings) error {
	dockerArgs, err := GetDockerFlags(nodeName)
	if err != nil {
		return err
	}
	args := dockerArgs
	args = append(args,
		"run",
		"-d",
		"--rm",
		"-v", fmt.Sprintf("%s:/data", settings.DataDir()),
		"-p", fmt.Sprintf("%d:%d", context.Port, context.Port),
		"--net", "host",
		"cockroachdb/cockroach",
		"start",
		"-insecure",
		"-stores", "ssd=/data",
		"-addr", fmt.Sprintf("%s:%d", settings.IPAddress(), context.Port),
		"-gossip", fmt.Sprintf("%s:%d", settings.GossipAddress(), context.Port),
	)
	log.Infof("running: docker %s", strings.Join(args, " "))
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
