/*
Copyright 2018 The Rook Authors. All rights reserved.

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

package controller

import (
	"math"
	"testing"

	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	"github.com/rook/rook/pkg/daemon/ceph/client"
	"github.com/rook/rook/pkg/operator/ceph/config"
	opconfig "github.com/rook/rook/pkg/operator/ceph/config"
	"github.com/rook/rook/pkg/operator/k8sutil"
	"github.com/rook/rook/pkg/operator/test"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestPodVolumes(t *testing.T) {
	dataPathMap := opconfig.NewDatalessDaemonDataPathMap("rook-ceph", "/var/lib/rook")

	if err := test.VolumeIsEmptyDir(k8sutil.DataDirVolume, PodVolumes(dataPathMap, "", false)); err != nil {
		t.Errorf("PodVolumes(\"\") - data dir source is not EmptyDir: %s", err.Error())
	}
	if err := test.VolumeIsHostPath(k8sutil.DataDirVolume, "/dev/sdb", PodVolumes(dataPathMap, "/dev/sdb", false)); err != nil {
		t.Errorf("PodVolumes(\"/dev/sdb\") - data dir source is not HostPath: %s", err.Error())
	}
}

func TestMountsMatchVolumes(t *testing.T) {

	dataPathMap := opconfig.NewDatalessDaemonDataPathMap("rook-ceph", "/var/lib/rook")

	volsMountsTestDef := test.VolumesAndMountsTestDefinition{
		VolumesSpec: &test.VolumesSpec{
			Moniker: "PodVolumes(\"/dev/sdc\")", Volumes: PodVolumes(dataPathMap, "/dev/sdc", false)},
		MountsSpecItems: []*test.MountsSpec{
			{Moniker: "CephVolumeMounts(true)", Mounts: CephVolumeMounts(dataPathMap, false)},
			{Moniker: "RookVolumeMounts(true)", Mounts: RookVolumeMounts(dataPathMap, false)}},
	}
	volsMountsTestDef.TestMountsMatchVolumes(t)

	volsMountsTestDef = test.VolumesAndMountsTestDefinition{
		VolumesSpec: &test.VolumesSpec{
			Moniker: "PodVolumes(\"/dev/sdc\")", Volumes: PodVolumes(dataPathMap, "/dev/sdc", true)},
		MountsSpecItems: []*test.MountsSpec{
			{Moniker: "CephVolumeMounts(false)", Mounts: CephVolumeMounts(dataPathMap, true)},
			{Moniker: "RookVolumeMounts(false)", Mounts: RookVolumeMounts(dataPathMap, true)}},
	}
	volsMountsTestDef.TestMountsMatchVolumes(t)
}

func TestCheckPodMemory(t *testing.T) {
	// This value is in MB
	const PodMinimumMemory uint64 = 1024
	name := "test"

	// A value for the memory used in the tests
	var memory_value = int64(PodMinimumMemory * 8 * uint64(math.Pow10(6)))

	// Case 1: No memory limits, no memory requested
	test_resource := v1.ResourceRequirements{}

	if err := CheckPodMemory(name, test_resource, PodMinimumMemory); err != nil {
		t.Errorf("Error case 1: %s", err.Error())
	}

	// Case 2: memory limit and memory requested
	test_resource = v1.ResourceRequirements{
		Limits: v1.ResourceList{
			v1.ResourceMemory: *resource.NewQuantity(memory_value, resource.BinarySI),
		},
		Requests: v1.ResourceList{
			v1.ResourceMemory: *resource.NewQuantity(memory_value, resource.BinarySI),
		},
	}

	if err := CheckPodMemory(name, test_resource, PodMinimumMemory); err != nil {
		t.Errorf("Error case 2: %s", err.Error())
	}

	// Only memory requested
	test_resource = v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceMemory: *resource.NewQuantity(memory_value, resource.BinarySI),
		},
	}

	if err := CheckPodMemory(name, test_resource, PodMinimumMemory); err != nil {
		t.Errorf("Error case 3: %s", err.Error())
	}
}

func TestBuildAdminSocketCommand(t *testing.T) {
	c := getDaemonConfig(config.OsdType, "")

	command := c.buildAdminSocketCommand()
	assert.Equal(t, "status", command)

	c.daemonType = config.MonType
	command = c.buildAdminSocketCommand()
	assert.Equal(t, "mon_status", command)
}

func TestBuildSocketName(t *testing.T) {
	daemonID := "0"
	c := getDaemonConfig(config.OsdType, daemonID)

	socketName := c.buildSocketName()
	assert.Equal(t, "ceph-osd.0.asok", socketName)

	c.daemonType = config.MonType
	c.daemonID = "a"
	socketName = c.buildSocketName()
	assert.Equal(t, "ceph-mon.a.asok", socketName)
}

func TestBuildSocketPath(t *testing.T) {
	daemonID := "0"
	c := getDaemonConfig(config.OsdType, daemonID)

	socketPath := c.buildSocketPath()
	assert.Equal(t, "/run/ceph/ceph-osd.0.asok", socketPath)
}

func TestGenerateLivenessProbeExecDaemon(t *testing.T) {
	daemonID := "0"
	probe := GenerateLivenessProbeExecDaemon(config.OsdType, daemonID)
	expectedCommand := []string{"env",
		"-i",
		"sh",
		"-c",
		"ceph --admin-daemon /run/ceph/ceph-osd.0.asok status",
	}

	assert.Equal(t, expectedCommand, probe.Handler.Exec.Command)
	// it's an OSD the delay must be 45
	assert.Equal(t, initialDelaySecondsOSDDaemon, probe.InitialDelaySeconds)

	// test with a mon so the delay should be 10
	probe = GenerateLivenessProbeExecDaemon(config.MonType, "a")
	assert.Equal(t, initialDelaySecondsNonOSDDaemon, probe.InitialDelaySeconds)
}

func TestDaemonFlags(t *testing.T) {
	testcases := []struct {
		label       string
		clusterInfo *client.ClusterInfo
		clusterSpec *cephv1.ClusterSpec
		daemonID    string
		expected    []string
	}{
		{
			label: "case 1: IPv6 enabled",
			clusterInfo: &client.ClusterInfo{
				FSID: "id",
			},
			clusterSpec: &cephv1.ClusterSpec{
				Network: cephv1.NetworkSpec{
					IPFamily: "IPv6",
				},
			},
			daemonID: "daemon-id",
			expected: []string{"--fsid=id", "--keyring=/etc/ceph/keyring-store/keyring", "--log-to-stderr=true", "--err-to-stderr=true",
				"--mon-cluster-log-to-stderr=true", "--log-stderr-prefix=debug ", "--default-log-to-file=false", "--default-mon-cluster-log-to-file=false",
				"--mon-host=$(ROOK_CEPH_MON_HOST)", "--mon-initial-members=$(ROOK_CEPH_MON_INITIAL_MEMBERS)", "--id=daemon-id", "--setuser=ceph", "--setgroup=ceph",
				"--ms-bind-ipv6=true"},
		},
		{
			label: "case 2: IPv6 disabled",
			clusterInfo: &client.ClusterInfo{
				FSID: "id",
			},
			clusterSpec: &cephv1.ClusterSpec{},
			daemonID:    "daemon-id",
			expected: []string{"--fsid=id", "--keyring=/etc/ceph/keyring-store/keyring", "--log-to-stderr=true", "--err-to-stderr=true",
				"--mon-cluster-log-to-stderr=true", "--log-stderr-prefix=debug ", "--default-log-to-file=false", "--default-mon-cluster-log-to-file=false",
				"--mon-host=$(ROOK_CEPH_MON_HOST)", "--mon-initial-members=$(ROOK_CEPH_MON_INITIAL_MEMBERS)", "--id=daemon-id", "--setuser=ceph", "--setgroup=ceph"},
		},
	}

	for _, tc := range testcases {
		actual := DaemonFlags(tc.clusterInfo, tc.clusterSpec, tc.daemonID)
		assert.Equalf(t, tc.expected, actual, "[%s]: failed to get correct daemonset flags", tc.label)
	}
}
