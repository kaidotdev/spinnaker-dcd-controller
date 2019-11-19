package controllers

import (
	"context"
	"encoding/json"
	applicationV1 "spinnaker-dcd-controller/api/v1"

	"github.com/spinnaker/roer/spinnaker"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ApplicationReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	SpinnakerClient spinnaker.Client
}

const myFinalizerName = "spinnaker.finalizers.kaidotdev.github.io"

func (r *ApplicationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	application := &applicationV1.Application{}
	ctx := context.Background()
	logger := r.Log.WithValues("application", req.NamespacedName)
	if err := r.Get(ctx, req.NamespacedName, application); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if application.ObjectMeta.DeletionTimestamp.IsZero() {
		task := r.buildCreateTask(req.Name, application)
		r.submitTask(req.Name, task, logger)

		if !containsString(application.ObjectMeta.Finalizers, myFinalizerName) {
			application.ObjectMeta.Finalizers = append(application.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), application); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if containsString(application.ObjectMeta.Finalizers, myFinalizerName) {
			task := r.buildDeleteTask(req.Name, application)
			r.submitTask(req.Name, task, logger)

			application.ObjectMeta.Finalizers = removeString(application.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), application); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
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

func (r *ApplicationReconciler) submitTask(applicationName string, task spinnaker.Task, logger logr.Logger) {
	ref, err := r.SpinnakerClient.ApplicationSubmitTask(applicationName, task)
	if err != nil {
		logger.Error(err, "Failed to ApplicationSubmitTask")
	}
	response, err := r.SpinnakerClient.PollTaskStatus(ref.Ref, 0)
	if err != nil {
		logger.Error(err, "Failed to PollTaskStatus")
	}

	if response.Status == "TERMINAL" {
		logger.V(1).Info("Task Terminated", "response", response)
	} else {
		logger.V(1).Info("Task Completed", "status", response.Status)
	}
}

func (r *ApplicationReconciler) buildDeleteTask(applicationName string, application *applicationV1.Application) spinnaker.Task {
	return spinnaker.Task{
		Application: applicationName,
		Description: "Delete Application: " + applicationName,
		Job: []interface{}{
			spinnaker.ApplicationJob{
				Application: func() map[string]interface{} {
					var m map[string]interface{}
					b, _ := json.Marshal(application.Spec)
					_ = json.Unmarshal(b, &m)
					m["name"] = applicationName
					return m
				}(),
				Type: "deleteApplication",
			},
		},
	}
}

func (r *ApplicationReconciler) buildCreateTask(applicationName string, application *applicationV1.Application) spinnaker.Task {
	return spinnaker.Task{
		Application: applicationName,
		Description: "Create Application: " + applicationName,
		Job: []interface{}{
			spinnaker.ApplicationJob{
				Application: func() map[string]interface{} {
					var m map[string]interface{}
					b, _ := json.Marshal(application.Spec)
					_ = json.Unmarshal(b, &m)
					m["name"] = applicationName
					return m
				}(),
				Type: "createApplication",
			},
		},
	}
}

func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&applicationV1.Application{}).Complete(r)
}
