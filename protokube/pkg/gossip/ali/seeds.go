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

package ali

import (
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"k8s.io/kops/protokube/pkg/gossip"
)

type SeedProvider struct {
	ecs    *ecs.Client
	region string
	tag    map[string]string
}

var _ gossip.SeedProvider = &SeedProvider{}

func (p *SeedProvider) GetSeeds() ([]string, error) {
	args := &ecs.DescribeInstancesArgs{
		// TODO: pending? starting?
		Status:   ecs.Running,
		RegionId: common.Region(p.region),
		// TODO: Number limit
		Pagination: common.Pagination{
			PageNumber: 100,
			PageSize:   100,
		},
		Tag: p.tag,
	}

	var seeds []string
	resp, _, err := p.ecs.DescribeInstances(args)
	if err != nil {
		return nil, err
	}

	for _, instance := range resp {
		// TODO: Multiple IP addresses?
		for _, ip := range instance.InnerIpAddress.IpAddress {
			seeds = append(seeds, ip)
		}
	}

	return seeds, nil
}

func NewSeedProvider(c *ecs.Client, region string, tag map[string]string) (*SeedProvider, error) {
	return &SeedProvider{
		ecs:    c,
		region: region,
		tag:    tag,
	}, nil
}
