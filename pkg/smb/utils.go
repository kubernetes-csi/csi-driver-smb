/*
Copyright 2020 The Kubernetes Authors.

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
	"context"
	"fmt"
	"sync"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// lockMap used to lock on entries
type lockMap struct {
	sync.Mutex
	mutexMap map[string]*sync.Mutex
}

// NewLockMap returns a new lock map
func newLockMap() *lockMap {
	return &lockMap{
		mutexMap: make(map[string]*sync.Mutex),
	}
}

// LockEntry acquires a lock associated with the specific entry
func (lm *lockMap) LockEntry(entry string) {
	lm.Lock()
	// check if entry does not exists, then add entry
	if _, exists := lm.mutexMap[entry]; !exists {
		lm.addEntry(entry)
	}

	lm.Unlock()
	lm.lockEntry(entry)
}

// UnlockEntry release the lock associated with the specific entry
func (lm *lockMap) UnlockEntry(entry string) {
	lm.Lock()
	defer lm.Unlock()

	if _, exists := lm.mutexMap[entry]; !exists {
		return
	}
	lm.unlockEntry(entry)
}

func (lm *lockMap) addEntry(entry string) {
	lm.mutexMap[entry] = &sync.Mutex{}
}

func (lm *lockMap) lockEntry(entry string) {
	lm.mutexMap[entry].Lock()
}

func (lm *lockMap) unlockEntry(entry string) {
	lm.mutexMap[entry].Unlock()
}

func setAzureCredentials(kubeClient kubernetes.Interface, accountName, accountKey, secretNamespace string) (string, error) {
	if kubeClient == nil {
		klog.Warningf("could not create secret: kubeClient is nil")
		return "", nil
	}
	if accountName == "" || accountKey == "" {
		return "", fmt.Errorf("the account info is not enough, accountName(%v), accountKey(%v)", accountName, accountKey)
	}
	if secretNamespace == "" {
		secretNamespace = defaultSecretNamespace
	}
	secretName := fmt.Sprintf(secretNameTemplate, accountName)
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: defaultSecretNamespace,
			Name:      secretName,
		},
		Data: map[string][]byte{
			defaultSecretAccountName: []byte(accountName),
			defaultSecretAccountKey:  []byte(accountKey),
		},
		Type: "Opaque",
	}
	_, err := kubeClient.CoreV1().Secrets(secretNamespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		err = nil
	}
	if err != nil {
		return "", fmt.Errorf("couldn't create secret %v", err)
	}
	return secretName, err
}
