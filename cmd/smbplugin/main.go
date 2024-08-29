/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/kubernetes-csi/csi-driver-smb/pkg/smb"
	"k8s.io/component-base/metrics/legacyregistry"
	"k8s.io/klog/v2"
)

func init() {
	klog.InitFlags(nil)
}

var (
	endpoint                      = flag.String("endpoint", "unix://tmp/csi.sock", "CSI endpoint")
	nodeID                        = flag.String("nodeid", "", "node id")
	driverName                    = flag.String("drivername", smb.DefaultDriverName, "name of the driver")
	ver                           = flag.Bool("ver", false, "Print the version and exit.")
	metricsAddress                = flag.String("metrics-address", "", "export the metrics")
	kubeconfig                    = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Required only when running out of cluster.")
	enableGetVolumeStats          = flag.Bool("enable-get-volume-stats", true, "allow GET_VOLUME_STATS on agent node")
	removeSMBMappingDuringUnmount = flag.Bool("remove-smb-mapping-during-unmount", true, "remove SMBMapping during unmount on Windows node")
	workingMountDir               = flag.String("working-mount-dir", "/tmp", "working directory for provisioner to mount smb shares temporarily")
	volStatsCacheExpireInMinutes  = flag.Int("vol-stats-cache-expire-in-minutes", 10, "The cache expire time in minutes for volume stats cache")
	krb5CacheDirectory            = flag.String("krb5-cache-directory", smb.DefaultKrb5CacheDirectory, "The directory for kerberos cache")
	krb5Prefix                    = flag.String("krb5-prefix", smb.DefaultKrb5CCName, "The prefix for kerberos cache")
	defaultOnDeletePolicy         = flag.String("default-ondelete-policy", "", "default policy for deleting subdirectory when deleting a volume")
	removeArchivedVolumePath      = flag.Bool("remove-archived-volume-path", true, "remove archived volume path in DeleteVolume")
)

func main() {
	flag.Parse()
	if *ver {
		info, err := smb.GetVersionYAML(*driverName)
		if err != nil {
			klog.Fatalln(err)
		}
		fmt.Println(info) // nolint
		os.Exit(0)
	}
	if *nodeID == "" {
		// nodeid is not needed in controller component
		klog.Warning("nodeid is empty")
	}
	exportMetrics()
	handle()
	os.Exit(0)
}

func handle() {
	driverOptions := smb.DriverOptions{
		NodeID:                        *nodeID,
		DriverName:                    *driverName,
		EnableGetVolumeStats:          *enableGetVolumeStats,
		RemoveSMBMappingDuringUnmount: *removeSMBMappingDuringUnmount,
		RemoveArchivedVolumePath:      *removeArchivedVolumePath,
		WorkingMountDir:               *workingMountDir,
		VolStatsCacheExpireInMinutes:  *volStatsCacheExpireInMinutes,
		Krb5CacheDirectory:            *krb5CacheDirectory,
		Krb5Prefix:                    *krb5Prefix,
		DefaultOnDeletePolicy:         *defaultOnDeletePolicy,
	}
	driver := smb.NewDriver(&driverOptions)
	driver.Run(*endpoint, *kubeconfig, false)
}

func exportMetrics() {
	if *metricsAddress == "" {
		return
	}
	l, err := net.Listen("tcp", *metricsAddress)
	if err != nil {
		klog.Warningf("failed to get listener for metrics endpoint: %v", err)
		return
	}
	serve(context.Background(), l, serveMetrics)
}

func serve(_ context.Context, l net.Listener, serveFunc func(net.Listener) error) {
	path := l.Addr().String()
	klog.V(2).Infof("set up prometheus server on %v", path)
	go func() {
		defer l.Close()
		if err := serveFunc(l); err != nil {
			klog.Fatalf("serve failure(%v), address(%v)", err, path)
		}
	}()
}

func serveMetrics(l net.Listener) error {
	m := http.NewServeMux()
	m.Handle("/metrics", legacyregistry.Handler())
	return trapClosedConnErr(http.Serve(l, m))
}

func trapClosedConnErr(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "use of closed network connection") {
		return nil
	}
	return err
}
