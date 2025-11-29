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
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete // <-- ВОЗВРАЩЕНО

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

	// 2. Create or Update Database Secret (Только если БД включена) <-- ВОЗВРАЩЕНО
	if nifiRegistry.Spec.Database.Enabled {
		secret := secretForNifiRegistry(nifiRegistry, r.Scheme)
		// SecretForNifiRegistry возвращает nil, если SecretName пуст.
		if secret != nil {
			// ControllerReference установлен в helper, если SecretName существует.
			foundSecret := &corev1.Secret{}
			err = r.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, foundSecret)
			if err != nil && errors.IsNotFound(err) {
				log.Info("Creating a new Database Secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
				// Если Secret не найден, мы его создаем (но только если БД включена)
				err = r.Create(ctx, secret)
				if err != nil {
					log.Error(err, "Failed to create new Database Secret")
					return ctrl.Result{}, err
				}
			} else if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// 3. Create or Update Service
	svc := serviceForNifiRegistry(nifiRegistry, r.Scheme)
	// ... (логика создания/обновления Service)
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
	if nifiRegistry.Spec.FlowStorage.Enabled {
		pvc := pvcForNifiRegistry(nifiRegistry, r.Scheme)
		// ... (логика создания/обновления PVC)
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
	// ... (логика создания/обновления Deployment)
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
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.Secret{}). // <-- ВОЗВРАЩЕНО
		Complete(r)
}
