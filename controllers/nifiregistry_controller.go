package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	registryv1 "github.com/repinoid/nreg-oper/api/v1"
)

const nifiRegistryFinalizer = "registry.nifi.oper/finalizer"

// NifiRegistryReconciler reconciles a NifiRegistry object
type NifiRegistryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=registry.nifi.oper,resources=nifiregistries,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=registry.nifi.oper,resources=nifiregistries/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=registry.nifi.oper,resources=nifiregistries/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete // <-- ДОБАВЛЕНО

func (r *NifiRegistryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("🎯 RECONCILE CALLED", "name", req.Name, "namespace", req.Namespace)

	var nifiRegistry registryv1.NifiRegistry
	if err := r.Get(ctx, req.NamespacedName, &nifiRegistry); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// --- ЛОГИКА FINALIZER (УДАЛЕНИЕ) ---
	isNifiRegistryMarkedForDeletion := nifiRegistry.GetDeletionTimestamp() != nil
	if isNifiRegistryMarkedForDeletion {
		if controllerutil.ContainsFinalizer(&nifiRegistry, nifiRegistryFinalizer) {
			log.Info("Performing Finalizer logic to delete external resources...")

			if err := r.deleteExternalResources(ctx, &nifiRegistry); err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(&nifiRegistry, nifiRegistryFinalizer)
			if err := r.Update(ctx, &nifiRegistry); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// --- ДОБАВЛЕНИЕ FINALIZER ---
	if !controllerutil.ContainsFinalizer(&nifiRegistry, nifiRegistryFinalizer) {
		controllerutil.AddFinalizer(&nifiRegistry, nifiRegistryFinalizer)
		if err := r.Update(ctx, &nifiRegistry); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 🎯 СОЗДАЕМ CONFIGMAP
	if err := r.reconcileConfigMap(ctx, &nifiRegistry); err != nil {
		return ctrl.Result{}, err
	}

	// 🎯 СОЗДАЕМ FLOW STORAGE PVC, если он включен
	if nifiRegistry.Spec.FlowStorage.Enabled {
		if err := r.reconcilePVC(ctx, &nifiRegistry); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 🎯 СОЗДАЕМ DEPLOYMENT
	if err := r.reconcileDeployment(ctx, &nifiRegistry); err != nil {
		return ctrl.Result{}, err
	}

	// 🎯 СОЗДАЕМ SERVICE (ClusterIP)
	if err := r.reconcileService(ctx, &nifiRegistry); err != nil {
		return ctrl.Result{}, err
	}

	// --- ОБНОВЛЕНИЕ СТАТУСА ---
	nifiRegistry.Status.State = "Ready"
	if err := r.Status().Update(ctx, &nifiRegistry); err != nil {
		log.Error(err, "Failed to update NifiRegistry status")
		return ctrl.Result{}, err
	}

	log.Info("✅ RECONCILE УСПЕШЕН")
	return ctrl.Result{RequeueAfter: 30 * 1000 * 1000 * 1000}, nil
}

// --- ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ДЛЯ RECONCILE/SECRETS/DELETION ---

func (r *NifiRegistryReconciler) deleteExternalResources(ctx context.Context, nifiRegistry *registryv1.NifiRegistry) error {
	log := log.FromContext(ctx)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: nifiRegistry.Name, Namespace: nifiRegistry.Namespace},
	}
	log.Info("Deleting Deployment")
	if err := r.Delete(ctx, deployment); client.IgnoreNotFound(err) != nil {
		return err
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: nifiRegistry.Name, Namespace: nifiRegistry.Namespace},
	}
	log.Info("Deleting Service")
	if err := r.Delete(ctx, service); client.IgnoreNotFound(err) != nil {
		return err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-config", nifiRegistry.Name), Namespace: nifiRegistry.Namespace},
	}
	log.Info("Deleting ConfigMap")
	if err := r.Delete(ctx, configMap); client.IgnoreNotFound(err) != nil {
		return err
	}

	// Удаление PVC, если он включен
	if nifiRegistry.Spec.FlowStorage.Enabled {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-flow", nifiRegistry.Name), Namespace: nifiRegistry.Namespace},
		}
		log.Info("Deleting Flow Storage PVC")
		if err := r.Delete(ctx, pvc); client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	return nil
}

func (r *NifiRegistryReconciler) getClientSecretValue(ctx context.Context, nifiRegistry *registryv1.NifiRegistry) (string, error) {
	secretName := nifiRegistry.Spec.Keycloak.ClientSecretName
	if secretName == "" {
		return "", fmt.Errorf("keycloak client secret name is not specified")
	}

	secret := &corev1.Secret{}
	key := "clientSecret"

	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: nifiRegistry.Namespace}, secret); err != nil {
		return "", fmt.Errorf("failed to get Keycloak Client Secret '%s': %w", secretName, err)
	}

	secretBytes, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("key '%s' not found in Keycloak Client Secret '%s'", key, secretName)
	}

	return string(secretBytes), nil
}

