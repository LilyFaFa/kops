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
	ecs "github.com/denverdino/aliyungo/ecs"
	"github.com/golang/glog"

	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/aliup"
	"k8s.io/kops/upup/pkg/fi/cloudup/terraform"
)

// Disk represents a ALI Cloud Disk
//go:generate fitask -type=Disk
const ResourceType = "disk"

type Disk struct {
	Lifecycle    *fi.Lifecycle
	DiskName     *string
	DiskId       *string
	ZoneId       *string
	DiskCategory *string
	Encrypted    *bool
	SizeGB       *int
	Tags         map[string]string
}

var _ fi.CompareWithID = &Disk{}

func (d *Disk) CompareWithID() *string {
	return d.DiskId
}

func (d *Disk) Find(c *fi.Context) (*Disk, error) {
	cloud := c.Cloud.(aliup.ALICloud)
	clusterTags := cloud.GetClusterTags()

	request := &ecs.DescribeDisksArgs{
		RegionId: common.Region(cloud.Region()),
		ZoneId:   fi.StringValue(d.ZoneId),
		Tag:      clusterTags,
		Name:     fi.StringValue(d.DiskName),
	}
	responseDisks, _, err := cloud.EcsClient().DescribeDisks(request)
	if err != nil {
		return nil, fmt.Errorf("error finding Disks: %v", err)
	}
	// Don't exist disk with specified ClusterTags or DiskName.
	if len(responseDisks) == 0 {
		return nil, nil
	}
	if len(responseDisks) > 1 {
		glog.V(4).Info("The number of specified disk whith the same name and ClusterTags exceeds 1, diskName:%q", *d.DiskName)
	}

	actual := &Disk{}
	actual.DiskName = &responseDisks[0].DiskName
	actual.DiskCategory = fi.String(string(responseDisks[0].Category))
	actual.ZoneId = fi.String(responseDisks[0].ZoneId)
	actual.SizeGB = fi.Int(responseDisks[0].Size)
	actual.DiskId = fi.String(responseDisks[0].DiskId)

	resourceType := ResourceType
	tags, err := cloud.GetTags(fi.StringValue(actual.DiskId), resourceType)
	if err != nil {
		glog.V(4).Info("Error getting tags on resourceId:%q", *actual.DiskId)
	}
	actual.Tags = tags

	// Ignore "system" fields
	actual.Lifecycle = d.Lifecycle
	return actual, nil
}

func (d *Disk) Run(c *fi.Context) error {
	c.Cloud.(aliup.ALICloud).AddClusterTags(d.Tags)
	return fi.DefaultDeltaRunMethod(d, c)
}

func (_ *Disk) CheckChanges(a, e, changes *Disk) error {
	if a == nil {
		if e.ZoneId == nil {
			return fi.RequiredField("ZoneId")
		}
		if e.DiskName == nil {
			return fi.RequiredField("DiskName")
		}
	} else {
		if changes.DiskId != nil {
			return fi.CannotChangeField("DiskId")
		}
		if changes.DiskCategory != nil {
			return fi.CannotChangeField("DiskCategory")
		}
		if changes.ZoneId != nil {
			return fi.CannotChangeField("ZoneId")
		}
		if changes.Encrypted != nil {
			return fi.CannotChangeField("Encrypted")
		}
	}
	return nil
}

//Disk can only modify tags.
//TODO: Whether should we allow modify size?
func (_ *Disk) RenderALI(t *aliup.ALIAPITarget, a, e, changes *Disk) error {
	if a == nil {
		request := &ecs.CreateDiskArgs{
			RegionId:     common.Region(t.Cloud.Region()),
			ZoneId:       fi.StringValue(e.ZoneId),
			Encrypted:    fi.BoolValue(e.Encrypted),
			DiskCategory: ecs.DiskCategory(fi.StringValue(e.DiskCategory)),
			Size:         fi.IntValue(e.SizeGB),
		}
		diskId, err := t.Cloud.EcsClient().CreateDisk(request)
		if err != nil {
			return fmt.Errorf("error creating disk: %v", err)
		}
		e.DiskId = fi.String(diskId)
	}

	resourceType := ResourceType
	if changes.Tags != nil {
		if err := t.Cloud.CreateTags(*e.DiskId, resourceType, e.Tags); err != nil {
			return fmt.Errorf("error adding Tags to ALI YunPan: %v", err)
		}
	}

	if a != nil && (len(a.Tags) > 0) {
		tagsToDelete := e.getDiskTagsToDelete(a.Tags)
		if len(tagsToDelete) > 0 {
			if err := t.Cloud.RemoveTags(*e.DiskId, resourceType, tagsToDelete); err != nil {
				return fmt.Errorf("error removing Tags from ALI YunPan: %v", err)
			}
		}
	}

	return nil
}

// getDiskTagsToDelete loops through the currently set tags and builds a list of tags to be deleted from the specificated disk
func (d *Disk) getDiskTagsToDelete(currentTags map[string]string) map[string]string {
	tagsToDelete := map[string]string{}
	for k, v := range currentTags {
		if _, ok := d.Tags[k]; !ok {
			tagsToDelete[k] = v
		}
	}

	return tagsToDelete
}

type terraformDisk struct {
	DiskName     *string           `json:"name"`
	DiskCategory *string           `json:"category"`
	SizeGB       *int              `json:"size"`
	Zone         *string           `json:"availability_zone"`
	Tags         map[string]string `json:"tags,omitempty"`
}

func (_ *Disk) RenderTerraform(t *terraform.TerraformTarget, a, e, changes *Disk) error {
	tf := &terraformDisk{
		DiskName:     e.DiskName,
		DiskCategory: e.DiskCategory,
		SizeGB:       e.SizeGB,
		Zone:         e.ZoneId,
		Tags:         e.Tags,
	}
	return t.RenderResource("alicloud_disk", *e.DiskName, tf)
}
