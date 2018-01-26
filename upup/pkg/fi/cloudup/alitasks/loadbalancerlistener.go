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
	"strconv"

	slb "github.com/denverdino/aliyungo/slb"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/aliup"
	//"k8s.io/kops/upup/pkg/fi/cloudup/terraform"
)

const ListenerRunningStatus = "running"

//go:generate fitask -type=LoadBalancerListener
type LoadBalancerListener struct {
	LoadBalancer      *LoadBalancer
	Name              *string
	ListenerPort      *int
	BackendServerPort *int
	Lifecycle         *fi.Lifecycle
	ListenerStatus    *string
	Bandwidth         *int
}

var _ fi.CompareWithID = &LoadBalancerListener{}

func (l *LoadBalancerListener) CompareWithID() *string {
	listenertPort := strconv.Itoa(fi.IntValue(l.ListenerPort))
	return fi.String(listenertPort)
}

func (l *LoadBalancerListener) Find(c *fi.Context) (*LoadBalancerListener, error) {
	if l.LoadBalancer == nil || l.LoadBalancer.LoadbalancerId == nil {
		return nil, fmt.Errorf("error finding LoadBalancerListener, lack of LoadBalancerId")
	}

	cloud := c.Cloud.(aliup.ALICloud)
	loadBalancerId := fi.StringValue(l.LoadBalancer.LoadbalancerId)
	listenertPort := fi.IntValue(l.ListenerPort)
	//TODO: should sort errors?
	response, err := cloud.SlbClient().DescribeLoadBalancerTCPListenerAttribute(loadBalancerId, listenertPort)
	if err != nil {
		return nil, nil
	}

	actual := &LoadBalancerListener{}
	actual.BackendServerPort = fi.Int(response.BackendServerPort)
	actual.ListenerPort = fi.Int(response.ListenerPort)
	actual.ListenerStatus = fi.String(string(response.Status))
	actual.Bandwidth = fi.Int(response.Bandwidth)
	// Ignore "system" fields
	actual.LoadBalancer = l.LoadBalancer
	actual.Lifecycle = l.Lifecycle
	return actual, nil
}

func (l *LoadBalancerListener) Run(c *fi.Context) error {
	return fi.DefaultDeltaRunMethod(l, c)
}

func (_ *LoadBalancerListener) CheckChanges(a, e, changes *LoadBalancerListener) error {
	if a == nil {
		if e.Name == nil {
			return fi.RequiredField("Name")
		}
		if e.LoadBalancer == nil || e.LoadBalancer.LoadbalancerId == nil {
			return fi.RequiredField("LoadBalnacerId")
		}
		if e.ListenerPort == nil {
			return fi.RequiredField("ListenerPort")
		}
		if e.BackendServerPort == nil {
			return fi.RequiredField("BackendServerPort")
		}
	} else {
		if changes.BackendServerPort != nil {
			return fi.CannotChangeField("BackendServerPort")
		}
		if changes.Name != nil {
			return fi.CannotChangeField("Name")
		}
	}
	return nil
}

//LoadBalancer can only modify tags.
func (_ *LoadBalancerListener) RenderALI(t *aliup.ALIAPITarget, a, e, changes *LoadBalancerListener) error {
	if e.LoadBalancer.LoadbalancerId == nil {
		return fmt.Errorf("error updating LoadBalancerListener, lack of LoadBalnacerId")
	}

	loadBalancerId := fi.StringValue(e.LoadBalancer.LoadbalancerId)
	listenertPort := fi.IntValue(e.ListenerPort)
	if a == nil {
		createLoadBalancerTCPListenerArgs := &slb.CreateLoadBalancerTCPListenerArgs{
			LoadBalancerId:    loadBalancerId,
			ListenerPort:      listenertPort,
			BackendServerPort: fi.IntValue(e.BackendServerPort),
			Bandwidth:         fi.IntValue(e.Bandwidth),
		}
		err := t.Cloud.SlbClient().CreateLoadBalancerTCPListener(createLoadBalancerTCPListenerArgs)
		if err != nil {
			return fmt.Errorf("error creating LoadBalancerlistener: %v", err)
		}
	}

	if fi.StringValue(e.ListenerStatus) == ListenerRunningStatus {
		err := t.Cloud.SlbClient().StartLoadBalancerListener(loadBalancerId, listenertPort)
		if err != nil {
			return fmt.Errorf("error starting LoadBalancerListener: %v", err)
		}
	} else {
		err := t.Cloud.SlbClient().StopLoadBalancerListener(loadBalancerId, listenertPort)
		if err != nil {
			return fmt.Errorf("error stopping LoadBalancerListener: %v", err)
		}
	}

	_, err := t.Cloud.SlbClient().WaitForListener(loadBalancerId, listenertPort, slb.TCP)
	if err != nil {
		return fmt.Errorf("error waitting LoadBalancerListener: %v", err)
	}

	return nil
}
