package workflow

import (
	"context"
	"time"

	lyrav1alpha1 "github.com/lyraproj/lyra-operator/pkg/apis/lyra/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_workflow")

// Applicator abstracts over workflow application
type Applicator interface {
	ApplyWorkflowWithHieraData(workflowName string, data map[string]string)
}

// Add creates a new Workflow Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, applicator Applicator) error {
	return add(mgr, newReconciler(mgr, applicator))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, applicator Applicator) reconcile.Reconciler {
	return &ReconcileWorkflow{
		client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		applicator: applicator,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("workflow-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Workflow
	err = c.Watch(&source.Kind{Type: &lyrav1alpha1.Workflow{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Workflow
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &lyrav1alpha1.Workflow{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileWorkflow{}

// ReconcileWorkflow reconciles a Workflow object
type ReconcileWorkflow struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client     client.Client
	scheme     *runtime.Scheme
	applicator Applicator
}

// Reconcile reads that state of the cluster for a Workflow object and makes changes based on the state read
// and what is in the Workflow.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileWorkflow) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Workflow")

	// Fetch the Workflow instance
	instance := &lyrav1alpha1.Workflow{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Ask Lyra to reconsile this workflow
	workflowName := instance.Spec.WorkflowName
	data := instance.Spec.Data
	refreshTime := time.Duration(instance.Spec.RefreshTime) * time.Second
	reqLogger.Info("Controller will apply workflow ...",
		"WorkflowName", workflowName,
		"Data", data,
		"RefreshTime", refreshTime)

	// FIXME error handling should be used here to requeue
	r.applicator.ApplyWorkflowWithHieraData(workflowName, data)
	if refreshTime == 0 {
		return reconcile.Result{}, nil
	}
	return reconcile.Result{RequeueAfter: refreshTime}, nil
}
