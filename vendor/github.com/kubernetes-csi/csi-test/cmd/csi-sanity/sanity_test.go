/*
Copyright 2017 Luis Pabón luis@portworx.com

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
package sanity

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/kubernetes-csi/csi-test/pkg/sanity"
)

const (
	prefix string = "csi."
)

var (
	VERSION = "(dev)"
	version bool
	config  sanity.Config
)

func init() {
	flag.StringVar(&config.Address, prefix+"endpoint", "", "CSI endpoint")
	flag.BoolVar(&version, prefix+"version", false, "Version of this program")
	flag.StringVar(&config.TargetPath, prefix+"mountdir", os.TempDir()+"/csi", "Mount point for NodePublish")
	flag.StringVar(&config.StagingPath, prefix+"stagingdir", os.TempDir()+"/csi", "Mount point for NodeStage if staging is supported")
	flag.StringVar(&config.SecretsFile, prefix+"secrets", "", "CSI secrets file")
	flag.Int64Var(&config.TestVolumeSize, prefix+"testvolumesize", sanity.DefTestVolumeSize, "Base volume size used for provisioned volumes")
	flag.StringVar(&config.TestVolumeParametersFile, prefix+"testvolumeparameters", "", "YAML file of volume parameters for provisioned volumes")
	flag.Parse()
}

func TestSanity(t *testing.T) {
	if version {
		fmt.Printf("Version = %s\n", VERSION)
		return
	}
	if len(config.Address) == 0 {
		t.Fatalf("--%sendpoint must be provided with an CSI endpoint", prefix)
	}
	sanity.Test(t, &config)
}
