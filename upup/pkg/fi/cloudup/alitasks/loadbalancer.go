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
	slb "github.com/denverdino/aliyungo/slb"
	"github.com/golang/glog"

	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/aliup"
	//"k8s.io/kops/upup/pkg/fi/cloudup/terraform"
)

// LoadBalancer represents a ALI Cloud LoadBalancer
//go:generate fitask -type=LoadBalancer

type LoadBalancer struct {
	Name                *string
	LoadbalancerId      *string
	AddressType         *string
	LoadBalancerAddress *string
	Lifecycle           *fi.Lifecycle
	Tags                map[string]string
}

var _ fi.CompareWithID = &LoadBalancer{}

func (l *LoadBalancer) CompareWithID() *string {
	return l.LoadbalancerId
}

func (l *LoadBalancer) Find(c *fi.Context) (*LoadBalancer, error) {
	cloud := c.Cloud.(aliup.ALICloud)
	//	clusterTags := cloud.GetClusterTags()
	//TODO:Get loadbalancer with LoadBalancerName, hope to support finding with tags
	describeLoadBalancersArgs := &slb.DescribeLoadBalancersArgs{
		RegionId:         common.Region(cloud.Region()),
		LoadBalancerName: fi.StringValue(l.Name),
		AddressType:      slb.AddressType(fi.StringValue(l.AddressType)),
	}
	responseLoadBalancers, err := cloud.SlbClient().DescribeLoadBalancers(describeLoadBalancersArgs)
	if err != nil {
		return nil, fmt.Errorf("error finding LoadBalancers: %v", err)
	}
	// Don't exist loadbalancer with specified ClusterTags or Name.
	if len(responseLoadBalancers) == 0 {
		return nil, nil
	}
	if len(responseLoadBalancers) > 1 {
		glog.V(4).Info("The number of specified loadbalancer whith the same name exceeds 1, loadbalancerName:%q", *l.Name)
	}

	actual := &LoadBalancer{}
	actual.Name = fi.String(responseLoadBalancers[0].LoadBalancerName)
	actual.AddressType = fi.String(string(responseLoadBalancers[0].AddressType))
	actual.LoadbalancerId = fi.String(responseLoadBalancers[0].LoadBalancerId)
	actual.LoadBalancerAddress = fi.String(responseLoadBalancers[0].Address)

	describeTagsArgs := &slb.DescribeTagsArgs{
		RegionId:       common.Region(cloud.Region()),
		LoadBalancerID: fi.StringValue(actual.LoadbalancerId),
	}
	tags, _, err := cloud.SlbClient().DescribeTags(describeTagsArgs)
	if err != nil {
		glog.V(4).Info("Error getting tags on loadbalancerID:%q", *actual.LoadbalancerId)
	}
	if len(tags) != 0 {
		for _, tag := range tags {
			key := tag.TagKey
			value := tag.TagValue
			actual.Tags[key] = value
		}
	}
	// Ignore "system" fields
	actual.Lifecycle = l.Lifecycle
	return actual, nil
}

func (l *LoadBalancer) FindIPAddress(context *fi.Context) (*string, error) {
	cloud := context.Cloud.(aliup.ALICloud)
	//	clusterTags := cloud.GetClusterTags()
	//TODO:Get loadbalancer with LoadBalancerName, hope to support finding with tags
	describeLoadBalancersArgs := &slb.DescribeLoadBalancersArgs{
		RegionId:         common.Region(cloud.Region()),
		LoadBalancerName: fi.StringValue(l.Name),
		AddressType:      slb.AddressType(fi.StringValue(l.AddressType)),
	}
	responseLoadBalancers, err := cloud.SlbClient().DescribeLoadBalancers(describeLoadBalancersArgs)
	if err != nil {
		return nil, fmt.Errorf("error finding LoadBalancers: %v", err)
	}
	// Don't exist loadbalancer with specified ClusterTags or Name.
	if len(responseLoadBalancers) == 0 {
		return nil, nil
	}
	if len(responseLoadBalancers) > 1 {
		glog.V(4).Info("The number of specified loadbalancer whith the same name exceeds 1, loadbalancerName:%q", *l.Name)
	}
	address := responseLoadBalancers[0].Address
	return &address, nil
}

