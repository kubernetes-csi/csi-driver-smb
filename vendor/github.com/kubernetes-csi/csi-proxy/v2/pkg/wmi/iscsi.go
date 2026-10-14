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
	"strconv"
)

const (
	MSFTiSCSITargetPortalClass = "MSFT_iSCSITargetPortal"
	MSFTiSCSITargetClass       = "MSFT_iSCSITarget"
	MSFTiSCSISessionClass      = "MSFT_iSCSISession"
)

var (
	ISCSITargetPortalDefaultSelectorList = []string{"TargetPortalAddress", "TargetPortalPortNumber"}
)

// ListISCSITargetPortals retrieves a list of iSCSI target portals.
//
// The equivalent WMI query is:
//
//	SELECT [selectors] FROM MSFT_IscsiTargetPortal
//
// Refer to https://learn.microsoft.com/en-us/previous-versions/windows/desktop/iscsidisc/msft-iscsitargetportal
// for the WMI class definition.
func ListISCSITargetPortals(scope *Scope, selectorList []string) ([]*COMDispatchObject, error) {
	q := NewQuery(MSFTiSCSITargetPortalClass).WithNamespace(WMINamespaceStorage).Select(selectorList...)
	instances, err := QueryObjectsWithBuilder(scope, q)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

// QueryISCSITargetPortal retrieves information about a specific iSCSI target portal
// identified by its network address and port number.
//
// The equivalent WMI query is:
//
//	SELECT [selectors] FROM MSFT_IscsiTargetPortal
//	  WHERE TargetPortalAddress = '<address>'
//	    AND TargetPortalPortNumber = '<port>'
//
// Refer to https://learn.microsoft.com/en-us/previous-versions/windows/desktop/iscsidisc/msft-iscsitargetportal
// for the WMI class definition.
func QueryISCSITargetPortal(scope *Scope, address string, port uint16, selectorList []string) (*COMDispatchObject, error) {
	portalQuery := NewQuery(MSFTiSCSITargetPortalClass).
		WithNamespace(WMINamespaceStorage).
		Select(selectorList...).
		WithCondition("TargetPortalAddress", "=", address).
		WithCondition("TargetPortalPortNumber", "=", strconv.FormatUint(uint64(port), 10))

	instance, err := QueryFirstObjectWithBuilder(scope, portalQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query iSCSI target portal at (%s:%d). error: %w", address, port, err)
	}

	return instance, nil
}

// ListISCSITargetsByTargetPortalAddressAndPort retrieves ISCSI targets by address and port of an iSCSI target portal.
func ListISCSITargetsByTargetPortalAddressAndPort(scope *Scope, address string, port uint16, selectorList []string) ([]*COMDispatchObject, error) {
	instance, err := QueryISCSITargetPortal(scope, address, port, selectorList)
	if err != nil {
		return nil, err
	}

	targets, err := ListISCSITargetsByTargetPortal(scope, []*COMDispatchObject{instance})
	if err != nil {
		return nil, err
	}

	return targets, nil
}

// NewISCSITargetPortal creates a new iSCSI target portal.
//
// Refer to https://learn.microsoft.com/en-us/previous-versions/windows/desktop/iscsidisc/msft-iscsitargetportal-new
// for the WMI method definition.
func NewISCSITargetPortal(targetPortalAddress string, targetPortalPortNumber uint16, initiatorInstanceName *string, initiatorPortalAddress *string, isHeaderDigest *bool, isDataDigest *bool) error {
	params := map[string]interface{}{
		"TargetPortalAddress":    targetPortalAddress,
		"TargetPortalPortNumber": targetPortalPortNumber,
	}
	if initiatorInstanceName != nil {
		params["InitiatorInstanceName"] = *initiatorInstanceName
	}
	if initiatorPortalAddress != nil {
		params["InitiatorPortalAddress"] = *initiatorPortalAddress
	}
	if isHeaderDigest != nil {
		params["IsHeaderDigest"] = *isHeaderDigest
	}
	if isDataDigest != nil {
		params["IsDataDigest"] = *isDataDigest
	}
	result, _, err := CallMethodOnWMIClass(WMINamespaceStorage, MSFTiSCSITargetPortalClass, "New", params, DiscardOutputParameter)
	if err != nil {
		return fmt.Errorf("failed to create iSCSI target portal with %v. result: %d, error: %w", params, result, err)
	}
	if result != 0 {
		return NewWMIError(MSFTiSCSITargetPortalClass, "New", nil, result)
	}

	return nil
}

// ParseISCSITargetPortal retrieves the portal address and port number of an iSCSI target portal.
func ParseISCSITargetPortal(instance *COMDispatchObject) (string, uint16, error) {
	portalAddress, err := instance.GetStringProperty("TargetPortalAddress")
	if err != nil {
		return "", 0, fmt.Errorf("failed parsing target portal address %v. err: %w", instance, err)
	}

	portalPort, err := instance.GetUint16Property("TargetPortalPortNumber")
	if err != nil {
		return "", 0, fmt.Errorf("failed parsing target portal port number %v. err: %w", instance, err)
	}

	return portalAddress, portalPort, nil
}

// RemoveISCSITargetPortal removes an iSCSI target portal.
//
// Refer to https://learn.microsoft.com/en-us/previous-versions/windows/desktop/iscsidisc/msft-iscsitargetportal-remove
// for the WMI method definition.
func RemoveISCSITargetPortal(instance *COMDispatchObject) error {
	address, port, err := ParseISCSITargetPortal(instance)
	if err != nil {
		return fmt.Errorf("failed to parse target portal %v. error: %w", instance, err)
	}

	result, err := instance.CallUint32("Remove",
		nil,
		nil,
		int(port),
		address,
	)
	if err != nil {
		return fmt.Errorf("failed to remove iSCSI target portal %v. error: %w", instance, err)
	}
	if result != 0 {
		return NewWMIError(MSFTiSCSITargetPortalClass, "Remove", instance.Dispatch(), result)
	}
	return nil
}

// ListISCSITargetsByTargetPortal retrieves all iSCSI targets from the specified iSCSI target portal
// using MSFT_iSCSITargetToiSCSITargetPortal association.
//
// WMI association MSFT_iSCSITargetToiSCSITargetPortal:
//
//	iSCSITarget                                                                  | iSCSITargetPortal
//	-----------                                                                  | -----------------
//	MSFT_iSCSITarget (NodeAddress = "iqn.1991-05.com.microsoft:win-8e2evaq9q...) | MSFT_iSCSITargetPortal (TargetPortalAdd...
//
// Refer to https://learn.microsoft.com/en-us/previous-versions/windows/desktop/iscsidisc/msft-iscsitarget
// for the WMI class definition.
func ListISCSITargetsByTargetPortal(scope *Scope, portals []*COMDispatchObject) ([]*COMDispatchObject, error) {
	targets := make([]*COMDispatchObject, 0)
	err := ForEach(portals, func(portal *COMDispatchObject) error {
		collection, err := portal.GetAssociated(scope, "MSFT_iSCSITargetToiSCSITargetPortal", MSFTiSCSITargetClass, "iSCSITarget", "iSCSITargetPortal")
		if err != nil {
			return fmt.Errorf("failed to query associated iSCSITarget for %v. error: %w", portal, err)
		}

		targets = append(targets, collection...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return targets, nil
}

// QueryISCSITarget retrieves the iSCSI target from the specified portal address, portal and node address.
func QueryISCSITarget(scope *Scope, address string, port uint16, nodeAddress string) (*COMDispatchObject, error) {
	portal, err := QueryISCSITargetPortal(scope, address, port, nil)
	if err != nil {
		return nil, err
	}

	targets, err := ListISCSITargetsByTargetPortal(scope, []*COMDispatchObject{portal})
	if err != nil {
		return nil, err
	}

	var result *COMDispatchObject
	err = ForEach(targets, func(target *COMDispatchObject) error {
		targetNodeAddress, err := GetISCSITargetNodeAddress(target)
		if err != nil {
			return fmt.Errorf("failed to query iSCSI target %v. error: %w", target, err)
		}

		if targetNodeAddress == nodeAddress {
			result = target
			return ErrStopIteration
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrNotFound
	}
	return result, nil
}

// GetISCSITargetNodeAddress returns the node address of an iSCSI target.
func GetISCSITargetNodeAddress(target *COMDispatchObject) (string, error) {
	return target.GetStringProperty("NodeAddress")
}

// IsISCSITargetConnected returns whether the iSCSI target is connected.
func IsISCSITargetConnected(target *COMDispatchObject) (bool, error) {
	return target.GetBoolProperty("IsConnected")
}

// QueryISCSISessionByTarget retrieves the iSCSI session from the specified iSCSI target
// using MSFT_iSCSITargetToiSCSISession association.
//
// WMI association MSFT_iSCSITargetToiSCSISession:
//
//	iSCSISession                                                                | iSCSITarget
//	------------                                                                | -----------
//	MSFT_iSCSISession (SessionIdentifier = "ffffac0cacbff010-4000013700000016") | MSFT_iSCSITarget (NodeAddress = "iqn.199...
//
// Refer to https://learn.microsoft.com/en-us/previous-versions/windows/desktop/iscsidisc/msft-iscsisession
// for the WMI class definition.
func QueryISCSISessionByTarget(scope *Scope, target *COMDispatchObject) (*COMDispatchObject, error) {
	collection, err := target.GetAssociated(scope, "MSFT_iSCSITargetToiSCSISession", MSFTiSCSISessionClass, "iSCSISession", "iSCSITarget")
	if err != nil {
		return nil, fmt.Errorf("failed to query associated iSCSISession for %v. error: %w", target, err)
	}

	if len(collection) == 0 {
		return nil, nil
	}

	return collection[0], nil
}

// UnregisterISCSISession unregisters the iSCSI session so that it is no longer persistent.
//
// Refer https://learn.microsoft.com/en-us/previous-versions/windows/desktop/iscsidisc/msft-iscsisession-unregister
// for the WMI method definition.
func UnregisterISCSISession(session *COMDispatchObject) error {
	result, err := session.CallUint32("Unregister")
	if err != nil {
		return fmt.Errorf("failed to unregister iSCSI session %v. error: %w", session, err)
	}
	if result != 0 {
		return NewWMIError(MSFTiSCSISessionClass, "Unregister", session.Dispatch(), result)
	}
	return nil
}

// SetISCSISessionChapSecret sets a CHAP secret key for use with iSCSI initiator connections.
//
// Refer https://learn.microsoft.com/en-us/previous-versions/windows/desktop/iscsidisc/msft-iscsitarget-disconnect
// for the WMI method definition.
func SetISCSISessionChapSecret(mutualChapSecret string) error {
	result, _, err := CallMethodOnWMIClass(WMINamespaceStorage, MSFTiSCSISessionClass, "SetCHAPSecret", map[string]interface{}{"ChapSecret": mutualChapSecret}, DiscardOutputParameter)
	if err != nil {
		return fmt.Errorf("failed to set iSCSI session CHAP secret. error: %w", err)
	}
	if result != 0 {
		return NewWMIError(MSFTiSCSISessionClass, "SetCHAPSecret", nil, result)
	}
	return err
}

// GetISCSISessionIdentifier returns the identifier of an iSCSI session.
func GetISCSISessionIdentifier(session *COMDispatchObject) (string, error) {
	return session.GetStringProperty("SessionIdentifier")
}

// IsISCSISessionPersistent returns whether an iSCSI session is persistent.
func IsISCSISessionPersistent(session *COMDispatchObject) (bool, error) {
	return session.GetBoolProperty("IsPersistent")
}

// ListDisksByTarget find all disks associated with an iSCSITarget.
// It finds out the iSCSIConnections from MSFT_iSCSITargetToiSCSIConnection association,
// then locate MSFT_Disk objects from MSFT_iSCSIConnectionToDisk association.
//
// WMI association MSFT_iSCSITargetToiSCSIConnection:
//
//	iSCSIConnection                                                     | iSCSITarget
//	---------------                                                     | -----------
//	MSFT_iSCSIConnection (ConnectionIdentifier = "ffffac0cacbff010-15") | MSFT_iSCSITarget (NodeAddress = "iqn.1991-05.com...
//
// WMI association MSFT_iSCSIConnectionToDisk:
//
//	Disk                                                               | iSCSIConnection
//	----                                                               | ---------------
//	MSFT_Disk (ObjectId = "{1}\\WIN-8E2EVAQ9QSB\root/Microsoft/Win...) | MSFT_iSCSIConnection (ConnectionIdentifier = "fff...
//
// Refer to https://learn.microsoft.com/en-us/previous-versions/windows/desktop/iscsidisc/msft-iscsiconnection
// for the WMI class definition.
func ListDisksByTarget(scope *Scope, target *COMDispatchObject) ([]*COMDispatchObject, error) {
	// list connections to the given iSCSI target
	collection, err := target.GetAssociated(scope, "MSFT_iSCSITargetToiSCSIConnection", "MSFT_iSCSIConnection", "iSCSIConnection", "iSCSITarget")
	if err != nil {
		return nil, fmt.Errorf("failed to query associated iSCSISession for %v. error: %w", target, err)
	}

	if len(collection) == 0 {
		return nil, nil
	}

	disks := make([]*COMDispatchObject, 0)
	err = ForEach(collection, func(conn *COMDispatchObject) error {
		instances, err := conn.GetAssociated(scope, "MSFT_iSCSIConnectionToDisk", MSFTDiskClass, "Disk", "iSCSIConnection")
		if err != nil {
			return fmt.Errorf("failed to query associated disk for %v. error: %w", target, err)
		}

		disks = append(disks, instances...)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return disks, nil
}

// ConnectISCSITarget establishes a connection to an iSCSI target with optional CHAP authentication credential.
//
// Refer https://learn.microsoft.com/en-us/previous-versions/windows/desktop/iscsidisc/msft-iscsitarget-connect
// for the WMI method definition.
func ConnectISCSITarget(portalAddress string, portalPortNumber uint16, nodeAddress string, authType string, chapUsername *string, chapSecret *string) error {
	inParams := map[string]interface{}{
		"NodeAddress":            nodeAddress,
		"TargetPortalAddress":    portalAddress,
		"TargetPortalPortNumber": portalPortNumber,
		"AuthenticationType":     authType,
	}
	// InitiatorPortalAddress
	// IsDataDigest
	// IsHeaderDigest
	// ReportToPnP
	if chapUsername != nil {
		inParams["ChapUsername"] = *chapUsername
	}
	if chapSecret != nil {
		inParams["ChapSecret"] = *chapSecret
	}

	result, _, err := CallMethodOnWMIClass(WMINamespaceStorage, MSFTiSCSITargetClass, "Connect", inParams, DiscardOutputParameter)
	if err != nil {
		return fmt.Errorf("failed to connect iSCSI target %s:%d. error: %w", portalAddress, portalPortNumber, err)
	}
	if result != 0 {
		return NewWMIError(MSFTiSCSITargetClass, "Connect", nil, result)
	}
	return nil
}

// DisconnectISCSITarget disconnects the specified session between an iSCSI initiator and an iSCSI target.
//
// Refer https://learn.microsoft.com/en-us/previous-versions/windows/desktop/iscsidisc/msft-iscsitarget-disconnect
// for the WMI method definition.
func DisconnectISCSITarget(target *COMDispatchObject, sessionIdentifier string) error {
	result, err := target.CallUint32("Disconnect", sessionIdentifier)
	if err != nil {
		return fmt.Errorf("failed to disconnect iSCSI target %v. error: %w", target, err)
	}
	if result != 0 {
		return NewWMIError(MSFTiSCSITargetClass, "Disconnect", target.Dispatch(), result)
	}
	return nil
}
