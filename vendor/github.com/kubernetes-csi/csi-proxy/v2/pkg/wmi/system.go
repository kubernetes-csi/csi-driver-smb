//go:build windows
// +build windows

/*
Copyright 2025 The Kubernetes Authors.

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

package wmi

import (
	"fmt"
)

const (
	Win32ServiceClass = "Win32_Service"
)

var (
	BIOSSelectorList    = []string{"SerialNumber"}
	ServiceSelectorList = []string{"DisplayName", "State", "StartMode"}
)

type ServiceInterface interface {
	GetPropertyName() (string, error)
	GetPropertyDisplayName() (string, error)
	GetPropertyState() (string, error)
	GetPropertyStartMode() (string, error)
	GetDependents(scope *Scope) ([]ServiceInterface, error)
	StartService() (result uint32, err error)
	StopService() (result uint32, err error)
	Refresh(scope *Scope) error
}

// QueryBIOSElement retrieves the BIOS element.
//
// The equivalent WMI query is:
//
//	SELECT [selectors] FROM CIM_BIOSElement
//
// Refer to https://learn.microsoft.com/en-us/windows/win32/cimwin32prov/cim-bioselement
// for the WMI class definition.
func QueryBIOSElement(scope *Scope, selectorList []string) (*COMDispatchObject, error) {
	biosQuery := NewQuery("CIM_BIOSElement").Select(selectorList...)

	bios, err := QueryFirstObjectWithBuilder(scope, biosQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query BIOS element: %w", err)
	}

	return bios, nil
}

// GetBIOSSerialNumber returns the BIOS serial number.
func GetBIOSSerialNumber(bios *COMDispatchObject) (string, error) {
	return bios.GetStringProperty("SerialNumber")
}

// QueryServiceByName retrieves a specific service by its name.
//
// The equivalent WMI query is:
//
//	SELECT [selectors] FROM Win32_Service
//
// Refer to https://learn.microsoft.com/en-us/windows/win32/cimwin32prov/win32-service
// for the WMI class definition.
func QueryServiceByName(scope *Scope, name string, selectorList []string) (*COMDispatchObject, error) {
	serviceQuery := NewQuery(Win32ServiceClass).
		Select(selectorList...).
		WithCondition("Name", "=", name)

	service, err := QueryFirstObjectWithBuilder(scope, serviceQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query service %s: %w", name, err)
	}

	return service, nil
}

// GetServiceName returns the name of a service.
func GetServiceName(service ServiceInterface) (string, error) {
	return service.GetPropertyName()
}

// GetServiceDisplayName returns the display name of a service.
func GetServiceDisplayName(service ServiceInterface) (string, error) {
	return service.GetPropertyDisplayName()
}

// GetServiceState returns the state of a service.
func GetServiceState(service ServiceInterface) (string, error) {
	return service.GetPropertyState()
}

// GetServiceStartMode returns the start mode of a service.
func GetServiceStartMode(service ServiceInterface) (string, error) {
	return service.GetPropertyStartMode()
}

// Win32Service wraps the WMI class Win32_Service (mainly for testing)
type Win32Service struct {
	*COMDispatchObject
}

func (s *Win32Service) GetDependents(scope *Scope) ([]ServiceInterface, error) {
	collection, err := s.GetAssociated(scope, "Win32_DependentService", Win32ServiceClass, "Dependent", "Antecedent")
	if err != nil {
		return nil, err
	}

	var result []ServiceInterface
	err = ForEach(collection, func(coll *COMDispatchObject) error {
		result = append(result, &Win32Service{COMDispatchObject: coll})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Win32Service) GetPropertyName() (string, error) {
	return s.GetStringProperty("Name")
}

func (s *Win32Service) GetPropertyDisplayName() (string, error) {
	return s.GetStringProperty("DisplayName")
}

func (s *Win32Service) GetPropertyState() (string, error) {
	return s.GetStringProperty("State")
}

func (s *Win32Service) GetPropertyStartMode() (string, error) {
	return s.GetStringProperty("StartMode")
}

func (s *Win32Service) Refresh(scope *Scope) error {
	name, err := s.GetPropertyName()
	if err != nil {
		return err
	}

	serviceQuery := NewQuery(Win32ServiceClass).WithCondition("Name", "=", name)

	object, err := QueryFirstObjectWithBuilder(scope, serviceQuery)
	if err != nil {
		return err
	}

	s.COMDispatchObject = object
	return nil
}

func (s *Win32Service) StartService() (uint32, error) {
	return s.CallUint32("StartService")
}

func (s *Win32Service) StopService() (uint32, error) {
	return s.CallUint32("StopService")
}

type Win32ServiceFactory struct {
}

func (impl Win32ServiceFactory) GetService(scope *Scope, name string) (ServiceInterface, error) {
	service, err := QueryServiceByName(scope, name, ServiceSelectorList)
	if err != nil {
		return nil, err
	}

	return &Win32Service{COMDispatchObject: service}, nil
}
