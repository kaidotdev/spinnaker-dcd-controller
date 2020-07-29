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

	if pipelineTemplate.ObjectMeta.DeletionTimestamp.IsZero() {
		hash := fmt.Sprintf("%x", sha256.Sum256(pipelineTemplate.Spec))
		oldHash := pipelineTemplate.Status.Hash
		if hash != oldHash {
			id, response, err := r.publishTemplate(pipelineTemplate)
			if err != nil {
				return ctrl.Result{}, err
			}
			pipelineTemplate.Status.SpinnakerResource.ID = id
			pipelineTemplate.Status.Hash = hash
			if !containsString(pipelineTemplate.ObjectMeta.Finalizers, myFinalizerName) {
				pipelineTemplate.ObjectMeta.Finalizers = append(pipelineTemplate.ObjectMeta.Finalizers, myFinalizerName)
			}
			if response.Status == "TERMINAL" {
				pipelineTemplate.Status.Conditions = append(pipelineTemplate.Status.Conditions, v1.PipelineTemplateCondition{
					Type:   v1.PipelineTemplatePublishingComplete,
					Status: "False",
				})
			} else {
				pipelineTemplate.Status.Conditions = append(pipelineTemplate.Status.Conditions, v1.PipelineTemplateCondition{
					Type:   v1.PipelineTemplatePublishingComplete,
					Status: "True",
				})
				r.Recorder.Eventf(pipelineTemplate, coreV1.EventTypeNormal, "SuccessfulPublished", "Published pipeline template: %q", req.Name)
				logger.V(1).Info("publish", "pipeline template", pipelineTemplate)
			}

			if err := r.Update(ctx, pipelineTemplate); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if containsString(pipelineTemplate.ObjectMeta.Finalizers, myFinalizerName) {
			response, err := r.deleteTemplate(pipelineTemplate)
			if err != nil {
				return ctrl.Result{}, err
			}
			if response.Status == "TERMINAL" {
				pipelineTemplate.Status.Conditions = append(pipelineTemplate.Status.Conditions, v1.PipelineTemplateCondition{
					Type:   v1.PipelineTemplateDeletionComplete,
					Status: "False",
				})
			} else {
				pipelineTemplate.Status.Conditions = append(pipelineTemplate.Status.Conditions, v1.PipelineTemplateCondition{
					Type:   v1.PipelineTemplateDeletionComplete,
					Status: "True",
				})
				r.Recorder.Eventf(pipelineTemplate, coreV1.EventTypeNormal, "SuccessfulDeleted", "Deleted pipeline template: %q", req.Name)
				logger.V(1).Info("delete", "pipeline template", pipelineTemplate)
			}

			pipelineTemplate.ObjectMeta.Finalizers = removeString(pipelineTemplate.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(ctx, pipelineTemplate); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *PipelineTemplateReconciler) publishTemplate(pipelineTemplate *v1.PipelineTemplate) (string, *spinnaker.ExecutionResponse, error) {
	templateMap := func() map[string]interface{} {
		var m map[string]interface{}
		_ = json.Unmarshal(pipelineTemplate.Spec, &m)
		return m
	}()
	id := templateMap["id"].(string)
	ref, err := r.SpinnakerClient.PublishTemplate(templateMap, spinnaker.PublishTemplateOptions{
		TemplateID: id,
	})
	if err != nil {
		return "", nil, err
	}

	response, err := r.SpinnakerClient.PollTaskStatus(ref.Ref, 30*time.Second)
	if err != nil {
		return "", nil, err
	}

	return id, response, nil
}

func (r *PipelineTemplateReconciler) deleteTemplate(pipelineTemplate *v1.PipelineTemplate) (*spinnaker.ExecutionResponse, error) {
	id := pipelineTemplate.Status.SpinnakerResource.ID

	ref, err := r.SpinnakerClient.DeleteTemplate(id)
	if err != nil {
		return nil, err
	}

	response, err := r.SpinnakerClient.PollTaskStatus(ref.Ref, 30*time.Second)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (r *PipelineTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1.PipelineTemplate{}).Complete(r)
}
