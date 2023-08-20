//go:build windows

/*
Copyright 2019 The KubeEdge Authors.

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

package util

import (
	types "github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/common"
	"golang.org/x/sys/windows/svc/mgr"
)

// Constants used by installers
const (
	KubeEdgeDownloadURL = "https://github.com/kubeedge/kubeedge/releases/download"
	EdgeServiceFile     = "edgecore.service"
	CloudServiceFile    = "cloudcore.service"
	KubeEdgePath        = "C:\\etc\\kubeedge\\"
	KubeEdgeBackupPath  = "C:\\etc\\kubeedge\\backup\\"
	KubeEdgeUpgradePath = "C:\\etc\\kubeedge\\upgrade\\"
	KubeEdgeUsrBinPath  = "C:\\usr\\local\\bin"
	KubeEdgeServiceName = "edgecore"
	KubeEdgeBinaryName  = "edgecore.exe"
	KeadmBinaryName     = "keadm"

	KubeEdgeConfigDir       = KubeEdgePath + "config\\"
	KubeEdgeEdgeCoreNewYaml = KubeEdgeConfigDir + "edgecore.yaml"

	KubeEdgeLogPath = "C:\\var\\log\\kubeedge\\"
	KubeEdgeCrdPath = KubeEdgePath + "crds"

	KubeEdgeSocketPath = "C:\\var\\lib\\kubeedge\\"

	EdgeRootDir = "C:\\var\\lib\\edged"

	SystemdBootPath = "C:\\run\\systemd\\system"

	KubeEdgeCRDDownloadURL = "https://raw.githubusercontent.com/kubeedge/kubeedge/release-%s/build/crds"

	latestReleaseVersionURL = "https://kubeedge.io/latestversion"
	RetryTimes              = 5
)

func IsKubeEdgeProcessRunning(service string) (bool, error) {
	m, err := mgr.Connect()
	if err != nil {
		return false, err
	}
	defer m.Disconnect()
	s, err := m.OpenService(service)
	if err != nil {
		return false, err
	}
	defer s.Close()

	return true, nil
}

func RunningModuleV2(_ *types.ResetOptions) types.ModuleRunning {
	if running, _ := IsKubeEdgeProcessRunning(KubeEdgeServiceName); running {
		return types.NoneRunning
	}
	return types.KubeEdgeEdgeRunning
}
