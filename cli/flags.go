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

import "github.com/cockroachdb/cockroach-prod/base"

// initFlags sets the base/context values to flag values.
func initFlags(ctx *base.Context) {
	cobraCommand.Flags().StringVar(&ctx.Certs, "certs", ctx.Certs, "certificates directory. Generated CA and node "+
		"certs and keys are stored there.")

	cobraCommand.Flags().Int64Var(&ctx.Port, "port", ctx.Port, "cockroach node and load balancer port.")

	// Region to run in. This takes a driver attribute.
	cobraCommand.Flags().StringVar(&ctx.Region, "region", ctx.Region, "region to run in. Specify a platform driver "+
		"and region. AWS EC2: aws:us-east-1, Google Compute Engine: gce:us-central1.")

	// Driver-specific flags.
	cobraCommand.Flags().StringVar(&ctx.GCEProject, "gce-project", ctx.GCEProject, "project name for Google Compute "+
		"engine. Defaults to \"cockroach-<local username>\".")

	cobraCommand.Flags().StringVar(&ctx.GCETokenPath, "gce-auth-token", ctx.GCETokenPath, "path to the OAuth "+
		"token for Google Compute Engine.")
}

func init() {
	initFlags(Context)
}
