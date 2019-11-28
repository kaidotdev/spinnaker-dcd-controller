package controllers

import (
	"context"
	"encoding/json"
	v1 "spinnaker-dcd-controller/api/v1"

	"github.com/spinnaker/roer"

	"github.com/mitchellh/mapstructure"

	"github.com/spinnaker/roer/spinnaker"

	"github.com/go-logr/logr"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PipelineReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	SpinnakerClient spinnaker.Client
}

func (r *PipelineReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	pipeline := &v1.Pipeline{}
	ctx := context.Background()
	logger := r.Log.WithValues("pipeline", req.NamespacedName)
	if err := r.Get(ctx, req.NamespacedName, pipeline); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if pipeline.Status.SpinnakerResource.ApplicationName == "" && pipeline.Status.SpinnakerResource.ID == "" {
		applicationName, id, err := r.savePipeline(pipeline, logger)
		if err != nil {
			return ctrl.Result{}, err
		}
		pipeline.Status.SpinnakerResource.ApplicationName = *applicationName
		pipeline.Status.SpinnakerResource.ID = *id
		pipeline.Status.Phase = "Deployed"

		if err := r.Update(context.Background(), pipeline); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Eventf(pipeline, coreV1.EventTypeNormal, "SuccessfulCreated", "Created pipeline: %q", req.Name)
		logger.V(1).Info("create", "pipeline", pipeline)
	}

	if pipeline.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(pipeline.ObjectMeta.Finalizers, myFinalizerName) {
			pipeline.ObjectMeta.Finalizers = append(pipeline.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), pipeline); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if containsString(pipeline.ObjectMeta.Finalizers, myFinalizerName) {
			if err := r.deletePipeline(pipeline, logger); err != nil {
				return ctrl.Result{}, err
			}

			pipeline.ObjectMeta.Finalizers = removeString(pipeline.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), pipeline); err != nil {
				return ctrl.Result{}, err
			}
			r.Recorder.Eventf(pipeline, coreV1.EventTypeNormal, "SuccessfulDeleted", "Deleted pipeline: %q", req.Name)
			logger.V(1).Info("delete", "pipeline", pipeline)
		}

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *PipelineReconciler) savePipeline(pipeline *v1.Pipeline, logger logr.Logger) (*string, *string, error) {
	var roerConfiguration roer.PipelineConfiguration
	if err := mapstructure.Decode(func() map[string]interface{} {
		var m map[string]interface{}
		_ = json.Unmarshal(pipeline.Spec, &m)
		return m
	}(), &roerConfiguration); err != nil {
		return nil, nil, err
	}

	if err := r.SpinnakerClient.SavePipelineConfig(roerConfiguration.ToClient()); err != nil {
		return nil, nil, err
	}
	return &roerConfiguration.Pipeline.Application, &roerConfiguration.Pipeline.PipelineConfigID, nil
}

func (r *PipelineReconciler) deletePipeline(pipeline *v1.Pipeline, logger logr.Logger) error {
	return r.SpinnakerClient.DeletePipeline(pipeline.Status.SpinnakerResource.ApplicationName, pipeline.Status.SpinnakerResource.ID)
}

func (r *PipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1.Pipeline{}).Complete(r)
}
