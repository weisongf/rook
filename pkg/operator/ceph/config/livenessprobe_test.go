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
	"reflect"
	"testing"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	rookv1 "github.com/rook/rook/pkg/apis/rook.io/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestConfigureLivenessProbe(t *testing.T) {
	keyTypes := []rookv1.KeyType{
		cephv1.KeyMds,
		cephv1.KeyMon,
		cephv1.KeyMgr,
		cephv1.KeyOSD,
	}

	for _, keyType := range keyTypes {
		configLivenessProbeHelper(t, keyType)
	}
}

func configLivenessProbeHelper(t *testing.T, keyType rookv1.KeyType) {
	p := &v1.Probe{
		Handler: v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt(8080),
			},
		},
	}
	container := v1.Container{LivenessProbe: p}
	l := map[rookv1.KeyType]*rookv1.ProbeSpec{keyType: {Disabled: true}}
	type args struct {
		daemon      rookv1.KeyType
		container   v1.Container
		healthCheck cephv1.CephClusterHealthCheckSpec
	}
	tests := []struct {
		name string
		args args
		want v1.Container
	}{
		{"probe-enabled", args{keyType, container, cephv1.CephClusterHealthCheckSpec{}}, container},
		{"probe-disabled", args{keyType, container, cephv1.CephClusterHealthCheckSpec{LivenessProbe: l}}, v1.Container{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConfigureLivenessProbe(tt.args.daemon, tt.args.container, tt.args.healthCheck); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConfigureLivenessProbe() = %v, want %v", got, tt.want)
			}
		})
	}
}
