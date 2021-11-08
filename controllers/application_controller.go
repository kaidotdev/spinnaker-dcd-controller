package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	v1 "spinnaker-dcd-controller/api/v1"
	"time"

	"github.com/spinnaker/roer/spinnaker"

	"github.com/go-logr/logr"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ApplicationCreateTaskType string = "createApplication"
	ApplicationUpdateTaskType string = "updateApplication"
	ApplicationDeleteTaskType string = "deleteApplication"
)

type ApplicationReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	SpinnakerClient spinnaker.Client
}

func (r *ApplicationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	application := &v1.Application{}
	ctx := context.Background()
	logger := r.Log.WithValues("application", req.NamespacedName)
	if err := r.Get(ctx, req.NamespacedName, application); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if application.ObjectMeta.DeletionTimestamp.IsZero() {
		hash := fmt.Sprintf("%x", sha256.Sum256(application.Spec.Raw))
		oldHash := application.Status.Hash
		if oldHash != hash {
			var taskType string
			if oldHash == "" {
				taskType = ApplicationCreateTaskType
			} else {
				taskType = ApplicationUpdateTaskType
			}

			task := r.buildTask(req.Name, application, taskType)
			response, err := r.submitTask(req.Name, task)
			if err != nil {
				return ctrl.Result{}, err
			}
			application.Status.SpinnakerResource.ApplicationName = req.Name
			application.Status.Hash = hash
			application.ObjectMeta.Finalizers = append(application.ObjectMeta.Finalizers, myFinalizerName)
			if response.Status == "TERMINAL" {
				application.Status.Conditions = append(application.Status.Conditions, v1.ApplicationCondition{
					Type:   v1.ApplicationCreationComplete,
					Status: "False",
				})
			} else {
				application.Status.Conditions = append(application.Status.Conditions, v1.ApplicationCondition{
					Type:   v1.ApplicationCreationComplete,
					Status: "True",
				})
				r.Recorder.Eventf(application, coreV1.EventTypeNormal, "SuccessfulCreated", "Created application: %q", req.Name)
				logger.V(1).Info("create", "application", application)
			}

			if err := r.Update(ctx, application); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if containsString(application.ObjectMeta.Finalizers, myFinalizerName) {
			task := r.buildTask(req.Name, application, ApplicationDeleteTaskType)
			response, err := r.submitTask(req.Name, task)
			if err != nil {
				return ctrl.Result{}, err
			}

			application.ObjectMeta.Finalizers = removeString(application.ObjectMeta.Finalizers, myFinalizerName)
			if response.Status == "TERMINAL" {
				application.Status.Conditions = append(application.Status.Conditions, v1.ApplicationCondition{
					Type:   v1.ApplicationDeletionComplete,
					Status: "False",
				})
			} else {
				application.Status.Conditions = append(application.Status.Conditions, v1.ApplicationCondition{
					Type:   v1.ApplicationDeletionComplete,
					Status: "True",
				})
				r.Recorder.Eventf(application, coreV1.EventTypeNormal, "SuccessfulDeleted", "Deleted application: %q", req.Name)
				logger.V(1).Info("delete", "application", application)
			}
			if err := r.Update(ctx, application); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *ApplicationReconciler) submitTask(applicationName string, task spinnaker.Task) (*spinnaker.ExecutionResponse, error) {
	ref, err := r.SpinnakerClient.ApplicationSubmitTask(applicationName, task)
	if err != nil {
		return nil, err
	}
	response, err := r.SpinnakerClient.PollTaskStatus(ref.Ref, 30*time.Second)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (r *ApplicationReconciler) buildTask(applicationName string, application *v1.Application, taskType string) spinnaker.Task {
	return spinnaker.Task{
		Application: applicationName,
		Description: fmt.Sprintf("Execute %s task: %s", taskType, applicationName),
		Job: []interface{}{
			spinnaker.ApplicationJob{
				Application: func() map[string]interface{} {
					var m map[string]interface{}
					_ = json.Unmarshal(application.Spec.Raw, &m)
					m["name"] = applicationName
					return m
				}(),
				Type: taskType,
			},
		},
	}
}

func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1.Application{}).Complete(r)
}
