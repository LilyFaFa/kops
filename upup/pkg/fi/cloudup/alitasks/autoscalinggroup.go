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
	"encoding/json"
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
	Active         *bool
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
	actual.ScalingGroupId = fi.String(groupList[0].ScalingGroupId)
	actual.Active = fi.Bool(groupList[0].LifecycleState == ess.Active)

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
	a.ScalingGroupId = actual.ScalingGroupId
	a.Active = actual.Active
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
	vswitchs := common.FlattenArray{}
	for _, vswitch := range e.VSwitchs {
		if vswitch.VSwitchId == nil {
			return fmt.Errorf("error updating autoscalingGroup, lack of VSwitchId")
		}
		vswitchs = append(vswitchs, fi.StringValue(vswitch.VSwitchId))
	}

	if a == nil {
		glog.V(2).Infof("Creating AutoscalingGroup with Name:%q", fi.StringValue(e.Name))

		createScalingGroupArgs := &ess.CreateScalingGroupArgs{
			ScalingGroupName: fi.StringValue(e.Name),
			RegionId:         common.Region(t.Cloud.Region()),
			MinSize:          e.MinSize,
			MaxSize:          e.MaxSize,
			VSwitchIds:       vswitchs,
		}

		if e.LoadBalancer != nil && e.LoadBalancer.LoadbalancerId != nil {
			loadBalancerIds := []string{fi.StringValue(e.LoadBalancer.LoadbalancerId)}
			loadBalancerId, _ := json.Marshal(loadBalancerIds)
			createScalingGroupArgs.LoadBalancerIds = string(loadBalancerId)
		}

		createScalingGroupResponse, err := t.Cloud.EssClient().CreateScalingGroup(createScalingGroupArgs)
		if err != nil {
			return fmt.Errorf("error creating autoscalingGroup: %v", err)
		}

		e.ScalingGroupId = fi.String(createScalingGroupResponse.ScalingGroupId)
		e.Active = fi.Bool(false)

	} else {
		//only support to update size
		if changes.MaxSize != nil || changes.MaxSize != nil {
			glog.V(2).Infof("Modifing AutoscalingGroup' size, GroupName:%q", fi.StringValue(e.Name))

			modifyScalingGroupArgs := &ess.ModifyScalingGroupArgs{
				ScalingGroupId: fi.StringValue(a.ScalingGroupId),
				MinSize:        e.MinSize,
				MaxSize:        e.MaxSize,
			}
			_, err := t.Cloud.EssClient().ModifyScalingGroup(modifyScalingGroupArgs)
			if err != nil {
				return fmt.Errorf("error modifing autoscalingGroup: %v", err)
			}
		}
	}

	return nil
}

/*
type terraformAutoscalingGroup struct {
	Name         *int                 `json:"scaling_group_name,omitempty"`
	MaxSize      *int                 `json:"max_size,omitempty"`
	MinSize      *int                 `json:"min_size,omitempty"`
	VSwitchs     []*terraform.Literal `json:"vswitch_ids,omitempty"`
	LoadBalancer []*terraform.Literal `json:"vswitch_ids,omitempty"`
}

func (_ *AutoscalingGroup) RenderTerraform(t *terraform.TerraformTarget, a, e, changes *AutoscalingGroup) error {
	tf := &terraformAutoscalingGroup{
		Name:    e.Name,
		MinSize: e.MinSize,
		MaxSize: e.MaxSize,
	}

	for _, s := range e.VSwitchs {
		tf.VSwitchs = append(tf.VPCZoneIdentifier, s.TerraformLink())
	}

	tags := e.buildTags(t.Cloud)
	// Make sure we output in a stable order
	var tagKeys []string
	for k := range tags {
		tagKeys = append(tagKeys, k)
	}
	sort.Strings(tagKeys)
	for _, k := range tagKeys {
		v := tags[k]
		tf.Tags = append(tf.Tags, &terraformASGTag{
			Key:               fi.String(k),
			Value:             fi.String(v),
			PropagateAtLaunch: fi.Bool(true),
		})
	}

	if e.LaunchConfiguration != nil {
		// Create TF output variable with security group ids
		// This is in the launch configuration, but the ASG has the information about the instance group type

		role := ""
		for k := range e.Tags {
			if strings.HasPrefix(k, CloudTagInstanceGroupRolePrefix) {
				suffix := strings.TrimPrefix(k, CloudTagInstanceGroupRolePrefix)
				if role != "" && role != suffix {
					return fmt.Errorf("Found multiple role tags: %q vs %q", role, suffix)
				}
				role = suffix
			}
		}

		if role != "" {
			for _, sg := range e.LaunchConfiguration.SecurityGroups {
				t.AddOutputVariableArray(role+"_security_group_ids", sg.TerraformLink())
			}
		}

		if role == "node" {
			for _, s := range e.Subnets {
				t.AddOutputVariableArray(role+"_subnet_ids", s.TerraformLink())
			}
		}
	}

	return t.RenderResource("aws_autoscaling_group", *e.Name, tf)
}
*/
