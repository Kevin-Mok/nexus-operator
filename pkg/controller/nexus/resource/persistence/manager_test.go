// Copyright 2020 Nexus Operator and/or its authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package persistence

import (
	ctx "context"
	"fmt"
	"reflect"
	"testing"

	"github.com/m88i/nexus-operator/apis/apps/v1alpha1"
	"github.com/m88i/nexus-operator/pkg/test"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var baseNexus = &v1alpha1.Nexus{ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "nexus"}}

func TestNewManager(t *testing.T) {
	// default-setting logic is tested elsewhere
	// so here we just check if the resulting manager took in the arguments correctly
	nexus := baseNexus
	client := test.NewFakeClientBuilder().Build()
	want := &Manager{
		nexus:  nexus,
		client: client,
	}
	got := NewManager(nexus, client)
	if !reflect.DeepEqual(want, got) {
		t.Errorf("TestNewManager()\nWant: %+v\tGot: %+v", want, got)
	}
}

func TestManager_GetRequiredResources(t *testing.T) {
	// correctness of the generated resources is tested elsewhere
	// here we just want to check if they have been created and returned
	mgr := &Manager{
		nexus:  baseNexus.DeepCopy(),
		client: test.NewFakeClientBuilder().Build(),
	}

	// first, let's test without persistence
	mgr.nexus.Spec.Persistence.Persistent = false
	resources, err := mgr.GetRequiredResources()
	assert.Nil(t, err)
	// there should be no PVC without persistence
	assert.Len(t, resources, 0)

	// now, let's enable persistence
	mgr.nexus.Spec.Persistence.Persistent = true
	mgr.nexus.Spec.Persistence.VolumeSize = "10Gi"
	resources, err = mgr.GetRequiredResources()
	assert.Nil(t, err)
	// there should be a PVC with persistence
	assert.Len(t, resources, 1)
	assert.True(t, test.ContainsType(resources, reflect.TypeOf(&corev1.PersistentVolumeClaim{})))
}

func TestManager_GetDeployedResources(t *testing.T) {
	// first with no deployed resources
	fakeClient := test.NewFakeClientBuilder().Build()
	mgr := &Manager{
		nexus:  baseNexus,
		client: fakeClient,
	}
	resources, err := mgr.GetDeployedResources()
	assert.Nil(t, resources)
	assert.Len(t, resources, 0)
	assert.NoError(t, err)

	// now with a deployed PVC
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: mgr.nexus.Name, Namespace: mgr.nexus.Namespace}}
	assert.NoError(t, mgr.client.Create(ctx.TODO(), pvc))

	resources, err = mgr.GetDeployedResources()
	assert.NoError(t, err)
	assert.Len(t, resources, 1)
	assert.True(t, test.ContainsType(resources, reflect.TypeOf(&corev1.PersistentVolumeClaim{})))

	// make the client return a mocked 500 response to test errors other than NotFound
	mockErrorMsg := "mock 500"
	fakeClient.SetMockErrorForOneRequest(errors.NewInternalError(fmt.Errorf(mockErrorMsg)))
	resources, err = mgr.GetDeployedResources()
	assert.Nil(t, resources)
	assert.Contains(t, err.Error(), mockErrorMsg)
}

func TestManager_getDeployedPVC(t *testing.T) {
	mgr := &Manager{
		nexus:  baseNexus,
		client: test.NewFakeClientBuilder().Build(),
	}

	// first, test without creating the pvc
	pvc, err := mgr.getDeployedPVC()
	assert.Nil(t, pvc)
	assert.True(t, errors.IsNotFound(err))

	// now test after creating the pvc
	pvc = &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: mgr.nexus.Name, Namespace: mgr.nexus.Namespace}}
	assert.NoError(t, mgr.client.Create(ctx.TODO(), pvc))
	pvc, err = mgr.getDeployedPVC()
	assert.NotNil(t, pvc)
	assert.NoError(t, err)
}

func TestManager_GetCustomComparator(t *testing.T) {
	// the nexus and the client should have no effect on the
	// comparator functions offered by the manager
	mgr := &Manager{}

	// there is no custom comparator function for PVCs
	pvcComp := mgr.GetCustomComparator(reflect.TypeOf(&corev1.PersistentVolumeClaim{}))
	assert.Nil(t, pvcComp)
}

func TestManager_GetCustomComparators(t *testing.T) {
	// the nexus and the client should have no effect on the
	// comparator functions offered by the manager
	mgr := &Manager{}

	// there is no custom comparator function for PVCs
	comparators := mgr.GetCustomComparators()
	assert.Nil(t, comparators)
}
