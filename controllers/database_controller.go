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

	"github.com/gauravkr19/db-operator/api/v1alpha1"
	gauravkr19devv1alpha1 "github.com/gauravkr19/db-operator/api/v1alpha1"

	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	// "reflect"

	// "github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

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
					// corev1.ResourceStorage: *db.Spec.StorageSize,
					corev1.ResourceStorage: *resource.NewQuantity(int64(db.Spec.StorageSize), resource.BinarySI),
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

	// Extend PVC storage size if needed
	if db.Spec.StorageSize != db.Status.LastStorageSize {
		pvcName := db.Name + "-pvc"
		pvc := &corev1.PersistentVolumeClaim{}
		err := r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: db.Namespace}, pvc)
		if err != nil {
			l.Error(err, "Failed to get PVC", "PVC.Name", pvcName)
			return ctrl.Result{}, err
		}

		// Update the PVC storage size
		newStorageSize := db.Spec.StorageSize
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *resource.NewQuantity(int64(newStorageSize), resource.BinarySI)

		if err := r.Update(ctx, pvc); err != nil {
			l.Error(err, "Failed to update PVC", "PVC.Name", pvcName)
			return ctrl.Result{}, err
		}

		// Update the last storage size in the status
		db.Status.LastStorageSize = db.Spec.StorageSize
		if err := r.Status().Update(ctx, db); err != nil {
			l.Error(err, "Failed to update DB status")
			return ctrl.Result{}, err
		}
	}

	// Define secret
	secret := &corev1.Secret{}
	// Create secret db-connection-secret

	targetSecretName := "dbapp-connection-secret"
	clientId := "alertsnitch"
	targetSecret, err := r.defineSecret(targetSecretName, db.Namespace, "POSTGRES_PASSWORD", clientId, db)
	// Error creating replicating the secret - requeue the request.
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Get(context.TODO(), types.NamespacedName{Name: targetSecret.Name, Namespace: targetSecret.Namespace}, secret)
	secretErr := verifySecrectStatus(ctx, r, targetSecretName, targetSecret, err)
	if secretErr != nil && errors.IsNotFound(secretErr) {
		return ctrl.Result{}, secretErr
	}

	// Create service NodePort
	servPort := &corev1.Service{}
	targetServPort, err := r.defineServiceNodePort(db.Name, db.Namespace, deploymentLabels, db)

	// Error creating replicating the NodePort svc - requeue the request.
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.Get(context.TODO(), types.NamespacedName{Name: targetServPort.Name, Namespace: targetServPort.Namespace}, servPort)
	if err != nil && errors.IsNotFound(err) {
		l.Info(fmt.Sprintf("Target service port %s doesn't exist, creating it", targetServPort.Name))
		err = r.Create(context.TODO(), targetServPort)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		l.Info(fmt.Sprintf("Target service port %s exists, updating it now", targetServPort))
		err = r.Update(context.TODO(), targetServPort)
		if err != nil {
			return ctrl.Result{}, err
		}
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
							Image: db.Spec.Image,
							Env: []corev1.EnvVar{{
								Name: "POSTGRES_PASSWORD",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "dbapp-connection-secret",
										},
										Key: "POSTGRES_PASSWORD",
									},
								}},
								{Name: "POSTGRES_DB",
									Value: "alertsnitch",
								},
								{Name: "POSTGRES_USER",
									Value: "alertsnitch",
								},
								{Name: "PGDATA",
									Value: "/var/lib/postgresql/data/pgdata",
								}}, // End of Env listed values and Env definition
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "database-volume",
									MountPath: "/var/lib/postgresql/data",
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

// additional functions
// Create Service NodePort definition
func (r *DatabaseReconciler) defineServiceNodePort(name string, namespace string, deploymentLabels map[string]string, database *v1alpha1.Database) (*corev1.Service, error) {
	// Define map for the selector and labels

	var port int32 = 5432

	serv := &corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: deploymentLabels},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{{
				Port: port,
				Name: "http",
			}},
			Selector: deploymentLabels,
		},
	}

	// Used to ensure that the service will be deleted when the custom resource object is removed
	ctrl.SetControllerReference(database, serv, r.Scheme)

	return serv, nil
}

// Create Secret definition
//
//	targetSecret, err := r.defineSecret(targetSecretName, db.Namespace, "POSTGRES_PASSWORD", clientId, db)
func (r *DatabaseReconciler) defineSecret(name string, namespace string, key string, value string, database *v1alpha1.Database) (*corev1.Secret, error) {
	secret := make(map[string]string)
	secret[key] = value

	sec := &corev1.Secret{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Immutable:  new(bool),
		Data:       map[string][]byte{},
		StringData: secret,
		Type:       "Opaque",
	}

	// Used to ensure that the secret will be deleted when the custom resource object is removed
	ctrl.SetControllerReference(database, sec, r.Scheme)

	return sec, nil
}

// secretErr := verifySecrectStatus(ctx, r, targetSecretName, targetSecret, err)
func verifySecrectStatus(ctx context.Context, r *DatabaseReconciler, targetSecretName string, targetSecret *corev1.Secret, err error) error {
	l := log.FromContext(ctx)

	if err != nil && errors.IsNotFound(err) {
		l.Info(fmt.Sprintf("Target secret %s doesn't exist, creating it", targetSecretName))
		err = r.Create(context.TODO(), targetSecret)
		if err != nil {
			return err
		}
	} else {
		l.Info(fmt.Sprintf("Target secret %s exists, updating it now", targetSecretName))
		err = r.Update(context.TODO(), targetSecret)
		if err != nil {
			return err
		}
	}

	return err
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
