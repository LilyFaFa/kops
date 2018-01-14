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

	"k8s.io/kops/upup/pkg/fi"
)

// Disk

// JSON marshalling boilerplate
type realDisk Disk

// UnmarshalJSON implements conversion to JSON, supporitng an alternate specification of the object as a string
func (o *Disk) UnmarshalJSON(data []byte) error {
	var jsonName string
	if err := json.Unmarshal(data, &jsonName); err == nil {
		o.DiskName = &jsonName
		return nil
	}

	var r realDisk
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	*o = Disk(r)
	return nil
}

var _ fi.HasLifecycle = &Disk{}

// GetLifecycle returns the Lifecycle of the object, implementing fi.HasLifecycle
func (o *Disk) GetLifecycle() *fi.Lifecycle {
	return o.Lifecycle
}

var _ fi.HasName = &Disk{}

// GetName returns the Name of the object, implementing fi.HasName
func (o *Disk) GetName() *string {
	return o.DiskName
}

// SetName sets the Name of the object, implementing fi.SetName
func (o *Disk) SetName(name string) {
	o.DiskName = &name
}

// String is the stringer function for the task, producing readable output using fi.TaskAsString
func (o *Disk) String() string {
	return fi.TaskAsString(o)
}
