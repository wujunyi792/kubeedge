//go:build !windows

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

// Constants used by installers
const (
	KubeEdgeDownloadURL  = "https://github.com/kubeedge/kubeedge/releases/download"
	EdgeServiceFile      = "edgecore.service"
	CloudServiceFile     = "cloudcore.service"
	ServiceFileURLFormat = "https://raw.githubusercontent.com/kubeedge/kubeedge/release-%s/build/tools/%s"
	KubeEdgePath         = "/etc/kubeedge/"
	KubeEdgeBackupPath   = "/etc/kubeedge/backup/"
	KubeEdgeUpgradePath  = "/etc/kubeedge/upgrade/"
	KubeEdgeUsrBinPath   = "/usr/local/bin"
	KubeEdgeBinaryName   = "edgecore"
	KeadmBinaryName      = "keadm"

	KubeCloudBinaryName = "cloudcore"

	KubeEdgeConfigDir        = KubeEdgePath + "config/"
	KubeEdgeCloudCoreNewYaml = KubeEdgeConfigDir + "cloudcore.yaml"
	KubeEdgeEdgeCoreNewYaml  = KubeEdgeConfigDir + "edgecore.yaml"

	KubeEdgeLogPath = "/var/log/kubeedge/"
	KubeEdgeCrdPath = KubeEdgePath + "crds"

	KubeEdgeSocketPath = "/var/lib/kubeedge/"

	EdgeRootDir = "/var/lib/edged"

	SystemdBootPath = "/run/systemd/system"

	KubeEdgeCRDDownloadURL = "https://raw.githubusercontent.com/kubeedge/kubeedge/release-%s/build/crds"

	latestReleaseVersionURL = "https://kubeedge.io/latestversion"
	RetryTimes              = 5

	APT    string = "apt"
	YUM    string = "yum"
	PACMAN string = "pacman"
)

func downloadServiceFile(componentType types.ComponentType, version semver.Version, storeDir string) error {
	// No need to download if
	// 1. the systemd not exists
	// 2. the service file already exists
	if HasSystemd() {
		var ServiceFileName string
		switch componentType {
		case types.CloudCore:
			ServiceFileName = CloudServiceFile
		case types.EdgeCore:
			ServiceFileName = EdgeServiceFile
		default:
			return fmt.Errorf("component type %s not support", componentType)
		}
		ServiceFilePath := storeDir + "/" + ServiceFileName
		strippedVersion := fmt.Sprintf("%d.%d", version.Major, version.Minor)

		// if the specified the version is greater than the latest version
		// this means we haven't released the version, this may only occur in keadm e2e test
		// in this case, we will download the latest version service file
		if latestVersion, err := GetLatestVersion(); err == nil {
			if v, err := semver.Parse(strings.TrimPrefix(latestVersion, "v")); err == nil {
				if version.GT(v) {
					strippedVersion = fmt.Sprintf("%d.%d", v.Major, v.Minor)
				}
			}
		}
		fmt.Printf("keadm will download version %s service file\n", strippedVersion)

		ServiceFileURL := fmt.Sprintf(ServiceFileURLFormat, strippedVersion, ServiceFileName)
		if _, err := os.Stat(ServiceFilePath); err != nil {
			if os.IsNotExist(err) {
				cmdStr := fmt.Sprintf("cd %s && sudo -E wget -t %d -k --no-check-certificate %s", storeDir, RetryTimes, ServiceFileURL)
				fmt.Printf("[Run as service] start to download service file for %s\n", componentType)
				if err := NewCommand(cmdStr).Exec(); err != nil {
					return err
				}
				fmt.Printf("[Run as service] success to download service file for %s\n", componentType)
			} else {
				return err
			}
		} else {
			fmt.Printf("[Run as service] service file already exisits in %s, skip download\n", ServiceFilePath)
		}
	}
	return nil
}

