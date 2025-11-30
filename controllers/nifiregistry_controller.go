// Filename: controllers/nifiregistry_controller.go
// Changes: Integrated Keycloak ConfigMap creation logic into Reconcile.
//          Updated reconcileConfigMap to handle all three ConfigMaps (providers, identity-providers, authorizers)
//          and use r.Log as defined in the Reconciler struct.

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	registryv1 "github.com/repinoid/nreg-oper/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// NifiRegistryReconciler reconciles a NifiRegistry object
type NifiRegistryReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=registry.nifi.oper,resources=nifiregistries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=registry.nifi.oper,resources=nifiregistries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=registry.nifi.oper,resources=nifiregistries/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NifiRegistryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("nifiregistry", req.NamespacedName)

	// Fetch the NifiRegistry instance
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

	// ========================================================================================
	// 1. Управление PostgreSQL (если включено)
	// ========================================================================================

	if nifiRegistry.Spec.PostgreSQL.Enabled {
		// A. Создание PVC для PostgreSQL (логика опущена)
		pvcPostgres := pvcForPostgreSQL(nifiRegistry)
		if err := r.ensurePVC(ctx, log, nifiRegistry, pvcPostgres); err != nil {
			return ctrl.Result{}, err
		}

		// B. Создание Service для PostgreSQL (логика опущена)
		svcPostgres := serviceForPostgreSQL(nifiRegistry)
		if err := controllerutil.SetControllerReference(nifiRegistry, svcPostgres, r.Scheme); err != nil {
			log.Error(err, "Failed to set controller reference for PostgreSQL Service")
			return ctrl.Result{}, err
		}
		foundSVCPostgres := &corev1.Service{}
		err = r.Get(ctx, types.NamespacedName{Name: svcPostgres.Name, Namespace: svcPostgres.Namespace}, foundSVCPostgres)
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating PostgreSQL Service", "Name", svcPostgres.Name)
			err = r.Create(ctx, svcPostgres)
			if err != nil {
				log.Error(err, "Failed to create PostgreSQL Service")
				return ctrl.Result{RequeueAfter: time.Second * 5}, err
			}
		} else if err != nil {
			log.Error(err, "Failed to get PostgreSQL Service")
			return ctrl.Result{}, err
		}

		// C. Создание Deployment для PostgreSQL (логика опущена)
		depPostgres := deploymentForPostgreSQL(nifiRegistry)
		if err := controllerutil.SetControllerReference(nifiRegistry, depPostgres, r.Scheme); err != nil {
			log.Error(err, "Failed to set controller reference for PostgreSQL Deployment")
			return ctrl.Result{}, err
		}
		foundDEPPostgres := &appsv1.Deployment{}
		err = r.Get(ctx, types.NamespacedName{Name: depPostgres.Name, Namespace: depPostgres.Namespace}, foundDEPPostgres)
		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating PostgreSQL Deployment", "Name", depPostgres.Name)
			err = r.Create(ctx, depPostgres)
			if err != nil {
				log.Error(err, "Failed to create PostgreSQL Deployment")
				return ctrl.Result{RequeueAfter: time.Second * 5}, err
			}
		} else if err != nil {
			log.Error(err, "Failed to get PostgreSQL Deployment")
			return ctrl.Result{}, err
		}
	}

	// ========================================================================================
	// 2. Управление хранилищем NiFi Registry (PVC)
	// ========================================================================================

	// A. Создание PVC для Flow Storage (если включено)
	if nifiRegistry.Spec.FlowStorage.Enabled {
		pvcFlow := pvcForFlowStorage(nifiRegistry)
		if err := r.ensurePVC(ctx, log, nifiRegistry, pvcFlow); err != nil {
			return ctrl.Result{}, err
		}
	}

	// B. Создание PVC для Lib Storage (если включено ИЛИ нужна БД)
	if nifiRegistry.Spec.LibStorage.Enabled || nifiRegistry.Spec.Database.Enabled {
		pvcLib := pvcForLibStorage(nifiRegistry)
		if err := r.ensurePVC(ctx, log, nifiRegistry, pvcLib); err != nil {
			return ctrl.Result{}, err
		}
	}

	// ========================================================================================
	// 3. Управление ConfigMap
	// ========================================================================================

	// A. ConfigMap для providers.xml (СУЩЕСТВУЮЩАЯ ЛОГИКА)
	configMapRegistry := configMapForNifiRegistry(nifiRegistry, r.Scheme)
	if err := r.reconcileConfigMap(ctx, log, nifiRegistry, configMapRegistry, "Providers ConfigMap"); err != nil {
		return ctrl.Result{}, err
	}

	// B. ConfigMaps для Keycloak (ЕСЛИ ВКЛЮЧЕН)
	if nifiRegistry.Spec.Keycloak.Enabled {
		// ConfigMap для identity-providers.xml (Аутентификация OIDC)
		configMapIdentity := configMapIdentityProvidersForNifiRegistry(nifiRegistry, r.Scheme)
		if err := r.reconcileConfigMap(ctx, log, nifiRegistry, configMapIdentity, "Identity Providers ConfigMap"); err != nil {
			return ctrl.Result{}, err
		}

		// ConfigMap для authorizers.xml (Авторизация)
		configMapAuthorizers := configMapAuthorizersForNifiRegistry(nifiRegistry, r.Scheme)
		if err := r.reconcileConfigMap(ctx, log, nifiRegistry, configMapAuthorizers, "Authorizers ConfigMap"); err != nil {
			return ctrl.Result{}, err
		}
	}

	// ========================================================================================
	// 4. Управление NiFi Registry
	// ========================================================================================

	// A. Создание Service для NiFi Registry (логика опущена)
	svcRegistry := serviceForNifiRegistry(nifiRegistry, r.Scheme)
	if err := controllerutil.SetControllerReference(nifiRegistry, svcRegistry, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference for Registry Service")
		return ctrl.Result{}, err
	}
	foundSVCRegistry := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: svcRegistry.Name, Namespace: svcRegistry.Namespace}, foundSVCRegistry)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Registry Service", "Name", svcRegistry.Name)
		err = r.Create(ctx, svcRegistry)
		if err != nil {
			log.Error(err, "Failed to create Registry Service")
			return ctrl.Result{RequeueAfter: time.Second * 5}, err
		}
	} else if err != nil {
		log.Error(err, "Failed to get Registry Service")
		return ctrl.Result{}, err
	}

	// B. Создание Deployment для NiFi Registry (логика опущена)
	depRegistry := deploymentForNifiRegistry(nifiRegistry)
	if err := controllerutil.SetControllerReference(nifiRegistry, depRegistry, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference for Registry Deployment")
		return ctrl.Result{}, err
	}
	foundDEPRegistry := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: depRegistry.Name, Namespace: depRegistry.Namespace}, foundDEPRegistry)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Registry Deployment", "Name", depRegistry.Name)
		err = r.Create(ctx, depRegistry)
		if err != nil {
			log.Error(err, "Failed to create Registry Deployment")
			return ctrl.Result{RequeueAfter: time.Second * 5}, err
		}
	} else if err != nil {
		log.Error(err, "Failed to get Registry Deployment")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// reconcileConfigMap проверяет существование ConfigMap и создает его, если он отсутствует.
// Теперь принимает желаемый ConfigMap и имя для логирования.
func (r *NifiRegistryReconciler) reconcileConfigMap(ctx context.Context, log logr.Logger, nifiRegistry *registryv1.NifiRegistry, desiredCm *corev1.ConfigMap, logName string) error {
	foundCM := &corev1.ConfigMap{}

	// 1. Попытка получить ConfigMap
	err := r.Get(ctx, types.NamespacedName{Name: desiredCm.Name, Namespace: nifiRegistry.Namespace}, foundCM)

	if err != nil && errors.IsNotFound(err) {
		// 2. Если не найден, СОЗДАЕМ его

		if err := controllerutil.SetControllerReference(nifiRegistry, desiredCm, r.Scheme); err != nil {
			log.Error(err, fmt.Sprintf("Failed to set controller reference for %s", logName))
			return err
		}

		log.Info("Creating", "ConfigMap", logName, "Name", desiredCm.Name)
		err = r.Client.Create(ctx, desiredCm)
		if err != nil {
			log.Error(err, fmt.Sprintf("Failed to create %s", logName))
			return err
		}
	} else if err != nil {
		// Ошибка при получении ConfigMap
		log.Error(err, fmt.Sprintf("Failed to get existing %s", logName))
		return err
	}

	// TODO: Здесь должна быть логика обновления ConfigMap, если его содержимое изменилось.

	return nil
}

// ensurePVC проверяет существование PVC и создает его, если он отсутствует.
func (r *NifiRegistryReconciler) ensurePVC(ctx context.Context, log logr.Logger, nifiRegistry *registryv1.NifiRegistry, pvc *corev1.PersistentVolumeClaim) error {
	if err := controllerutil.SetControllerReference(nifiRegistry, pvc, r.Scheme); err != nil {
		log.Error(err, fmt.Sprintf("Failed to set controller reference for PVC %s", pvc.Name))
		return err
	}
	foundPVC := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, foundPVC)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating PVC", "Name", pvc.Name)
		err = r.Create(ctx, pvc)
		if err != nil {
			log.Error(err, fmt.Sprintf("Failed to create PVC %s", pvc.Name))
			return err
		}
	} else if err != nil {
		log.Error(err, fmt.Sprintf("Failed to get PVC %s", pvc.Name))
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NifiRegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&registryv1.NifiRegistry{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.ConfigMap{}). // Добавлено владение ConfigMap
		Complete(r)
}
