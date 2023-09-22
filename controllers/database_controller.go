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
	"sigs.k8s.io/controller-runtime/pkg/log"

	gauravkr19devv1alpha1 "github.com/gauravkr19/db-operator/api/v1alpha1"

	// "fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	// "reflect"

	// "github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	// "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	// "sigs.k8s.io/controller-runtime/pkg/controller"
	// "sigs.k8s.io/controller-runtime/pkg/handler"
	// "sigs.k8s.io/controller-runtime/pkg/manager"
	// "sigs.k8s.io/controller-runtime/pkg/reconcile"
	// "sigs.k8s.io/controller-runtime/pkg/source"
)

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=gauravkr19.dev,resources=databases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gauravkr19.dev,resources=databases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gauravkr19.dev,resources=databases/finalizers,verbs=update

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Database object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	// log := r.Log.WithValues("database", req.NamespacedName)

	// Fetch the Database custom resource
	db := &gauravkr19devv1alpha1.Database{}
	err := r.Get(ctx, req.NamespacedName, db)
	if err != nil {
		if errors.IsNotFound(err) {
			// Database resource not found. It may have been deleted.
			return ctrl.Result{}, nil
		}
		// Error fetching the Database resource
		return ctrl.Result{}, err
	}

	replicas := db.Spec.Replicas
	// Ensure that the Database resource has the necessary fields set
	// if replicas == 0 { // Check if Replicas is nil or zero
	// 	// Replicas field not specified or zero, cannot create PVC without size
	// 	return ctrl.Result{}, fmt.Errorf("Replicas not specified in Database spec")
	// }

	// Define labels for the Deployment
	deploymentLabels := map[string]string{
		"app":      "report",
		"database": "postgres",
	}

	// Define the PVC object
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name + "-pvc",
			Namespace: db.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: *db.Spec.Size,
				},
			},
		},
	}

	// Check if the PVC already exists
	foundPVC := &corev1.PersistentVolumeClaim{}
	err = r.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, foundPVC)
	if err != nil && errors.IsNotFound(err) {
		l.Info("Creating PVC", "PVC.Name", pvc.Name)
		err = r.Create(ctx, pvc)
		if err != nil {
			l.Error(err, "Failed to create PVC", "PVC.Name", pvc.Name)
			return ctrl.Result{}, err
		}
	} else if err != nil {
		l.Error(err, "Failed to check PVC", "PVC.Name", pvc.Name)
		return ctrl.Result{}, err
	}

	// Define the Deployment object
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name + "-deployment",
			Namespace: db.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: deploymentLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: deploymentLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "database-container",
							Image: db.Spec.Image, // Use the image specified in the spec
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "database-volume",
									MountPath: "/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "database-volume",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvc.Name,
								},
							},
						},
					},
				},
			},
		},
	}

	// Check if the Deployment already exists
	foundDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		l.Info("Creating Deployment", "Deployment.Name", deployment.Name)
		err = r.Create(ctx, deployment)
		if err != nil {
			l.Error(err, "Failed to create Deployment", "Deployment.Name", deployment.Name)
			return ctrl.Result{}, err
		}
	} else if err != nil {
		l.Error(err, "Failed to check Deployment", "Deployment.Name", deployment.Name)
		return ctrl.Result{}, err
	}

	// Update the Database status
	// if !reflect.DeepEqual(db.Status.PVCName, pvc.Name) {
	// 	db.Status.PVCName = pvc.Name
	// 	if err := r.Status().Update(ctx, db); err != nil {
	// 		l.Error(err, "Failed to update Database status")
	// 		return ctrl.Result{}, err
	// 	}
	// }

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gauravkr19devv1alpha1.Database{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

// func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
// 	return ctrl.NewControllerManagedBy(mgr).
// 		For(&gauravkr19devv1alpha1.Database{}).
// 		Complete(r)
// }