func retryDownload(filename, checksumFilename string, version semver.Version, tarballPath string) error {
	filePath := filepath.Join(tarballPath, filename)
	for try := 0; try < downloadRetryTimes; try++ {
		//Download the tar from repo
		dwnldURL := fmt.Sprintf("cd %s && wget -k --no-check-certificate --progress=bar:force %s/v%s/%s",
			tarballPath, KubeEdgeDownloadURL, version, filename)
		if err := NewCommand(dwnldURL).Exec(); err != nil {
			return err
		}

		//Verify the tar with checksum
		success, err := checkSum(filename, checksumFilename, version, tarballPath)
		if err != nil {
			return err
		}
		if success {
			return nil
		}
		fmt.Printf("Failed to verify the checksum of %s, try to download it again ... \n\n", filename)
		//Cleanup the downloaded files
		if err = NewCommand(fmt.Sprintf("rm -f %s", filePath)).Exec(); err != nil {
			return err
		}
	}
	return fmt.Errorf("failed to download %s", filename)
}

func checkSum(filename, checksumFilename string, version semver.Version, tarballPath string) (bool, error) {
	//Verify the tar with checksum
	fmt.Printf("%s checksum: \n", filename)

	filepath := fmt.Sprintf("%s/%s", tarballPath, filename)
	actualChecksum, err := computeSHA512Checksum(filepath)
	if err != nil {
		return false, fmt.Errorf("failed to compute checksum for %s: %v", filename, err)
	}

	fmt.Printf("%s content: \n", checksumFilename)
	checksumFilepath := fmt.Sprintf("%s/%s", tarballPath, checksumFilename)

	if _, err := os.Stat(checksumFilepath); err == nil {
		fmt.Printf("Expected or Default checksum file %s is already downloaded. \n", checksumFilename)
		content, err := os.ReadFile(checksumFilepath)
		if err != nil {
			return false, err
		}
		checksum := strings.Replace(string(content), "\n", "", -1)
		if checksum != actualChecksum {
			fmt.Printf("Failed to verify the checksum of %s ... \n\n", filename)
			return false, nil
		}
	} else {
		getDesiredCheckSum := NewCommand(fmt.Sprintf("wget -qO- %s/v%s/%s", KubeEdgeDownloadURL, version, checksumFilename))
		if err := getDesiredCheckSum.Exec(); err != nil {
			return false, err
		}

		if getDesiredCheckSum.GetStdOut() != actualChecksum {
			fmt.Printf("Failed to verify the checksum of %s ... \n\n", filename)
			return false, nil
		}
	}

	return true, nil
}

// HasSystemd checks if systemd exist.
// if command run failed, then check it by sd_booted.
func HasSystemd() bool {
	cmd := "file /sbin/init"

	if err := NewCommand(cmd).Exec(); err == nil {
		return true
	}
	// checks whether `SystemdBootPath` exists and is a directory
	// reference http://www.freedesktop.org/software/systemd/man/sd_booted.html
	fi, err := os.Lstat(SystemdBootPath)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

func isEdgeCoreServiceRunning(serviceName string) (bool, error) {
	serviceRunning := fmt.Sprintf("systemctl list-unit-files | grep enabled | grep %s ", serviceName)
	cmd := NewCommand(serviceRunning)
	err := cmd.Exec()

	if cmd.ExitCode == 0 {
		return true, nil
	} else if cmd.ExitCode == 1 {
		return false, nil
	}

	return false, err
}

// IsKubeEdgeProcessRunning checks if the given process is running or not
func IsKubeEdgeProcessRunning(proc string) (bool, error) {
	procRunning := fmt.Sprintf("pidof %s 2>&1", proc)
	cmd := NewCommand(procRunning)

	err := cmd.Exec()

	if cmd.ExitCode == 0 {
		return true, nil
	} else if cmd.ExitCode == 1 {
		return false, nil
	}

	return false, err
}

// KillKubeEdgeBinary will search for KubeEdge process and forcefully kill it
func KillKubeEdgeBinary(proc string) error {
	var binExec string
	if proc == "cloudcore" {
		binExec = fmt.Sprintf("pkill %s", proc)
	} else {
		systemdExist := HasSystemd()

		var serviceName string
		if running, err := isEdgeCoreServiceRunning("edge"); err == nil && running {
			serviceName = "edge"
		}
		if running, err := isEdgeCoreServiceRunning("edgecore"); err == nil && running {
			serviceName = "edgecore"
		}

		if systemdExist && serviceName != "" {
			// remove the system service.
			serviceFilePath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
			serviceFileRemoveExec := fmt.Sprintf("&& sudo rm %s", serviceFilePath)
			if _, err := os.Stat(serviceFilePath); err != nil && os.IsNotExist(err) {
				serviceFileRemoveExec = ""
			}
			binExec = fmt.Sprintf("sudo systemctl stop %s.service && sudo systemctl disable %s.service %s && sudo systemctl daemon-reload", serviceName, serviceName, serviceFileRemoveExec)
		} else {
			binExec = fmt.Sprintf("pkill %s", proc)
		}
	}
	cmd := NewCommand(binExec)
	if err := cmd.Exec(); err != nil {
		return err
	}
	fmt.Println(proc, "is stopped")
	return nil
}

// runEdgeCore starts edgecore with logs being captured
func runEdgeCore() error {
	// create the log dir for kubeedge
	err := os.MkdirAll(KubeEdgeLogPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgeLogPath)
	}

	systemdExist := HasSystemd()

	var binExec string
	if systemdExist {
		binExec = fmt.Sprintf("sudo ln /etc/kubeedge/%s.service /etc/systemd/system/%s.service && sudo systemctl daemon-reload && sudo systemctl enable %s && sudo systemctl start %s",
			types.EdgeCore, types.EdgeCore, types.EdgeCore, types.EdgeCore)
	} else {
		binExec = fmt.Sprintf("%s/%s > %skubeedge/edge/%s.log 2>&1 &", KubeEdgeUsrBinPath, KubeEdgeBinaryName, KubeEdgePath, KubeEdgeBinaryName)
	}

	cmd := NewCommand(binExec)
	if err := cmd.Exec(); err != nil {
		return err
	}
	fmt.Println(cmd.GetStdOut())

	if systemdExist {
		fmt.Printf("KubeEdge edgecore is running, For logs visit: journalctl -u %s.service -xe\n", types.EdgeCore)
	} else {
		fmt.Println("KubeEdge edgecore is running, For logs visit: ", KubeEdgeLogPath+KubeEdgeBinaryName+".log")
	}
	return nil
}

