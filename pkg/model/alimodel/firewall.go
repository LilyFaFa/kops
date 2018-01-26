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

package alimodel

import (
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/alitasks"
)

const IpProtocolAll = "all"

// FirewallModelBuilder configures firewall network objects
type FirewallModelBuilder struct {
	*ALIModelContext
	Lifecycle *fi.Lifecycle
}

var _ fi.ModelBuilder = &FirewallModelBuilder{}

func (b *FirewallModelBuilder) Build(c *fi.ModelBuilderContext) error {

	// Create nodeInstances security group
	var nodeSecurityGroup *alitasks.SecurityGroup
	{
		groupName := b.GetNameForSecurityGroup("node")
		nodeSecurityGroup = &alitasks.SecurityGroup{
			Name:      s(groupName),
			Lifecycle: b.Lifecycle,
			VPC:       b.LinkToVPC(),
		}
		c.AddTask(nodeSecurityGroup)
	}

	// Create masterInstances security group
	var masterSecurityGroup *alitasks.SecurityGroup
	{
		groupName := b.GetNameForSecurityGroup("master")
		masterSecurityGroup = &alitasks.SecurityGroup{
			Name:      s(groupName),
			Lifecycle: b.Lifecycle,
			VPC:       b.LinkToVPC(),
		}
		c.AddTask(masterSecurityGroup)
	}

	// Create rules for nodeInstances security group
	// Allow full egress for nodes
	ipProtocolAll := IpProtocolAll
	{
		nodeSecurityGroupRules := &alitasks.SecurityGroupRule{
			Name:          s("node-egress"),
			Lifecycle:     b.Lifecycle,
			IpProtocol:    s(ipProtocolAll),
			SecurityGroup: nodeSecurityGroup,
			PortRange:     s("-1/-1"),
			In:            fi.Bool(false),
		}
		c.AddTask(nodeSecurityGroupRules)
	}

	// Allow traffic from nodes to nodes
	{
		nodeSecurityGroupRules := &alitasks.SecurityGroupRule{
			Name:          s("node-to-node"),
			Lifecycle:     b.Lifecycle,
			IpProtocol:    s(ipProtocolAll),
			SecurityGroup: nodeSecurityGroup,
			SourceGroup:   nodeSecurityGroup,
			PortRange:     s("-1/-1"),
			In:            fi.Bool(true),
		}
		c.AddTask(nodeSecurityGroupRules)
	}

	// Allow traffic from masters to nodes
	{
		nodeSecurityGroupRules := &alitasks.SecurityGroupRule{
			Name:          s("node-to-master"),
			Lifecycle:     b.Lifecycle,
			IpProtocol:    s(ipProtocolAll),
			SecurityGroup: nodeSecurityGroup,
			SourceGroup:   masterSecurityGroup,
			PortRange:     s("-1/-1"),
			In:            fi.Bool(true),
		}
		c.AddTask(nodeSecurityGroupRules)
	}

	// Allow full egress for masters
	{
		masterSecurityGroupRules := &alitasks.SecurityGroupRule{
			Name:          s("master-egress"),
			Lifecycle:     b.Lifecycle,
			IpProtocol:    s(ipProtocolAll),
			SecurityGroup: masterSecurityGroup,
			PortRange:     s("-1/-1"),
			In:            fi.Bool(false),
		}
		c.AddTask(masterSecurityGroupRules)
	}

	// Allow traffic from masters to masters
	{
		masterSecurityGroupRules := &alitasks.SecurityGroupRule{
			Name:          s("master-master"),
			Lifecycle:     b.Lifecycle,
			IpProtocol:    s(ipProtocolAll),
			SecurityGroup: masterSecurityGroup,
			SourceGroup:   masterSecurityGroup,
			PortRange:     s("-1/-1"),
			In:            fi.Bool(true),
		}
		c.AddTask(masterSecurityGroupRules)
	}

	// Allow traffic from nodes to masters
	{
		masterSecurityGroupRules := &alitasks.SecurityGroupRule{
			Name:          s("node-master"),
			Lifecycle:     b.Lifecycle,
			IpProtocol:    s(ipProtocolAll),
			SecurityGroup: masterSecurityGroup,
			SourceGroup:   nodeSecurityGroup,
			PortRange:     s("-1/-1"),
			In:            fi.Bool(true),
		}
		c.AddTask(masterSecurityGroupRules)
	}

	return nil

}
