/*
Copyright 2016 The Kubernetes Authors.

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
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/golang/glog"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/aliup"
	"k8s.io/kops/upup/pkg/fi/cloudup/terraform"
)

//go:generate fitask -type=VPC
type VPC struct {
	Name      *string
	Lifecycle *fi.Lifecycle

	ID                 *string
	Region             *common.Region
	CIDR               *string

	Tags map[string]string
}

var _ fi.CompareWithID = &VPC{}

func (e *VPC) CompareWithID() *string {
	return e.ID
}

func (e *VPC) Find(c *fi.Context) (*VPC, error) {
	cloud := c.Cloud.(aliup.ALICloud)

	if fi.StringValue(e.ID) == "" {
		return nil, fmt.Errorf("find vpc but no id specifed")
	}

	request := &ecs.DescribeVpcsArgs{
		VpcId:    fi.StringValue(e.ID),
		RegionId: common.Region(cloud.Region()),
	}

	vpcs, _, err := cloud.EcsClient().DescribeVpcs(request)

	if err != nil {
		return nil, fmt.Errorf("error listing VPCs: %v", err)
	}

	if vpcs == nil || len(vpcs) == 0 {
		return nil, nil
	}

	if len(vpcs) != 1 {
		return nil, fmt.Errorf("found multiple VPCs for %q", fi.StringValue(e.ID))
	}

	vpc := vpcs[0]

	actual := &VPC{
		ID:     fi.String(vpc.VpcId),
		CIDR:   fi.String(vpc.CidrBlock),
		Name:   fi.String(vpc.VpcName),
		Region: &vpc.RegionId,
	}

	glog.V(4).Infof("found matching VPC %v", actual)

	if e.ID == nil {
		e.ID = actual.ID
	}
	actual.Lifecycle = e.Lifecycle

	return actual, nil
}

func (s *VPC) CheckChanges(a, e, changes *VPC) error {
	if a == nil {
		if e.CIDR == nil {
			// TODO: Auto-assign CIDR?
			return fi.RequiredField("CIDR")
		}
	}
	if a != nil {
		if changes.CIDR != nil {
			// TODO: Do we want to destroy & recreate the VPC?
			return fi.CannotChangeField("CIDR")
		}
	}
	return nil
}

func (e *VPC) Run(c *fi.Context) error {
	return fi.DefaultDeltaRunMethod(e, c)
}

func (_ *VPC) RenderALI(t *aliup.ALIAPITarget, a, e, changes *VPC) error {
	if a == nil {
		glog.V(2).Infof("Creating VPC with CIDR: %q", *e.CIDR)

		request := &ecs.CreateVpcArgs{
			RegionId:  *e.Region,
			CidrBlock: fi.StringValue(e.CIDR),
		}

		response, err := t.Cloud.EcsClient().CreateVpc(request)
		if err != nil {
			return fmt.Errorf("error creating VPC: %v", err)
		}

		e.ID = fi.String(response.VpcId)
	}
	return nil
}

type terraformVPC struct {
	CIDR *string `json:"cidr_block,omitempty"`
	Name *string `json:"name,omitempty"`
}

func (_ *VPC) RenderTerraform(t *terraform.TerraformTarget, a, e, changes *VPC) error {
	if err := t.AddOutputVariable("id", e.TerraformLink()); err != nil {
		return err
	}

	tf := &terraformVPC{
		CIDR: e.CIDR,
		Name: e.Name,
	}

	return t.RenderResource("alicloud_vpc", *e.Name, tf)
}

func (e *VPC) TerraformLink() *terraform.Literal {
	return terraform.LiteralProperty("alicloud_vpc", *e.Name, "id")
}