func (l *LoadBalancer) Run(c *fi.Context) error {
	c.Cloud.(aliup.ALICloud).AddClusterTags(l.Tags)
	return fi.DefaultDeltaRunMethod(l, c)
}

func (_ *LoadBalancer) CheckChanges(a, e, changes *LoadBalancer) error {
	if a == nil {
		if e.Name == nil {
			return fi.RequiredField("Name")
		}
		if e.AddressType == nil {
			return fi.RequiredField("AddressType")
		}
	} else {
		if changes.AddressType != nil {
			return fi.CannotChangeField("AddressType")
		}
		if changes.Name != nil {
			return fi.CannotChangeField("Name")
		}
	}
	return nil
}

//LoadBalancer can only modify tags.
func (_ *LoadBalancer) RenderALI(t *aliup.ALIAPITarget, a, e, changes *LoadBalancer) error {
	if a == nil {
		createLoadBalancerArgs := &slb.CreateLoadBalancerArgs{
			RegionId:         common.Region(t.Cloud.Region()),
			LoadBalancerName: fi.StringValue(e.Name),
			AddressType:      slb.AddressType(fi.StringValue(e.AddressType)),
		}
		response, err := t.Cloud.SlbClient().CreateLoadBalancer(createLoadBalancerArgs)
		if err != nil {
			return fmt.Errorf("error creating loadbalancer: %v", err)
		}
		e.LoadbalancerId = fi.String(response.LoadBalancerId)
		e.LoadBalancerAddress = fi.String(response.Address)
	} else {
		e.LoadbalancerId = a.LoadbalancerId
	}

	if changes.Tags != nil {
		tagItems := e.jsonMarshalTags(e.Tags)
		addTagsArgs := &slb.AddTagsArgs{
			RegionId:       common.Region(t.Cloud.Region()),
			LoadBalancerID: fi.StringValue(e.LoadbalancerId),
			Tags:           string(tagItems),
		}
		err := t.Cloud.SlbClient().AddTags(addTagsArgs)
		if err != nil {
			return fmt.Errorf("error adding Tags to Loadbalancer: %v", err)
		}
	}

	if a != nil && (len(a.Tags) > 0) {
		tagsToDelete := e.getLoadBalancerTagsToDelete(a.Tags)
		if len(tagsToDelete) > 0 {
			tagItems := e.jsonMarshalTags(tagsToDelete)
			removeTagsArgs := &slb.RemoveTagsArgs{
				RegionId:       common.Beijing,
				LoadBalancerID: fi.StringValue(a.LoadbalancerId),
				Tags:           string(tagItems),
			}
			if err := t.Cloud.SlbClient().RemoveTags(removeTagsArgs); err != nil {
				return fmt.Errorf("error removing Tags from LoadBalancer: %v", err)
			}
		}
	}

	return nil
}

// getDiskTagsToDelete loops through the currently set tags and builds a list of tags to be deleted from the specificated disk
func (d *LoadBalancer) getLoadBalancerTagsToDelete(currentTags map[string]string) map[string]string {
	tagsToDelete := map[string]string{}
	for k, v := range currentTags {
		if _, ok := d.Tags[k]; !ok {
			tagsToDelete[k] = v
		}
	}

	return tagsToDelete
}

func (d *LoadBalancer) jsonMarshalTags(tags map[string]string) string {
	tagItemArr := []slb.TagItem{}
	tagItem := slb.TagItem{}
	for key, value := range tags {
		tagItem.TagKey = key
		tagItem.TagValue = value
		tagItemArr = append(tagItemArr, tagItem)
	}
	tagItems, _ := json.Marshal(tagItemArr)

	return string(tagItems)
}
