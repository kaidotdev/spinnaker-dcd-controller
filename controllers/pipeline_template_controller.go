package controllers

import (
	"context"
	"encoding/json"
	v1 "spinnaker-dcd-controller/api/v1"

	"github.com/google/uuid"
	"github.com/spinnaker/roer/spinnaker"

	"github.com/go-logr/logr"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PipelineTemplateReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	SpinnakerClient spinnaker.Client
}

func (r *PipelineTemplateReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	pipelineTemplate := &v1.PipelineTemplate{}
	logger := r.Log.WithValues("pipelineTemplate", req.NamespacedName)
	ctx := context.Background()
	if err := r.Get(ctx, req.NamespacedName, pipelineTemplate); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if pipelineTemplate.Status.SpinnakerResource.ID == "" {
		id, err := r.publishTemplate(pipelineTemplate, logger)
		if err != nil {
			return ctrl.Result{}, err
		}
		pipelineTemplate.Status.SpinnakerResource.ID = *id
		pipelineTemplate.Status.Phase = "Deployed"

		if err := r.Update(ctx, pipelineTemplate); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Eventf(pipelineTemplate, coreV1.EventTypeNormal, "SuccessfulCreated", "Created pipeline template: %q", req.Name)
		logger.V(1).Info("create", "pipeline template", pipelineTemplate)
	}

	if pipelineTemplate.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(pipelineTemplate.ObjectMeta.Finalizers, myFinalizerName) {
			pipelineTemplate.ObjectMeta.Finalizers = append(pipelineTemplate.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(ctx, pipelineTemplate); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if containsString(pipelineTemplate.ObjectMeta.Finalizers, myFinalizerName) {
			if err := r.deleteTemplate(pipelineTemplate, logger); err != nil {
				return ctrl.Result{}, err
			}

			pipelineTemplate.ObjectMeta.Finalizers = removeString(pipelineTemplate.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(ctx, pipelineTemplate); err != nil {
				return ctrl.Result{}, err
			}
			r.Recorder.Eventf(pipelineTemplate, coreV1.EventTypeNormal, "SuccessfulDeleted", "Deleted pipeline template: %q", req.Name)
			logger.V(1).Info("delete", "pipeline template", pipelineTemplate)
		}

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *PipelineTemplateReconciler) publishTemplate(pipelineTemplate *v1.PipelineTemplate, logger logr.Logger) (*string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	id := u.String()

	ref, err := r.SpinnakerClient.PublishTemplate(func() map[string]interface{} {
		var m map[string]interface{}
		_ = json.Unmarshal(pipelineTemplate.Spec, &m)
		return m
	}(), spinnaker.PublishTemplateOptions{
		TemplateID: id,
	})
	if err != nil {
		return nil, err
	}

	response, err := r.SpinnakerClient.PollTaskStatus(ref.Ref, 0)
	if err != nil {
		return nil, err
	}

	if response.Status == "TERMINAL" {
		logger.V(1).Info("Task Terminated", "response", response)
	} else {
		logger.V(1).Info("Task Completed", "status", response.Status)
	}

	return &id, nil
}

func (r *PipelineTemplateReconciler) deleteTemplate(pipelineTemplate *v1.PipelineTemplate, logger logr.Logger) error {
	id := pipelineTemplate.Status.SpinnakerResource.ID

	ref, err := r.SpinnakerClient.DeleteTemplate(id)
	if err != nil {
		return err
	}

	response, err := r.SpinnakerClient.PollTaskStatus(ref.Ref, 0)
	if err != nil {
		return err
	}

	if response.Status == "TERMINAL" {
		logger.V(1).Info("Task Terminated", "response", response)
	} else {
		logger.V(1).Info("Task Completed", "status", response.Status)
	}

	return nil
}

func (r *PipelineTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1.PipelineTemplate{}).Complete(r)
}
