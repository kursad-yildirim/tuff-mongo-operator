package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	appv1alpha1 "gitlab.cee.redhat.com/kyildiri/tuff-mongo-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// TuffMongoReconciler reconciles a TuffMongo object
type TuffMongoReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.tuff.local,resources=tuffmongoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.tuff.local,resources=tuffmongoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.tuff.local,resources=tuffmongoes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TuffMongo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *TuffMongoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//_ = log.FromContext(ctx)
	log := ctrllog.FromContext(ctx)

	// TODO(user): your logic here

	// Fetch the PodSet instance
	instance := &appv1alpha1.TuffMongo{}
	err := r.Get(context.TODO(), req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	// List all pods owned by this MicroMongo instance
	tuffMongo := instance
	mongoPodList := &corev1.PodList{}
	tmLabels := map[string]string{
		"app":     tuffMongo.Name,
		"version": "v0.1",
	}
	tmLabelSelector := labels.SelectorFromSet(tmLabels)
	tmListOptions := &client.ListOptions{Namespace: tuffMongo.Namespace, LabelSelector: tmLabelSelector}
	if err = r.List(context.TODO(), mongoPodList, tmListOptions); err != nil {
		return ctrl.Result{}, err
	}
	// Count the pods that are pending or running as available
	var availableMongoPods []corev1.Pod
	for _, pod := range mongoPodList.Items {
		if pod.ObjectMeta.DeletionTimestamp != nil {
			continue
		}
		if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodPending {
			availableMongoPods = append(availableMongoPods, pod)
		}
	}
	numAvailableMongoPods := int32(len(availableMongoPods))
	availableMongoPodNames := []string{}
	for _, pod := range availableMongoPods {
		availableMongoPodNames = append(availableMongoPodNames, pod.ObjectMeta.Name)
	}
	// Update the status if necessary
	status := appv1alpha1.TuffMongoStatus{
		MongoPodNames:          availableMongoPodNames,
		MongoAvailableReplicas: numAvailableMongoPods,
	}
	if !reflect.DeepEqual(tuffMongo.Status, status) {
		tuffMongo.Status = status
		err = r.Status().Update(context.TODO(), tuffMongo)
		if err != nil {
			log.Error(err, "Failed  to update tuffMongo status.")
			return ctrl.Result{}, err
		}
	}

	if numAvailableMongoPods > tuffMongo.Spec.MongoReplicas {
		log.Info("Scaling down mongodb pods", "Currently available", numAvailableMongoPods, "Required replicas", tuffMongo.Spec.MongoReplicas)
		diff := numAvailableMongoPods - tuffMongo.Spec.MongoReplicas
		dpods := availableMongoPods[:diff]
		for _, podToDelete := range dpods {
			err = r.Delete(context.TODO(), &podToDelete)
			if err != nil {
				log.Error(err, "Failed to delete mongodb pod", "mongoPod.name", podToDelete.Name)
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if numAvailableMongoPods < tuffMongo.Spec.MongoReplicas {
		log.Info("Scaling up mongodb pods", "Currently available", numAvailableMongoPods, "Required replicas", tuffMongo.Spec.MongoReplicas)
		// Define a new mongo pod object
		mongoPod := newMongoPodForCR(tuffMongo)
		// Set TuffMongo instance as the owner and controller
		if err := controllerutil.SetControllerReference(tuffMongo, mongoPod, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		err = r.Create(context.TODO(), mongoPod)
		if err != nil {
			log.Error(err, "FAiled to create mongodb pod", "MongoPod.name", mongoPod.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

func newMongoPodForCR(cr *appv1alpha1.TuffMongo) (mongoPod *corev1.Pod) {
	tmLabels := map[string]string{
		"app":     cr.Name,
		"version": "v0.1",
	}
	mongoPod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: cr.Name + "-pod-",
			Namespace:    cr.Namespace,
			Labels:       tmLabels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:         cr.Spec.MongoContainerName,
					Image:        cr.Spec.MongoImage,
					Ports:        cr.Spec.MongoPorts,
					VolumeMounts: cr.Spec.MongoVolumeMounts,
				},
			},
			Volumes: cr.Spec.MongoVolumes,
		},
	}

	return
}

// SetupWithManager sets up the controller with the Manager.
func (r *TuffMongoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.TuffMongo{}).
		Complete(r)
}
