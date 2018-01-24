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
	"k8s.io/kops/pkg/model"
	"k8s.io/kops/upup/pkg/fi/cloudup/alitasks"
)

type ALIModelContext struct {
	*model.KopsModelContext
}

// LinkToVPC returns the VPC object the cluster is located in
func (c *ALIModelContext) LinkToVPC(name string) *alitasks.VPC {
	return &alitasks.VPC{Name: s(name)}
}

// LinkToSecurityGroup returns the SecurityGroup with specific name
func (c *ALIModelContext) LinkToSecurityGroup(name string) *alitasks.SecurityGroup {
	return &alitasks.SecurityGroup{Name: s(name)}
}
