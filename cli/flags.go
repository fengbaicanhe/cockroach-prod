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
	"reflect"

	"github.com/cockroachdb/cockroach-prod/base"
	"github.com/spf13/pflag"
)

var _ pflag.Value = pflagValue{}

// pflagValue wraps flag.Value and implements the extra methods of the
// pflag.Value interface.
type pflagValue struct {
	flag.Value
}

func (v pflagValue) Type() string {
	t := reflect.TypeOf(v.Value).Elem()
	return t.Kind().String()
}

func (v pflagValue) IsBoolFlag() bool {
	t := reflect.TypeOf(v.Value).Elem()
	return t.Kind() == reflect.Bool
}

// initFlags sets the server.Context values to flag values.
// Keep in sync with "server/context.go". Values in Context should be
// settable here.
// initFlags sets the base/context values to flag values.
func initFlags(ctx *base.Context) {
	// Map any flags registered in the standard "flag" package into the
	// top-level cockroach command.
	pf := cobraCommand.PersistentFlags()
	flag.VisitAll(func(f *flag.Flag) {
		pf.Var(pflagValue{f.Value}, f.Name, f.Usage)
	})

	cobraCommand.PersistentFlags().StringVar(&ctx.Certs, "certs", ctx.Certs, "certificates directory. Generated CA and node "+
		"certs and keys are stored there.")

	cobraCommand.PersistentFlags().Int64Var(&ctx.Port, "port", ctx.Port, "cockroach node and load balancer port.")

	// Region to run in. This takes a driver attribute.
	cobraCommand.PersistentFlags().StringVar(&ctx.Region, "region", ctx.Region, "region to run in. Specify a platform driver "+
		"and region. AWS EC2: aws:us-east-1, Google Compute Engine: gce:us-central1.")

	// Driver-specific flags.
	cobraCommand.PersistentFlags().StringVar(&ctx.GCEProject, "gce-project", ctx.GCEProject, "project name for Google Compute "+
		"engine. Defaults to \"cockroach-<local username>\".")

	cobraCommand.PersistentFlags().StringVar(&ctx.GCETokenPath, "gce-auth-token", ctx.GCETokenPath, "path to the OAuth "+
		"token for Google Compute Engine.")
}

func init() {
	initFlags(Context)
}
