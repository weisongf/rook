/*
Copyright 2020 The Rook Authors. All rights reserved.

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

// Package config allows a ceph config file to be stored in Kubernetes and mounted as volumes into
// Ceph daemon containers.
package config

import (
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	rookv1 "github.com/rook/rook/pkg/apis/rook.io/v1"
	v1 "k8s.io/api/core/v1"
)

type fn func(cephv1.CephClusterHealthCheckSpec) *v1.Probe

// ConfigureLivenessProbe returns the desired liveness probe for a given daemon
func ConfigureLivenessProbe(daemon rookv1.KeyType, container v1.Container, healthCheck cephv1.CephClusterHealthCheckSpec) v1.Container {
	// Map of functions
	probeFnMap := map[rookv1.KeyType]fn{
		cephv1.KeyMon: cephv1.GetMonLivenessProbe,
		cephv1.KeyMgr: cephv1.GetMgrLivenessProbe,
		cephv1.KeyOSD: cephv1.GetOSDLivenessProbe,
		cephv1.KeyMds: cephv1.GetMdsLivenessProbe,
	}

	if _, ok := healthCheck.LivenessProbe[daemon]; ok {
		if !healthCheck.LivenessProbe[daemon].Disabled {
			probe := probeFnMap[daemon](healthCheck)

			// If the spec value is empty, let's use a default
			if probe != nil {
				container.LivenessProbe = probe
			}
		} else {
			container.LivenessProbe = nil
		}
	}

	return container
}
