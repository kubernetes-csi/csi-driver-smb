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

func stageSecretsKey(volumeID, stagingPath string) string {
	return volumeID + "\x00" + stagingPath
}

func cloneSecrets(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (d *Driver) putStageSecrets(volumeID, stagingPath string, secrets map[string]string) {
	if d == nil || d.stageSecrets == nil {
		return
	}
	copied := cloneSecrets(secrets)
	if copied == nil {
		return
	}
	d.stageSecrets.Store(stageSecretsKey(volumeID, stagingPath), copied)
}

func (d *Driver) getStageSecrets(volumeID, stagingPath string) map[string]string {
	if d == nil || d.stageSecrets == nil {
		return nil
	}
	v, ok := d.stageSecrets.Load(stageSecretsKey(volumeID, stagingPath))
	if !ok {
		return nil
	}
	stored, ok := v.(map[string]string)
	if !ok {
		return nil
	}
	return cloneSecrets(stored)
}

func (d *Driver) deleteStageSecrets(volumeID, stagingPath string) {
	if d == nil || d.stageSecrets == nil {
		return
	}
	d.stageSecrets.Delete(stageSecretsKey(volumeID, stagingPath))
}
