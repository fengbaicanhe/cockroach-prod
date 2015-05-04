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

	"github.com/cockroachdb/cockroach-prod/base"
)

// initFlags sets the base/context values to flag values.
func initFlags(ctx *base.Context) {
	flag.StringVar(&ctx.Certs, "certs", ctx.Certs, "certificates directory. Generated CA and node "+
		"certs and keys are stored there.")

	flag.Int64Var(&ctx.Port, "port", ctx.Port, "cockroach node and load balancer port.")

	// TODO(marc): this may take a "cloud platform" attribute (eg: aws:<region> or gce:<region>).
	// We will also need multi-region commands (eg: status, adding new region, etc...)
	flag.StringVar(&ctx.Region, "region", ctx.Region, "region to run in.")
}

func init() {
	initFlags(Context)
}
