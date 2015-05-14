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

package google

import

// The package is called "resourceviews" but is in v1beta2. Specify import name for clarify.
// The documentation refers to those as "instance groups" now, but the API still has the old name.
// We follow the same naming. Our API calls mention "resource views", but our comments,
// logs, and method names say "instance groups".
(
	"time"

	"github.com/cockroachdb/cockroach/util"
	"github.com/cockroachdb/cockroach/util/log"
	resourceviews "google.golang.org/api/resourceviews/v1beta2"
)

const (
	instanceGroupName = "cockroach-group"
)

// getInstanceGroup looks for the cockroach instance group.
func (g *Google) getInstanceGroup() (*resourceviews.ResourceView, error) {
	return g.instanceGroupsService.ZoneViews.Get(g.project, g.zone, instanceGroupName).Do()
}

// createInstanceGroup creates the cockroach instance group if it does not exist.
// It returns its resource link,
func (g *Google) createInstanceGroup() (string, error) {
	if group, err := g.getInstanceGroup(); err == nil {
		log.Infof("found InstanceGroup %s: %s", instanceGroupName, group.SelfLink)
		return group.SelfLink, nil
	}

	op, err := g.instanceGroupsService.ZoneViews.Insert(g.project, g.zone,
		&resourceviews.ResourceView{
			Name: instanceGroupName,
			Endpoints: []*resourceviews.ServiceEndpoint{
				{
					Name: "http",
					Port: g.context.Port,
				},
			},
		}).Do()
	if err != nil {
		return "", err
	}
	err = g.waitForInstanceGroupOperation(op)
	if err != nil {
		return "", err
	}
	log.Infof("created InstanceGroup %s: %s", instanceGroupName, op.TargetLink)
	return op.TargetLink, nil
}

// addInstanceToGroup adds the instance (specified by resource link) to the
// cockroach instance group.
func (g *Google) addInstanceToGroup(instanceLink string) error {
	op, err := g.instanceGroupsService.ZoneViews.AddResources(g.project, g.zone, instanceGroupName,
		&resourceviews.ZoneViewsAddResourcesRequest{
			Resources: []string{instanceLink},
		}).Do()
	if err != nil {
		return err
	}
	return g.waitForInstanceGroupOperation(op)
}

// removeInstanceFromGroup removes the instance (specified by resource link) from the
// cockroach instance group.
func (g *Google) removeInstanceFromGroup(instanceLink string) error {
	op, err := g.instanceGroupsService.ZoneViews.RemoveResources(g.project, g.zone, instanceGroupName,
		&resourceviews.ZoneViewsRemoveResourcesRequest{
			Resources: []string{instanceLink},
		}).Do()
	if err != nil {
		return err
	}
	return g.waitForInstanceGroupOperation(op)
}

func errorFromInstanceGroupOperationError(opError *resourceviews.OperationError) error {
	if opError == nil {
		return nil
	}
	if len(opError.Errors) == 0 {
		return nil
	}
	return util.Errorf("operation error: %+v", opError.Errors[0])
}

// Repeatedly poll the given operation until its status is DONE, then return its Error.
// We determine whether it's a zone or global operation by parsing its resource link.
// TODO(marc): give up after a while.
func (g *Google) waitForInstanceGroupOperation(op *resourceviews.Operation) error {
	// Early out for finished ops.
	if op.Status == "DONE" {
		if log.V(1) {
			log.Infof("Operation %s %s: DONE, err=%v", op.OperationType, op.TargetLink,
				errorFromInstanceGroupOperationError(op.Error))
		}
		return errorFromInstanceGroupOperationError(op.Error)
	}

	for {
		liveOp, err := g.instanceGroupsService.ZoneOperations.Get(g.project, g.zone, op.Name).Do()
		// This usually indicates a bad operation object.
		if err != nil {
			return util.Errorf("could not lookup operation %+v: %s", op, err)
		}
		if log.V(1) {
			log.Infof("Operation %s %s: %s, err=%v", liveOp.OperationType, liveOp.TargetLink,
				liveOp.Status, errorFromInstanceGroupOperationError(liveOp.Error))
		}
		if liveOp.Status == "DONE" {
			return errorFromInstanceGroupOperationError(liveOp.Error)
		}
		time.Sleep(time.Second)
	}

	return nil
}
