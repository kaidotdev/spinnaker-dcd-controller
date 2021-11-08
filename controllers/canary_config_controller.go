package controllers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	v1 "spinnaker-dcd-controller/api/v1"

	"github.com/go-logr/logr"
	"github.com/spinnaker/spin/cmd/gateclient"
	gate "github.com/spinnaker/spin/gateapi"
	"golang.org/x/xerrors"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CanaryConfigReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	Recorder        record.EventRecorder
	SpinnakerClient gateclient.GatewayClient
}

func (r *CanaryConfigReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	canaryConfig := &v1.CanaryConfig{}
	ctx := context.Background()
	logger := r.Log.WithValues("canaryConfig", req.NamespacedName)
	if err := r.Get(ctx, req.NamespacedName, canaryConfig); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if canaryConfig.ObjectMeta.DeletionTimestamp.IsZero() {
		hash := fmt.Sprintf("%x", sha256.Sum256(canaryConfig.Spec.Raw))
		oldHash := canaryConfig.Status.Hash
		if hash != oldHash {
			var configJSON map[string]interface{}
			_ = json.Unmarshal(canaryConfig.Spec.Raw, &configJSON)

			if _, exists := configJSON["id"]; !exists {
				return ctrl.Result{}, xerrors.New("required canary config key 'id' missing...")
			}

			if err := r.saveCanaryConfig(configJSON); err != nil {
				return ctrl.Result{}, err
			}

			canaryConfig.Status.SpinnakerResource.Name = configJSON["name"].(string)
			canaryConfig.Status.SpinnakerResource.ID = configJSON["id"].(string)
			canaryConfig.Status.Hash = hash
			if !containsString(canaryConfig.ObjectMeta.Finalizers, myFinalizerName) {
				canaryConfig.ObjectMeta.Finalizers = append(canaryConfig.ObjectMeta.Finalizers, myFinalizerName)
			}
			canaryConfig.Status.Conditions = append(canaryConfig.Status.Conditions, v1.CanaryConfigCondition{
				Type:   v1.CanaryConfigCreationComplete,
				Status: "True",
			})
			r.Recorder.Eventf(canaryConfig, coreV1.EventTypeNormal, "SuccessfulCreated", "Created canary config: %q", req.Name)
			logger.V(1).Info("create", "canary config", canaryConfig)
			if err := r.Update(ctx, canaryConfig); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if containsString(canaryConfig.ObjectMeta.Finalizers, myFinalizerName) {
			if err := r.deleteCanaryConfig(canaryConfig.Status.SpinnakerResource.ID); err != nil {
				return ctrl.Result{}, err
			}
			canaryConfig.Status.Conditions = append(canaryConfig.Status.Conditions, v1.CanaryConfigCondition{
				Type:   v1.CanaryConfigDeletionComplete,
				Status: "True",
			})
			r.Recorder.Eventf(canaryConfig, coreV1.EventTypeNormal, "SuccessfulDeleted", "Deleted canary config: %q", req.Name)
			logger.V(1).Info("delete", "canary config", canaryConfig)

			canaryConfig.ObjectMeta.Finalizers = removeString(canaryConfig.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(ctx, canaryConfig); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *CanaryConfigReconciler) saveCanaryConfig(configJSON map[string]interface{}) error {
	configID := configJSON["id"].(string)

	_, resp, getErr := r.SpinnakerClient.V2CanaryConfigControllerApi.GetCanaryConfigUsingGET(
		r.SpinnakerClient.Context, configID, &gate.V2CanaryConfigControllerApiGetCanaryConfigUsingGETOpts{})

	var saveResp *http.Response
	var saveErr error
	if resp.StatusCode == http.StatusOK {
		_, saveResp, saveErr = r.SpinnakerClient.V2CanaryConfigControllerApi.UpdateCanaryConfigUsingPUT(
			r.SpinnakerClient.Context, configJSON, configID, &gate.V2CanaryConfigControllerApiUpdateCanaryConfigUsingPUTOpts{})
	} else if resp.StatusCode == http.StatusNotFound {
		_, saveResp, saveErr = r.SpinnakerClient.V2CanaryConfigControllerApi.CreateCanaryConfigUsingPOST(
			r.SpinnakerClient.Context, configJSON, &gate.V2CanaryConfigControllerApiCreateCanaryConfigUsingPOSTOpts{})
	} else {
		if getErr != nil {
			return getErr
		}

		return xerrors.Errorf(
			"encountered an unexpected status code %d querying canary config with id %s",
			resp.StatusCode, configID)
	}

	if saveErr != nil {
		return saveErr
	}

	if saveResp.StatusCode != http.StatusOK {
		return xerrors.Errorf(
			"encountered an error saving canary config %v, status code: %d",
			configJSON, saveResp.StatusCode)
	}

	return nil
}

func (r *CanaryConfigReconciler) deleteCanaryConfig(id string) error {
	resp, err := r.SpinnakerClient.V2CanaryConfigControllerApi.DeleteCanaryConfigUsingDELETE(
		r.SpinnakerClient.Context, id, &gate.V2CanaryConfigControllerApiDeleteCanaryConfigUsingDELETEOpts{})
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return xerrors.Errorf(
			"encountered an error deleting canary config, status code: %d", resp.StatusCode)
	}

	return nil
}

func (r *CanaryConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1.CanaryConfig{}).Complete(r)
}
