/*
Copyright 2023.

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

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	// "sigs.k8s.io/controller-runtime/pkg/log"

	logr "github.com/go-logr/logr"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	// "k8s.io/apimachinery/pkg/api/resource"

	gauravkr19devv1alpha1 "github.com/gauravkr19/db-operator/api/v1alpha1"
)

// PVCExtensionReconciler reconciles a PVCExtension object
type PVCExtensionReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=gauravkr19.dev,resources=pvcextensions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gauravkr19.dev,resources=pvcextensions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gauravkr19.dev,resources=pvcextensions/finalizers,verbs=update

// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PVCExtension object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile

// Fetch the PVC using r.Get, modify its spec, and update it.
func (r *PVCExtensionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// log := r.Log.WithValues("pvcextension", req.NamespacedName)

	// Fetch the ExtendedPVC resource
	extendedPVC := &gauravkr19devv1alpha1.PVCExtension{}
	if err := r.Get(ctx, req.NamespacedName, extendedPVC); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Fetch the corresponding PVC
	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.Get(ctx, req.NamespacedName, pvc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the ExtendedPVC's size is different from the PVC's size
	if !extendedPVC.Spec.Size.Equal(pvc.Spec.Resources.Requests[corev1.ResourceStorage]) {
		// Update the PVC's size
		newPVC := pvc.DeepCopy()
		newPVC.Spec.Resources.Requests[corev1.ResourceStorage] = extendedPVC.Spec.Size.DeepCopy()
		if err := r.Update(ctx, newPVC); err != nil {
			return ctrl.Result{}, err
		}

		// Log the PVC extension
		r.Log.Info("PVC extended", "PVCName", newPVC.Name, "NewSize", extendedPVC.Spec.Size)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PVCExtensionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gauravkr19devv1alpha1.PVCExtension{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
