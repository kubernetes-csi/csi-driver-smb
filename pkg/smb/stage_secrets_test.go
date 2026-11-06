/*
Copyright 2026 The Kubernetes Authors.

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

package smb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStageSecretsCache(t *testing.T) {
	d := NewFakeDriver()
	volumeID := "vol_1"
	stagingPath := "/var/lib/kubelet/plugins/kubernetes.io/csi/pv/vol_1/globalmount"
	secrets := map[string]string{
		usernameField: "test_username",
		passwordField: "test_password",
		domainField:   "test_doamin",
	}

	d.putStageSecrets(volumeID, stagingPath, secrets)
	got := d.getStageSecrets(volumeID, stagingPath)
	assert.Equal(t, secrets, got)

	got["username"] = "mutated"
	secrets[usernameField] = "mutated-input"
	cached := d.getStageSecrets(volumeID, stagingPath)
	assert.Equal(t, "test_username", cached[usernameField])
	assert.NotEqual(t, secrets, cached)
	assert.NotEqual(t, got, cached)

	d.putStageSecrets(volumeID, stagingPath, nil)
	still := d.getStageSecrets(volumeID, stagingPath)
	assert.Equal(t, "test_username", still[usernameField])
	assert.Equal(t, "test_password", still[passwordField])

	d.putStageSecrets(volumeID, stagingPath, map[string]string{})
	still = d.getStageSecrets(volumeID, stagingPath)
	assert.Equal(t, "test_username", still[usernameField])

	d.deleteStageSecrets(volumeID, stagingPath)
	assert.Nil(t, d.getStageSecrets(volumeID, stagingPath))
}
