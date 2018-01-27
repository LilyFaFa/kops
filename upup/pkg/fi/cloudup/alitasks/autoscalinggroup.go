/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package alitasks

import (
	"fmt"

	common "github.com/denverdino/aliyungo/common"
	ess "github.com/denverdino/aliyungo/ess"
	"github.com/golang/glog"

	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/aliup"
	//	"k8s.io/kops/upup/pkg/fi/cloudup/terraform"
)

//go:generate fitask -type=AutoscalingGroup

type AutoscalingGroup struct {
	Name           *string
	Lifecycle      *fi.Lifecycle
	ScalingGroupId *string
	LoadBalancer   *LoadBalancer
	VSwitchs       []*VSwitch
	MinSize        *int
	MaxSize        *int
}

var _ fi.CompareWithID = &AutoscalingGroup{}

func (a *AutoscalingGroup) CompareWithID() *string {
	return a.ScalingGroupId
}

func (a *AutoscalingGroup) Find(c *fi.Context) (*AutoscalingGroup, error) {
	cloud := c.Cloud.(aliup.ALICloud)

	describeScalingGroupsArgs := &ess.DescribeScalingGroupsArgs{
		RegionId:         common.Region(cloud.Region()),
		ScalingGroupName: common.FlattenArray{fi.StringValue(a.Name)},
	}

	groupList, _, err := cloud.EssClient().DescribeScalingGroups(describeScalingGroupsArgs)
	if err != nil {
		return nil, fmt.Errorf("error finding autoscalingGroup: %v", err)
	}

	// Don't exist autoscalingGroup with specified ClusterTags or Name.
	if len(groupList) == 0 {
		return nil, nil
	}
	if len(groupList) > 1 {
		glog.V(4).Info("The number of specified scalingGroup whith the same name and ClusterTags exceeds 1, diskName:%q", *a.Name)
	}

	actual := &AutoscalingGroup{}
	actual.Name = fi.String(groupList[0].ScalingGroupName)
	actual.MinSize = fi.Int(groupList[0].MinSize)
	actual.MaxSize = fi.Int(groupList[0].MaxSize)
	actual.LoadBalancer = &LoadBalancer{
		LoadbalancerId: fi.String(groupList[0].LoadBalancerId),
	}
	if len(groupList[0].VSwitchIds.VSwitchId) != 0 {
		for _, vswitch := range groupList[0].VSwitchIds.VSwitchId {
			v := &VSwitch{
				VSwitchId: fi.String(vswitch),
			}
			actual.VSwitchs = append(actual.VSwitchs, v)
		}
	}

	// Ignore "system" fields
	actual.Lifecycle = a.Lifecycle
	return actual, nil
}

func (a *AutoscalingGroup) Run(c *fi.Context) error {
	return fi.DefaultDeltaRunMethod(a, c)
}

func (_ *AutoscalingGroup) CheckChanges(a, e, changes *AutoscalingGroup) error {
	if a == nil {
		if e.MaxSize == nil {
			return fi.RequiredField("MaxSize")
		}
		if e.MinSize == nil {
			return fi.RequiredField("MinSize")
		}
		if e.Name == nil {
			return fi.RequiredField("Name")
		}
	}
	return nil
}

func (_ *AutoscalingGroup) RenderALI(t *aliup.ALIAPITarget, a, e, changes *AutoscalingGroup) error {
	/*
		if e.LoadBalancer == nil || e.LoadBalancer.LoadbalancerId == nil {
			return fmt.Errorf("error updating autoscalingGroup, lack of LoadBalnacerId")
		}

		if len(e.VSwitchs) == 0 {
			return fmt.Errorf("error updating autoscalingGroup, lack of VSwitch")
		}
	*/
	vswitchs := common.FlattenArray{}
	for _, vswitch := range e.VSwitchs {
		if vswitch.VSwitchId == nil {
			return fmt.Errorf("error updating autoscalingGroup, lack of VSwitchId")
		}
		vswitchs = append(vswitchs, fi.StringValue(vswitch.VSwitchId))
	}

	if a == nil {
		createScalingGroupArgs := &ess.CreateScalingGroupArgs{
			ScalingGroupName: fi.StringValue(e.Name),
			RegionId:         common.Region(t.Cloud.Region()),
			MinSize:          e.MinSize,
			MaxSize:          e.MaxSize,
			LoadBalancerIds:  fi.StringValue(e.LoadBalancer.LoadbalancerId),
			VSwitchIds:       vswitchs,
		}
		createScalingGroupResponse, err := t.Cloud.EssClient().CreateScalingGroup(createScalingGroupArgs)
		if err != nil {
			return fmt.Errorf("error creating autoscalingGroup: %v", err)
		}
		e.ScalingGroupId = fi.String(createScalingGroupResponse.ScalingGroupId)
	}

	return nil
}
