package workflow

import (
	"context"
	"math/rand"
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

const (
	finalizerName = "workflow.finalizers.lyra.org"
)

// Applicator abstracts over workflow application and deletion
type Applicator interface {
	ApplyWorkflowWithHieraData(workflowName string, data map[string]string)

	//DeleteWorkflowWithHieraData calls the delete on the workflow in lyra, meaning that resources will be destroyed, if applicable
	DeleteWorkflowWithHieraData(workflowName string, data map[string]string)
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
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	reqLogger = reqLogger.WithValues("instance", instance)

	//Ensure that our finalizer is present, and error out if not present
	if !containsString(instance.ObjectMeta.Finalizers, finalizerName) {
		reqLogger.Info("Adding finalizer", "finalizerName", finalizerName)
		instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, finalizerName)
		if err := r.client.Update(context.Background(), instance); err != nil {
			reqLogger.Info("Error adding finalizer, will exit and requeue", "err", err)
			return reconcile.Result{Requeue: true}, nil
		}
		reqLogger.Info("Added our finalizer to the list", "finalizers", instance.ObjectMeta.Finalizers)
	}

	workflowName := instance.Spec.WorkflowName
	reqLogger = reqLogger.WithValues("WorkflowName", workflowName)
	data := instance.Spec.Data
	refreshTime := time.Duration(instance.Spec.RefreshTime) * time.Second

	//This is a Delete: delete the workflow resources and do not requeue if successful
	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		reqLogger.WithValues("finalizers", instance.ObjectMeta.Finalizers).Info("The object is being deleted, let's check the finalizers")
		if containsString(instance.ObjectMeta.Finalizers, finalizerName) {

			reqLogger.Info("Controller will delete workflow ...", "data", data)

			//attempt to delete but recover from panics
			requeue := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						if feelingLucky() {
							reqLogger.Info("We recovered from a panic whilst deleting the workflow. We decided to requeue", "cause-of-panic", r)
							requeue = true
						} else {
							reqLogger.Info("We recovered from a panic whilst deleting the workflow. We decided (random circuit breaker) NOT to requeue", "cause-of-panic", r)
						}
					}
				}()
				r.applicator.DeleteWorkflowWithHieraData(workflowName, data)
			}()
			if requeue {
				return reconcile.Result{Requeue: true}, nil
			}

			reqLogger.Info("Deleted workflow", "data", data)
			instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, finalizerName)
			if err := r.client.Update(context.Background(), instance); err != nil {
				reqLogger.Info("Something went wrong attempting to remove our finalizer, will exit and requeue", "err", err)
				return reconcile.Result{Requeue: true}, nil
			}
			reqLogger.Info("we removed our finalizer and deleted the workflow resources", "finalizers", instance.ObjectMeta.Finalizers)
			return reconcile.Result{}, nil
		}
		reqLogger.Info("A deletion event happened but our finalizer was not present", "finalizers", instance.ObjectMeta.Finalizers, "")
		return reconcile.Result{}, nil
	}

	reqLogger.Info("Controller will apply workflow ...", "data", data, "refreshTime", refreshTime)

	//attempt to delete but recover from panics
	requeue := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				reqLogger.Info("We recovered a failure applying the workflow", "cause-of-panic", r)
				requeue = true
			}
		}()
		r.applicator.ApplyWorkflowWithHieraData(workflowName, data)
	}()

	reqLogger.Info("Controller has completed applying workflow...", "requeue", requeue)

	if refreshTime == 0 && !requeue {
		return reconcile.Result{}, nil
	}
	return reconcile.Result{RequeueAfter: refreshTime}, nil
}

// Returns (randomly) true in roughly 80% of cases, and false in roughly 20%
// although there is an exponential backoff in k8s, we could be trying to delete a workflow forever if a persistent error
// (anecdotally, requeue time roughly doubles from 3 seconds each attempt after 10 attempts)
func feelingLucky() bool {
	rand.Seed(time.Now().UTC().UnixNano())
	num := rand.Intn(100)
	log.WithValues("num", num).Info("random number")
	return num%10 > 1 //80% chance of true
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