// installKubeEdge downloads the provided version of KubeEdge.
// Untar's in the specified location /etc/kubeedge/ and then copies
// the binary to excecutables' path (eg: /usr/local/bin)
func installKubeEdge(options types.InstallOptions, version semver.Version) error {
	// program's architecture: amd64, arm64, arm
	arch := runtime.GOARCH

	// create the storage path of the kubeedge installation packages
	if options.TarballPath == "" {
		options.TarballPath = KubeEdgePath
	} else {
		err := os.MkdirAll(options.TarballPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("not able to create %s folder path", options.TarballPath)
		}
	}

	err := os.MkdirAll(KubeEdgePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("not able to create %s folder path", KubeEdgePath)
	}

	//Check if the same version exists, then skip the download and just checksum for it
	//and if checksum failed, there will be an option to choose to continue to untar or quit.
	//checksum available at download URL. So that both can be compared to see if
	//proper download has happened and then only proceed further.
	//Currently it is missing and once checksum is in place, checksum check required
	//to be added here.
	dirname := fmt.Sprintf("kubeedge-v%s-linux-%s", version, arch)
	filename := fmt.Sprintf("kubeedge-v%s-linux-%s.tar.gz", version, arch)
	checksumFilename := fmt.Sprintf("checksum_kubeedge-v%s-linux-%s.tar.gz.txt", version, arch)
	filePath := fmt.Sprintf("%s/%s", options.TarballPath, filename)
	if _, err = os.Stat(filePath); err == nil {
		fmt.Printf("Expected or Default KubeEdge version %v is already downloaded and will checksum for it. \n", version)
		if success, _ := checkSum(filename, checksumFilename, version, options.TarballPath); !success {
			fmt.Printf("%v in your path checksum failed and do you want to delete this file and try to download again? \n", filename)
			for {
				confirm, err := askForconfirm()
				if err != nil {
					fmt.Println(err.Error())
					continue
				}
				if confirm {
					cmdStr := fmt.Sprintf("cd %s && rm -f %s", options.TarballPath, filename)
					if err := NewCommand(cmdStr).Exec(); err != nil {
						return err
					}
					fmt.Printf("%v have been deleted and will try to download again\n", filename)
					if err := retryDownload(filename, checksumFilename, version, options.TarballPath); err != nil {
						return err
					}
				} else {
					fmt.Println("failed to checksum and will continue to install.")
				}
				break
			}
		} else {
			fmt.Println("Expected or Default KubeEdge version", version, "is already downloaded")
		}
	} else if !os.IsNotExist(err) {
		return err
	} else {
		if err := retryDownload(filename, checksumFilename, version, options.TarballPath); err != nil {
			return err
		}
	}

	if err := downloadServiceFile(options.ComponentType, version, KubeEdgePath); err != nil {
		return fmt.Errorf("fail to download service file,error:{%s}", err.Error())
	}

	var untarFileAndMoveCloudCore, untarFileAndMoveEdgeCore string

	if options.ComponentType == types.CloudCore {
		untarFileAndMoveCloudCore = fmt.Sprintf("cd %s && tar -C %s -xvzf %s && cp %s/%s/cloud/cloudcore/%s %s/",
			options.TarballPath, options.TarballPath, filename, options.TarballPath, dirname, KubeCloudBinaryName, KubeEdgeUsrBinPath)

		cmd := NewCommand(untarFileAndMoveCloudCore)
		if err := cmd.Exec(); err != nil {
			return err
		}
		fmt.Println(cmd.GetStdOut())
	} else if options.ComponentType == types.EdgeCore {
		untarFileAndMoveEdgeCore = fmt.Sprintf("cd %s && tar -C %s -xvzf %s && cp %s/%s/edge/%s %s/",
			options.TarballPath, options.TarballPath, filename, options.TarballPath, dirname, KubeEdgeBinaryName, KubeEdgeUsrBinPath)
		cmd := NewCommand(untarFileAndMoveEdgeCore)
		if err := cmd.Exec(); err != nil {
			return err
		}
		fmt.Println(cmd.GetStdOut())
	}

	return nil
}

