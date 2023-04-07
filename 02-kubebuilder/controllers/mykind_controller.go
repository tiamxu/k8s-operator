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
	"fmt"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	appsv1 "github.com/tiamxu/k8s-operator/02-kubebuilder/api/v1"
)

// MyKindReconciler reconciles a MyKind object
type MyKindReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=apps.gopron.online,resources=mykinds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.gopron.online,resources=mykinds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.gopron.online,resources=mykinds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyKind object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *MyKindReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx = context.Background()
	log := r.Log.WithValues("mykind", req.NamespacedName)
	log.Info("fetching MyKind resource", "ns", req.Namespace)
	myKind := appsv1.MyKind{}
	//check mykind resource is or not exists.
	if err := r.Get(ctx, req.NamespacedName, &myKind); err != nil {
		log.Error(err, "failed to get MyKind resource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	fmt.Printf("NamespacedName: %v,myKind: %v\n", req.NamespacedName, req.Name)
	// if err := r.cleanupOwnedResources(ctx, log, &myKind); err != nil {
	// 	log.Error(err, "failed to clean up old Deployment resource for this MyKind...")
	// 	return ctrl.Result{}, err

	// }
	// log = log.WithValues("deployment_name", myKind.Spec.DeploymentName)
	// log.Info("checking if an existing Deployment exists for this resource")

	deployment := apps.Deployment{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: myKind.Spec.DeploymentName}, &deployment)
	fmt.Printf("#####deployment:%v######\n", deployment)
	// IsNotFound returns true if the specified error was created by NewNotFound.
	// It supports wrapped errors and returns false when the error is nil.
	if apierrors.IsNotFound(err) {
		log.Info("could not find existing Deployment for MyKind, create one...")
		deployment = *buildDeployment(myKind)
		if err := r.Client.Create(ctx, &deployment); err != nil {
			log.Error(err, "failed to create Deployment resource")
			return ctrl.Result{}, err

		}
		r.Recorder.Eventf(&myKind, core.EventTypeNormal, "Created", "Created deployment %q", deployment.Name)
		log.Info("created Depoyment resource for MyKind")
		return ctrl.Result{}, err
	}
	if err != nil {
		log.Error(err, "failed to get Deployment for MyKind resource.")
		return ctrl.Result{}, err
	}
	log.Info("existing Deployment resource already exists for MyKind ,checking replica coount")
	expectedReplicas := int32(1)

	if myKind.Spec.Replicas != nil {
		expectedReplicas = *myKind.Spec.Replicas
	}
	if *deployment.Spec.Replicas != expectedReplicas {
		log.Info("update replica count", "old_count", *deployment.Spec.Replicas, "new_count", expectedReplicas)
		deployment.Spec.Replicas = &expectedReplicas
		if err := r.Client.Update(ctx, &deployment); err != nil {
			log.Error(err, "failed to deployment update replica count")
			return ctrl.Result{}, err
		}
		r.Recorder.Eventf(&myKind, core.EventTypeNormal, "Scaled", "Scaled deployment %q to %d replicas", deployment.Name, expectedReplicas)
		return ctrl.Result{}, nil
	}
	log.Info("replica count up to date", "replica_count", *deployment.Spec.Replicas)
	log.Info("updating MyKind resource status")
	myKind.Status.ReadyReplicas = *deployment.Spec.Replicas
	if err := r.Client.Status().Update(ctx, &myKind); err != nil {
		log.Error(err, "failed to update MyKind status")
		return ctrl.Result{}, err
	}
	log.Info("resource status synced")
	return ctrl.Result{}, nil
}

var (
	deploymentOwnerKey = ".metadata.controller"
)

// return point type
func int64Ptr(i int64) *int64 { return &i }
func int32Ptr(i int32) *int32 { return &i }

func (r *MyKindReconciler) cleanupOwnedResources(ctx context.Context, log logr.Logger, myKind *appsv1.MyKind) error {
	log.Info("finding existing Deployments for MyKind resource")
	//var deployments apps.DeploymentList
	deployments := &apps.DeploymentList{}
	labelSelector := labels.SelectorFromSet(map[string]string{"app": "deploystack"})
	listOps := &client.ListOptions{Namespace: myKind.Namespace, LabelSelector: labelSelector}
	if err := r.List(ctx, deployments, listOps); err != nil {
		return err
	}
	// if err := r.List(ctx, deployments, client.InNamespace(myKind.Namespace), client.MatchingLabelsSelector{Selector: selector}); err != nil {
	// 	return err
	// }

	fmt.Println("###############")

	fmt.Println("deployments:", deployments)
	deleted := 0
	for _, depl := range deployments.Items {
		fmt.Println("deplName:", depl.Name)
		if depl.Name == myKind.Spec.DeploymentName {
			continue
		}
		deletePolicy := metav1.DeletePropagationBackground
		deleteOps := client.DeleteOptions{GracePeriodSeconds: int64Ptr(15), PropagationPolicy: &deletePolicy}
		if err := r.Client.Delete(ctx, &depl, &deleteOps); err != nil {
			log.Error(err, "failed to delete Deployment resource")
			return err
		}
		r.Recorder.Eventf(myKind, core.EventTypeNormal, "Deleted", "Deleted deployment %q", depl.Name)
		deleted++
	}
	log.Info("finished cleaning up old Deployment resources", "number_deleted", deleted)
	return nil
}

func buildDeployment(myKind appsv1.MyKind) *apps.Deployment {
	deployment := apps.Deployment{

		ObjectMeta: metav1.ObjectMeta{
			Name:      myKind.Spec.DeploymentName,
			Namespace: myKind.Namespace,
			Labels: map[string]string{
				"app": "deploystack",
			},
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(&myKind, appsv1.GroupVersion.WithKind("MyKind"))},
		},
		Spec: apps.DeploymentSpec{
			Replicas: myKind.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/created-by": myKind.Spec.DeploymentName,
					"app":                          "deploystack",
				},
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/created-by": myKind.Spec.DeploymentName,
						"app":                          "deploystack",
					},
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
	}
	return &deployment
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyKindReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.MyKind{}).
		Complete(r)
}
