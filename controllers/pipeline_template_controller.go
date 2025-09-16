package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	v1 "spinnaker-dcd-controller/api/v1"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/spinnaker/roer/spinnaker"
	"golang.org/x/xerrors"

	"github.com/go-logr/logr"
	coreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if pipelineTemplate.ObjectMeta.DeletionTimestamp.IsZero() {
		hash := fmt.Sprintf("%x", sha256.Sum256(pipelineTemplate.Spec.Raw))
		oldHash := pipelineTemplate.Status.Hash
		if hash != oldHash {
			id, response, err := r.publishTemplate(pipelineTemplate)
			if err != nil {
				if errors.Is(err, valueIsNotFoundError) {
					return ctrl.Result{RequeueAfter: 60 * time.Second}, err
				}
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
	processedYAML, err := r.processTemplateVariables(pipelineTemplate.Spec.Raw)
	if err != nil {
		return "", nil, xerrors.Errorf("failed to process template variables: %w", err)
	}

	var templateMap map[string]interface{}
	_ = json.Unmarshal(processedYAML, &templateMap)

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

var valueIsNotFoundError = errors.New("value is not found")

func (r *PipelineTemplateReconciler) processTemplateVariables(yamlData []byte) (result []byte, err error) {
	variablePattern := regexp.MustCompile(`\$\{([^}]+)\}`)

	result = variablePattern.ReplaceAllFunc(yamlData, func(match []byte) []byte {
		expression := string(match[2 : len(match)-1])

		if strings.HasPrefix(expression, "ImportValue:") {
			exportName := strings.TrimPrefix(expression, "ImportValue:")
			exportValue, ierr := r.getCloudFormationExportValue(exportName)
			if ierr != nil {
				err = ierr
				return match
			}
			return []byte(exportValue)
		}

		return match
	})

	return result, err
}

func (r *PipelineTemplateReconciler) getCloudFormationExportValue(exportName string) (string, error) {
	ctx := context.Background()

	configuration, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", xerrors.Errorf("failed to load AWS configuration: %w", err)
	}
	c := cloudformation.NewFromConfig(configuration)

	input := &cloudformation.ListExportsInput{}
	for {
		output, err := c.ListExports(ctx, input)
		if err != nil {
			return "", xerrors.Errorf("failed to list CloudFormation exports: %w", err)
		}

		for _, export := range output.Exports {
			if export.Name != nil && *export.Name == exportName {
				if export.Value != nil {
					return *export.Value, nil
				}
				return "", xerrors.Errorf("export value is nil for name: %s", exportName)
			}
		}

		if output.NextToken == nil {
			break
		}
		input.NextToken = output.NextToken
	}

	return "", xerrors.Errorf("CloudFormation export value not found for name %s: %w", exportName, valueIsNotFoundError)
}

func (r *PipelineTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1.PipelineTemplate{}).Complete(r)
}
