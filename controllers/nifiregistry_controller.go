// controllers/nifiregistry_controller.go

package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
)

// NifiRegistryReconciler reconciles a NifiRegistry object
type NifiRegistryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=registry.nifi.oper,resources=nifiregistries,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=registry.nifi.oper,resources=nifiregistries/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=registry.nifi.oper,resources=nifiregistries/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NifiRegistryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("🎯 RECONCILE CALLED")

	nifiRegistry := &registryv1.NifiRegistry{}
	err := r.Get(ctx, req.NamespacedName, nifiRegistry)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("NifiRegistry resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get NifiRegistry")
		return ctrl.Result{}, err
	}

	// 1. ConfigMap
	configMap := configMapForNifiRegistry(nifiRegistry, r.Scheme)
	foundConfigMap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: configMap.Name, Namespace: nifiRegistry.Namespace}, foundConfigMap)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
		err = r.Create(ctx, configMap)
		if err != nil {
			log.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	}

	// 2. PVC for Flow Storage (если включено)
	if nifiRegistry.Spec.FlowStorage.Enabled {
		pvc := persistentVolumeClaimForFlowStorage(nifiRegistry)
		foundPVC := &corev1.PersistentVolumeClaim{}
		err = r.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: nifiRegistry.Namespace}, foundPVC)
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating a new PVC", "PVC.Namespace", pvc.Namespace, "PVC.Name", pvc.Name)
			err = r.Create(ctx, pvc)
			if err != nil {
				log.Error(err, "Failed to create new PVC", "PVC.Namespace", pvc.Namespace, "PVC.Name", pvc.Name)
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			log.Error(err, "Failed to get PVC")
			return ctrl.Result{}, err
		}
	}

	// 3. Service
	serviceName := fmt.Sprintf("%s-service", nifiRegistry.Name) // ИСПРАВЛЕНИЕ: Используем имя с суффиксом
	service := &corev1.Service{}

	err = r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: nifiRegistry.Namespace}, service)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Service", "Service.Namespace", nifiRegistry.Namespace, "Service.Name", serviceName)
		newService := serviceForNifiRegistry(nifiRegistry, r.Scheme)
		err = r.Create(ctx, newService)
		if err != nil {
			log.Error(err, "Failed to create new Service", "Service.Namespace", nifiRegistry.Namespace, "Service.Name", serviceName)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		return ctrl.Result{}, err
	}

	// 4. Deployment
	deployment := deploymentForNifiRegistry(nifiRegistry, r.Scheme)
	foundDeployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: nifiRegistry.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = r.Create(ctx, deployment)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// Check if the deployment size is the desired size
	if *foundDeployment.Spec.Replicas != nifiRegistry.Spec.Size {
		log.Info("Updating Deployment size", "Current", *foundDeployment.Spec.Replicas, "Desired", nifiRegistry.Spec.Size)
		foundDeployment.Spec.Replicas = &nifiRegistry.Spec.Size
		err = r.Update(ctx, foundDeployment)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", foundDeployment.Namespace, "Deployment.Name", foundDeployment.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// 5. Обновление статуса (ИСПРАВЛЕНИЕ КОНФЛИКТА)

	// Читаем последнюю версию объекта перед обновлением статуса,
	// чтобы избежать ошибки "object has been modified".
	latestNifiRegistry := &registryv1.NifiRegistry{}
	err = r.Get(ctx, req.NamespacedName, latestNifiRegistry)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to re-fetch NifiRegistry before status update")
		return ctrl.Result{}, err
	}

	// Если статус не "Running", обновляем его
	if latestNifiRegistry.Status.State != "Running" {
		latestNifiRegistry.Status.State = "Running"

		// Используем Status().Update()
		if err := r.Status().Update(ctx, latestNifiRegistry); err != nil {
			// Если ошибка - это конфликт (object has been modified),
			// возвращаем Requeue, чтобы попробовать снова с новой версией.
			if errors.IsConflict(err) {
				log.Info("Status update conflict, retrying reconcile loop.")
				return ctrl.Result{Requeue: true}, nil
			}
			log.Error(err, "Failed to update NifiRegistry status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NifiRegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&registryv1.NifiRegistry{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Service{}). // ВОССТАНОВЛЕНО: Отслеживание Service
		Complete(r)
}
