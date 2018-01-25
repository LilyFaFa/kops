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
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup/aliup"
)

//go:generate fitask -type=SSHKey
type SSHKey struct {
	Name               *string
	Lifecycle          *fi.Lifecycle
	PublicKey          *fi.ResourceHolder
	KeyPairFingerPrint *string
}

var _ fi.CompareWithID = &SSHKey{}

func (e *SSHKey) CompareWithID() *string {
	return e.Name
}

func (e *SSHKey) Find(c *fi.Context) (*SSHKey, error) {
	cloud := c.Cloud.(aliup.ALICloud)

	describeKeyPairsArgs := &ecs.DescribeKeyPairsArgs{
		RegionId:    common.Region(cloud.Region()),
		KeyPairName: fi.StringValue(e.Name),
	}
	keypairs, _, err := cloud.EcsClient().DescribeKeyPairs(describeKeyPairsArgs)

	if err != nil {
		return nil, fmt.Errorf("error listing SSHKeys: %v", err)
	}

	if len(keypairs) == 0 {
		return nil, nil
	}
	if len(keypairs) != 1 {
		return nil, fmt.Errorf("Found multiple SSHKeys with Name %q", *e.Name)
	}

	k := keypairs[0]

	actual := &SSHKey{
		Name:               fi.String(k.KeyPairName),
		KeyPairFingerPrint: fi.String(k.KeyPairFingerPrint),
	}
	// Ignore "system" fields
	actual.Lifecycle = e.Lifecycle

	return actual, nil
}

func (e *SSHKey) Run(c *fi.Context) error {
	return fi.DefaultDeltaRunMethod(e, c)
}

func (s *SSHKey) CheckChanges(a, e, changes *SSHKey) error {
	if a != nil {
		if changes.Name != nil {
			return fi.CannotChangeField("Name")
		}
	}
	return nil
}

func (_ *SSHKey) RenderAWS(t *aliup.ALIAPITarget, a, e, changes *SSHKey) error {
	if a == nil {
		importKeyPairArgs := &ecs.ImportKeyPairArgs{
			RegionId:    common.Region(t.Cloud.Region()),
			KeyPairName: fi.StringValue(e.Name),
		}

		if e.PublicKey != nil {
			d, err := e.PublicKey.AsString()
			if err != nil {
				return fmt.Errorf("error rendering SSHKey PublicKey: %v", err)
			}
			importKeyPairArgs.PublicKeyBody = d
		}

		importKeyPairResponse, err := t.Cloud.EcsClient().ImportKeyPair(importKeyPairArgs)
		if err != nil {
			return fmt.Errorf("error creating SSHKey: %v", err)
		}
		e.KeyPairFingerPrint = fi.String(importKeyPairResponse.KeyPairFingerPrint)
		return nil
	}

	return nil
}
