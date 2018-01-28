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

	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/aliup"
	//"k8s.io/kops/upup/pkg/fi/cloudup/terraform"
)

//go:generate fitask -type=VSwitch
type VSwitch struct {
	Name      *string
	VSwitchId *string

	Lifecycle *fi.Lifecycle
	ZoneId    *string

	CidrBlock *string
	Region    *common.Region
	VPC       *VPC
}

var _ fi.CompareWithID = &VSwitch{}

func (v *VSwitch) CompareWithID() *string {
	return v.VSwitchId
}

func (v *VSwitch) Find(c *fi.Context) (*VSwitch, error) {
	/*
		if v.VPC == nil || v.VPC.ID == nil {
			return nil, fmt.Errorf("error finding VSwitch, lack of VPCId")
		}
	*/
	if v.VPC == nil || v.VPC.ID == nil {
		glog.V(4).Infof("VPC / VPCID not found for %s, skipping Find", fi.StringValue(v.Name))
		return nil, nil
	}
	cloud := c.Cloud.(aliup.ALICloud)

	describeVSwitchesArgs := &ecs.DescribeVSwitchesArgs{
		VpcId:    fi.StringValue(v.VPC.ID),
		RegionId: common.Region(cloud.Region()),
		ZoneId:   fi.StringValue(v.ZoneId),
	}

	vswitcheList, _, err := cloud.EcsClient().DescribeVSwitches(describeVSwitchesArgs)
	if err != nil {
		return nil, fmt.Errorf("error listing VSwitchs: %v", err)
	}

	if len(vswitcheList) == 0 {
		return nil, nil
	}

	for _, vswitch := range vswitcheList {
		if vswitch.CidrBlock == fi.StringValue(v.CidrBlock) {
			actual := &VSwitch{
				Name:      fi.String(vswitch.VSwitchName),
				VSwitchId: fi.String(vswitch.VSwitchId),
				VPC: &VPC{
					ID: fi.String(vswitch.VpcId),
				},

				ZoneId:    fi.String(vswitch.ZoneId),
				CidrBlock: fi.String(vswitch.CidrBlock),
				// Ignore "system" fields
				Lifecycle: v.Lifecycle,
			}
			return actual, nil

		}
	}

	return nil, nil
}

func (s *VSwitch) CheckChanges(a, e, changes *VSwitch) error {
	if a == nil {
		if e.CidrBlock == nil {
			return fi.RequiredField("CidrBlock")
		}
		/*
			if e.VPC == nil || e.VPC.ID == nil {
				return fi.RequiredField("VPCId")
			}
		*/
		if e.ZoneId == nil {
			return fi.RequiredField("ZoneId")
		}
	} else {
		if changes.ZoneId != nil {
			return fi.CannotChangeField("ZoneId")
		}
	}
	return nil
}

func (e *VSwitch) Run(c *fi.Context) error {
	return fi.DefaultDeltaRunMethod(e, c)
}

func (_ *VSwitch) RenderALI(t *aliup.ALIAPITarget, a, e, changes *VSwitch) error {
	if e.VPC == nil || e.VPC.ID == nil {
		return fmt.Errorf("error updating VSwitch, lack of VPCId")
	}
	if a == nil {
		glog.V(2).Infof("Creating VSwitch with CIDR: %q", *e.CidrBlock)

		createVSwitchArgs := &ecs.CreateVSwitchArgs{
			ZoneId:      fi.StringValue(e.ZoneId),
			CidrBlock:   fi.StringValue(e.CidrBlock),
			VpcId:       fi.StringValue(e.VPC.ID),
			VSwitchName: fi.StringValue(e.Name),
		}

		vswitchId, err := t.Cloud.EcsClient().CreateVSwitch(createVSwitchArgs)
		if err != nil {
			return fmt.Errorf("error creating VSwitch: %v", err)
		}
		e.VSwitchId = fi.String(vswitchId)
	} else {
		e.VSwitchId = a.VSwitchId
	}

	return nil
}