func (r *NifiRegistryReconciler) getDBSecretValue(ctx context.Context, nifiRegistry *registryv1.NifiRegistry) (string, error) {
	dbSpec := nifiRegistry.Spec.Database
	if !dbSpec.Enabled || dbSpec.SecretName == "" {
		return "", nil
	}

	secret := &corev1.Secret{}
	key := "password"
	secretName := dbSpec.SecretName

	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: nifiRegistry.Namespace}, secret); err != nil {
		return "", fmt.Errorf("failed to get Database Secret '%s': %w", secretName, err)
	}

	secretBytes, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("key '%s' not found in Database Secret '%s'", key, secretName)
	}

	return string(secretBytes), nil
}

func (r *NifiRegistryReconciler) reconcileConfigMap(ctx context.Context, nifiRegistry *registryv1.NifiRegistry) error {
	log := log.FromContext(ctx)

	// 1. Получаем Keycloak Secret
	clientSecret, err := r.getClientSecretValue(ctx, nifiRegistry)
	if err != nil {
		log.Error(err, "Failed to get Keycloak Client Secret", "Secret.Name", nifiRegistry.Spec.Keycloak.ClientSecretName)
		return err
	}

	// 2. Получаем DB Password
	dbPassword := ""
	if nifiRegistry.Spec.Database.Enabled {
		dbPassword, err = r.getDBSecretValue(ctx, nifiRegistry)
		if err != nil {
			log.Error(err, "Failed to get Database Password Secret", "Secret.Name", nifiRegistry.Spec.Database.SecretName)
			return err
		}
	}

	configMapName := fmt.Sprintf("%s-config", nifiRegistry.Name)
	configMap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: nifiRegistry.Namespace}, configMap)

	if errors.IsNotFound(err) {
		// !!! ИСПРАВЛЕН ВЫЗОВ: теперь как функция, передающая r.Scheme !!!
		newConfigMap := configMapForNifiRegistry(nifiRegistry, configMapName, clientSecret, dbPassword, r.Scheme)
		log.Info("Creating a new ConfigMap", "ConfigMap.Namespace", newConfigMap.Namespace, "ConfigMap.Name", newConfigMap.Name)
		if err = r.Create(ctx, newConfigMap); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

// reconcilePVC создает PVC, если он включен в CR
func (r *NifiRegistryReconciler) reconcilePVC(ctx context.Context, nifiRegistry *registryv1.NifiRegistry) error {
	log := log.FromContext(ctx)
	pvcName := fmt.Sprintf("%s-flow", nifiRegistry.Name)
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: pvcName, Namespace: nifiRegistry.Namespace}, pvc)

	if errors.IsNotFound(err) {
		// !!! ИСПРАВЛЕН ВЫЗОВ: теперь как функция, передающая r.Scheme !!!
		newPVC := persistentVolumeClaimForFlowStorage(nifiRegistry)

		// Устанавливаем OwnerReference, используя r.Scheme
		if err := ctrl.SetControllerReference(nifiRegistry, newPVC, r.Scheme); err != nil {
			return err
		}

		log.Info("Creating a new PersistentVolumeClaim for Flow Storage", "PVC.Namespace", newPVC.Namespace, "PVC.Name", newPVC.Name)
		if err = r.Create(ctx, newPVC); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

func (r *NifiRegistryReconciler) reconcileDeployment(ctx context.Context, nifiRegistry *registryv1.NifiRegistry) error {
	log := log.FromContext(ctx)
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: nifiRegistry.Name, Namespace: nifiRegistry.Namespace}, deployment)

	if errors.IsNotFound(err) {
		// !!! ИСПРАВЛЕН ВЫЗОВ: теперь как функция, передающая r.Scheme !!!
		newDeployment := deploymentForNifiRegistry(nifiRegistry, r.Scheme)
		log.Info("Creating a new Deployment", "Deployment.Namespace", newDeployment.Namespace, "Deployment.Name", newDeployment.Name)
		if err = r.Create(ctx, newDeployment); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func (r *NifiRegistryReconciler) reconcileService(ctx context.Context, nifiRegistry *registryv1.NifiRegistry) error {
	log := log.FromContext(ctx)
	service := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: nifiRegistry.Name, Namespace: nifiRegistry.Namespace}, service)

	if errors.IsNotFound(err) {
		// !!! ИСПРАВЛЕН ВЫЗОВ: теперь как функция, передающая r.Scheme !!!
		newService := serviceForNifiRegistry(nifiRegistry, r.Scheme)
		log.Info("Creating a new Service", "Service.Namespace", newService.Namespace, "Service.Name", newService.Name)
		if err = r.Create(ctx, newService); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func (r *NifiRegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&registryv1.NifiRegistry{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.PersistentVolumeClaim{}). // <-- ДОБАВЛЕНО
		Complete(r)
}
