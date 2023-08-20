//go:build windows

package common

import (
	"fmt"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
	"strings"
)

func InstallWindowsService(exepath, serviceName, displayName, description string, param ...string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", serviceName)
	}
	var startType uint32 = mgr.StartAutomatic

	serviceType := windows.SERVICE_TYPE_ALL

	s, err = m.CreateService(serviceName, exepath, mgr.Config{
		DisplayName:      displayName,
		Description:      description,
		StartType:        startType,
		ServiceStartName: "",
		Password:         "",
		Dependencies:     nil,
		DelayedAutoStart: false,
		ServiceType:      uint32(serviceType),
	}, param...)
	if err != nil {
		return err
	}

	defer s.Close()
	err = eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		if !strings.Contains(err.Error(), "exists") {
			s.Delete()
			return fmt.Errorf("SetupEventLogSource() failed: %s", err)
		}
	}
	return nil
}

// 卸载windows服务
func UninstallWindowsService(serviceName string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", serviceName)
	}
	defer s.Close()
	err = s.Delete()
	if err != nil {
		return err
	}
	err = eventlog.Remove(serviceName)
	if err != nil {
		return fmt.Errorf("RemoveEventLogSource() failed: %s", err)
	}
	return nil
}
