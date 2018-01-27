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

	"github.com/golang/glog"

	common "github.com/denverdino/aliyungo/common"
	ecs "github.com/denverdino/aliyungo/ecs"

	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/aliup"
	//"k8s.io/kops/upup/pkg/fi/cloudup/terraform"
)

//go:generate fitask -type=SecurityGroup

type SecurityGroup struct {
	Name            *string
	SecurityGroupId *string
	Lifecycle       *fi.Lifecycle
	VPC             *VPC
}

var _ fi.CompareWithID = &SecurityGroup{}

func (s *SecurityGroup) CompareWithID() *string {
	return s.SecurityGroupId
}

func (s *SecurityGroup) Find(c *fi.Context) (*SecurityGroup, error) {
	/*
		if s.VPC == nil || s.VPC.ID == nil {
			return nil, fmt.Errorf("error finding LoadBalancerListener, lack of VPCId")
		}
	*/
	if s.VPC == nil || s.VPC.ID == nil {
		glog.V(4).Infof("VPC / VPCId not found for %s, skipping Find", fi.StringValue(s.Name))
		return nil, nil
	}
	cloud := c.Cloud.(aliup.ALICloud)
	describeSecurityGroupsArgs := &ecs.DescribeSecurityGroupsArgs{
		RegionId: common.Region(cloud.Region()),
		VpcId:    fi.StringValue(s.VPC.ID),
	}

	securityGroupList, _, err := cloud.EcsClient().DescribeSecurityGroups(describeSecurityGroupsArgs)
	if err != nil {
		return nil, fmt.Errorf("error finding SecurityGroups : %v", err)
	}

	// Don't exist securityGroup with specified  Name.
	if len(securityGroupList) == 0 {
		return nil, nil
	}

	actual := &SecurityGroup{}
	// Find the securityGroup match the name.
	for _, securityGroup := range securityGroupList {
		if securityGroup.SecurityGroupName == fi.StringValue(s.Name) {
			actual.Name = fi.String(securityGroup.SecurityGroupName)
			actual.SecurityGroupId = fi.String(securityGroup.SecurityGroupId)
			// Ignore "system" fields
			actual.Lifecycle = s.Lifecycle
			actual.VPC = s.VPC
			return actual, nil
		}
	}
	return nil, nil
}

func (s *SecurityGroup) Run(c *fi.Context) error {
	return fi.DefaultDeltaRunMethod(s, c)
}

func (_ *SecurityGroup) CheckChanges(a, e, changes *SecurityGroup) error {

	if a == nil {
		if e.Name == nil {
			return fi.RequiredField("Name")
		}
		/*
			if e.VPC.ID == nil {
				return fi.RequiredField("VPCId")
			}
		*/
	} else {
		if changes.Name != nil {
			return fi.CannotChangeField("Name")
		}
	}
	return nil
}

func (_ *SecurityGroup) RenderALI(t *aliup.ALIAPITarget, a, e, changes *SecurityGroup) error {
	/*
		if e.VPC == nil || e.VPC.ID == nil {
			return fmt.Errorf("error updating LoadBalancerListener, lack of VPCId")
		}
	*/
	if a == nil {
		createSecurityGroupArgs := &ecs.CreateSecurityGroupArgs{
			RegionId:          common.Region(t.Cloud.Region()),
			SecurityGroupName: fi.StringValue(e.Name),
			VpcId:             fi.StringValue(e.VPC.ID),
		}

		securityGroupId, err := t.Cloud.EcsClient().CreateSecurityGroup(createSecurityGroupArgs)
		if err != nil {
			return fmt.Errorf("error creating securityGroup: %v", err)
		}
		e.SecurityGroupId = fi.String(securityGroupId)
	} else {
		e.SecurityGroupId = a.SecurityGroupId
	}

	return nil
}