// GetOSInterface helps in returning OS specific object which implements OSTypeInstaller interface.
func GetOSInterface() types.OSTypeInstaller {
	switch GetPackageManager() {
	case APT:
		return &DebOS{}
	case YUM:
		return &RpmOS{}
	case PACMAN:
		return &PacmanOS{}
	default:
		fmt.Println("Failed to detect supported package manager command(apt, yum, pacman), exit")
		panic("Failed to detect supported package manager command(apt, yum, pacman), exit")
	}
}

// RunningModule identifies cloudcore/edgecore running or not.
func RunningModule() (types.ModuleRunning, error) {
	osType := GetOSInterface()
	cloudCoreRunning, err := osType.IsKubeEdgeProcessRunning(KubeCloudBinaryName)

	if cloudCoreRunning {
		return types.KubeEdgeCloudRunning, nil
	} else if err != nil {
		return types.NoneRunning, err
	}

	edgeCoreRunning, err := osType.IsKubeEdgeProcessRunning(KubeEdgeBinaryName)

	if edgeCoreRunning {
		return types.KubeEdgeEdgeRunning, nil
	} else if err != nil {
		return types.NoneRunning, err
	}

	return types.NoneRunning, nil
}

// RunningModuleV2 identifies cloudcore/edgecore running or not.
// only used for cloudcore container install and edgecore binary install
func RunningModuleV2(opt *types.ResetOptions) types.ModuleRunning {
	osType := GetOSInterface()
	cloudCoreRunning, err := IsCloudcoreContainerRunning(constants.SystemNamespace, opt.Kubeconfig)
	if err != nil {
		// just log the error, maybe we do not care
		klog.Warningf("failed to check cloudcore is running: %v", err)
	}
	if cloudCoreRunning {
		return types.KubeEdgeCloudRunning
	}

	edgeCoreRunning, err := osType.IsKubeEdgeProcessRunning(KubeEdgeBinaryName)
	if err != nil {
		// just log the error, maybe we do not care
		klog.Warningf("failed to check edgecore is running: %v", err)
	}
	if edgeCoreRunning {
		return types.KubeEdgeEdgeRunning
	}

	return types.NoneRunning
}

// GetPackageManager get package manager of OS
func GetPackageManager() string {
	cmd := NewCommand("command -v apt || command -v yum || command -v pacman")
	err := cmd.Exec()
	if err != nil {
		fmt.Println(err)
		return ""
	}

	if strings.HasSuffix(cmd.GetStdOut(), APT) {
		return APT
	} else if strings.HasSuffix(cmd.GetStdOut(), YUM) {
		return YUM
	} else if strings.HasSuffix(cmd.GetStdOut(), PACMAN) {
		return PACMAN
	} else {
		return ""
	}
}
