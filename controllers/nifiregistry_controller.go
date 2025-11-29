// controllers/nifiregistry_controller.go

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NifiRegistryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// 1. Fetch the NifiRegistry instance
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

	// 2. Create or Update ConfigMap
	cm := configMapForNifiRegistry(nifiRegistry, r.Scheme)
	if err := controllerutil.SetControllerReference(nifiRegistry, cm, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	foundCM := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, foundCM)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
		err = r.Create(ctx, cm)
		if err != nil {
			log.Error(err, "Failed to create new ConfigMap")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		// Update existing ConfigMap
		if foundCM.Data["log4j2.xml"] != cm.Data["log4j2.xml"] || foundCM.Data["nifi-registry.properties"] != cm.Data["nifi-registry.properties"] {
			log.Info("Updating ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
			cm.ObjectMeta.ResourceVersion = foundCM.ObjectMeta.ResourceVersion
			err = r.Update(ctx, cm)
			if err != nil {
				log.Error(err, "Failed to update ConfigMap")
				return ctrl.Result{}, err
			}
		}
	}

	// 3. Create or Update Service
	svc := serviceForNifiRegistry(nifiRegistry, r.Scheme)
	if err := controllerutil.SetControllerReference(nifiRegistry, svc, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	foundSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, foundSvc)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		err = r.Create(ctx, svc)
		if err != nil {
			log.Error(err, "Failed to create new Service")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// 4. Create or Update PVC (только если FlowStorage задан)
	if nifiRegistry.Spec.FlowStorage.Enabled { // <--- ИСПРАВЛЕНИЕ ЗДЕСЬ
		pvc := pvcForNifiRegistry(nifiRegistry, r.Scheme)
		if err := controllerutil.SetControllerReference(nifiRegistry, pvc, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		foundPVC := &corev1.PersistentVolumeClaim{}
		err = r.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, foundPVC)
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating a new PVC", "PVC.Namespace", pvc.Namespace, "PVC.Name", pvc.Name)
			err = r.Create(ctx, pvc)
			if err != nil {
				log.Error(err, "Failed to create new PVC")
				return ctrl.Result{}, err
			}
		} else if err != nil {
			return ctrl.Result{}, err
		}
	}

	// 5. Create or Update Deployment
	dep := deploymentForNifiRegistry(nifiRegistry, r.Scheme)
	if err := controllerutil.SetControllerReference(nifiRegistry, dep, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	foundDep := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: dep.Name, Namespace: dep.Namespace}, foundDep)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NifiRegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&registryv1.NifiRegistry{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}