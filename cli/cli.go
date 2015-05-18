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
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/cockroachdb/cockroach-prod/base"
	"github.com/cockroachdb/cockroach-prod/drivers"
	"github.com/cockroachdb/cockroach-prod/drivers/amazon"
	"github.com/cockroachdb/cockroach-prod/drivers/google"
	"github.com/cockroachdb/cockroach/util"
	"github.com/spf13/cobra"
)

// Context contains basic configuration settings.
var Context = base.NewContext()

var listParamsCmd = &cobra.Command{
	Use:   "listparams",
	Short: "list all available parameters and their default values",
	Long: `
List all available parameters and their default values.
Note that parameter parsing stops after the first non-
option after the command name. Hence, the options need
to precede any additional arguments,

  cockroach-prod <command> [options] [arguments].`,
	Run: func(cmd *cobra.Command, args []string) {
		flag.CommandLine.PrintDefaults()
	},
}

// NewDriver creates a new driver based on the passed-in Context
// and initializes it.
// This sets up authentication and should be called before
// driver-specific docker-machine commands.
func NewDriver(context *base.Context) (drivers.Driver, error) {
	tokens := strings.SplitN(context.Region, ":", 2)
	if len(tokens) != 2 {
		return nil, util.Errorf("invalid region syntax, expected <driver>:<region name>, got: %q", context.Region)
	}

	var driver drivers.Driver

	provider := tokens[0]
	region := tokens[1]
	switch provider {
	case "aws":
		driver = amazon.NewDriver(context, region)
	case "gce":
		driver = google.NewDriver(context, region)
	default:
		return nil, util.Errorf("unknown driver: %s", driver)
	}

	err := driver.Init()
	return driver, err
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "output version information",
	Long: `
Output build version information.
`,
	Run: func(cmd *cobra.Command, args []string) {
		info := util.GetBuildInfo()
		w := &tabwriter.Writer{}
		w.Init(os.Stdout, 2, 1, 2, ' ', 0)
		fmt.Fprintf(w, "Build Vers:  %s\n", info.Vers)
		fmt.Fprintf(w, "Build Tag:   %s\n", info.Tag)
		fmt.Fprintf(w, "Build Time:  %s\n", info.Time)
		fmt.Fprintf(w, "Build Deps:\n\t%s\n",
			strings.Replace(strings.Replace(info.Deps, " ", "\n\t", -1), ":", "\t", -1))
	},
}

var cobraCommand = &cobra.Command{
	Use:   "cockroach-prod",
	Short: "cockroach deployment tool",
}

func init() {
	cobraCommand.AddCommand(
		// Cluster setup.
		initCmd,
		addNodesCmd,

		// Start and stop.
		startCmd,
		stopCmd,

		// Status commands.
		statusCmd,

		// Misc commands.
		listParamsCmd,
		versionCmd,
	)
}

// Run ...
func Run(args []string) error {
	cobraCommand.SetArgs(args)
	return cobraCommand.Execute()
}
