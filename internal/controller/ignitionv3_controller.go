// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	ignitionConfig "github.com/coreos/ignition/v2/config/v3_5"
	ignitiontypes "github.com/coreos/ignition/v2/config/v3_5/types"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	// "k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	metalv1alpha1 "github.com/cobaltcore-dev/khalkeon/api/v1alpha1"
)

const secretConfigData = "config"

var finalizer = metalv1alpha1.GroupVersion.Group + "/ignitionv3"

// IgnitionV3Reconciler reconciles a IgnitionV3 object
type IgnitionV3Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=metal.cobaltcore.dev,resources=ignitionv3s,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=metal.cobaltcore.dev,resources=ignitionv3s/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=metal.cobaltcore.dev,resources=ignitionv3s/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=list;watch;create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IgnitionV3 object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *IgnitionV3Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	ignition := &metalv1alpha1.IgnitionV3{}
	if err := r.Get(ctx, req.NamespacedName, ignition); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	extraLogs := []any{}
	if cond := meta.FindStatusCondition(ignition.Status.Conditions, metalv1alpha1.SecretType); cond != nil && cond.Status == metav1.ConditionFalse {
		extraLogs = append(extraLogs, "secret status", cond.Message)
	}
	log.Info("Reconcile", extraLogs...)

	isIgnitionCreated := false
	if controllerutil.AddFinalizer(ignition, finalizer) {
		// ignition is created and target ignitions should be triggered
		if err := r.Update(ctx, ignition); err != nil {
			return ctrl.Result{}, fmt.Errorf("couldn't add finalizer: %w", err)
		}
		log.V(1).Info("Finalizer was added")
		isIgnitionCreated = true
	}

	if err := r.patchSecretStatus(ctx, ignition, isIgnitionCreated); err != nil {
		return ctrl.Result{}, fmt.Errorf("couldn't patch secret status: %w", err)
	}

	if !ignition.DeletionTimestamp.IsZero() {
		if controllerutil.RemoveFinalizer(ignition, finalizer) {
			log.V(1).Info("Finalizer was removed")
			return ctrl.Result{}, r.Update(ctx, ignition)
		}
		return ctrl.Result{}, nil
	}

	if err := r.patchConfigurationStatus(ctx, ignition); err != nil {
		return ctrl.Result{}, fmt.Errorf("couldn't patch configuration status: %w", err)
	}

	if ignition.Spec.TargetSecret == nil {
		return ctrl.Result{}, nil
	}

	ignitions := map[string]struct{}{}
	mergedConfig, err := r.createMergedConfig(ctx, ignition, ignitions)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("couldn't create merged configuration: %w", err)
	}

	mergedConfigBytes, err := json.Marshal(mergedConfig)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("couldn't marshal merged configuration: %w", err)
	}

	if err := r.reconcileSecret(ctx, ignition, mergedConfigBytes); err != nil {
		return ctrl.Result{}, fmt.Errorf("couldn't reconcile secret: %w", err)
	}

	if err := r.patchTargetIgnitionsStatus(ctx, ignitions, ignition); err != nil {
		return ctrl.Result{}, fmt.Errorf("couldn't patch target ignitions status: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *IgnitionV3Reconciler) patchSecretStatus(ctx context.Context, reconcileIgnition *metalv1alpha1.IgnitionV3, isIgnitionCreated bool) error {
	if len(reconcileIgnition.Status.TargetIgnitions) == 0 && !isIgnitionCreated {
		return nil
	}
	ignitionList := &metalv1alpha1.IgnitionV3List{}
	if err := r.List(ctx, ignitionList, &client.ListOptions{Namespace: reconcileIgnition.Namespace}); err != nil {
		return err
	}
	condition := metav1.Condition{
		Type:               metalv1alpha1.SecretType,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionFalse,
		Reason:             "MergeResourceChanged",
	}
	for _, ign := range ignitionList.Items {
		if reconcileIgnition.Name != ign.Name && isIgnitionCreated && ign.Spec.TargetSecret != nil {
			condition.Message = fmt.Sprintf("%s was created", client.ObjectKeyFromObject(reconcileIgnition).String())
		} else if slices.ContainsFunc(reconcileIgnition.Status.TargetIgnitions,
			func(ref corev1.LocalObjectReference) bool { return ref.Name == ign.Name }) {
			condition.Message = fmt.Sprintf("%s has changed", client.ObjectKeyFromObject(reconcileIgnition).String())
		} else {
			continue
		}
		if err := r.patchStatusIfNeeded(ctx, &ign, condition); err != nil {
			return err
		}
	}
	return nil
}

func (r *IgnitionV3Reconciler) patchConfigurationStatus(ctx context.Context, ignition *metalv1alpha1.IgnitionV3) error {
	condition := metav1.Condition{
		Type:               metalv1alpha1.ConfigurationType,
		LastTransitionTime: metav1.Now(),
		Status:             metav1.ConditionTrue,
		Reason:             "ConversionSucceeded",
		Message:            "Specification is a valid ignition configuration",
	}

	if _, err := convert(ignition.Spec); err != nil {
		condition.Status = metav1.ConditionFalse
		condition.Reason = "ConversionFailed"
		condition.Message = err.Error()
	}
	return r.patchStatusIfNeeded(ctx, ignition, condition)
}

func (r *IgnitionV3Reconciler) patchStatusIfNeeded(ctx context.Context, ignition *metalv1alpha1.IgnitionV3, condition metav1.Condition) error {
	ignitionBase := ignition.DeepCopy()
	if changed := meta.SetStatusCondition(&ignition.Status.Conditions, condition); changed {
		if err := r.Status().Patch(ctx, ignition, client.MergeFrom(ignitionBase)); err != nil {
			return fmt.Errorf("failed to patch IgnitionV3 status: %w", err)
		}
	}
	return nil
}

func (r *IgnitionV3Reconciler) createMergedConfig(ctx context.Context, ign *metalv1alpha1.IgnitionV3, collectedIgns map[string]struct{}) (ignitiontypes.Config, error) {
	if _, isIgnCollected := collectedIgns[ign.Name]; isIgnCollected {
		return ignitiontypes.Config{}, fmt.Errorf("loop with %s", client.ObjectKeyFromObject(ign).String())
	}
	collectedIgns[ign.Name] = struct{}{}

	if ign.Spec.Ignition.Config.Replace != nil {
		replaceIng := &metalv1alpha1.IgnitionV3{}
		nn := types.NamespacedName{Name: ign.Spec.Ignition.Config.Replace.Name, Namespace: ign.Namespace}
		if err := r.Get(ctx, nn, replaceIng); err != nil {
			return ignitiontypes.Config{}, fmt.Errorf("couldn't get ignition. Reason: %v", err)
		}
		return r.createMergedConfig(ctx, replaceIng, collectedIgns)
	}

	config, err := convert(ign.Spec)
	if err != nil {
		return ignitiontypes.Config{}, fmt.Errorf("couldn't convert ignition spec. Reason: %v", err)
	}

	if ign.Spec.Ignition.Config.Merge != nil {
		selector, err := metav1.LabelSelectorAsSelector(ign.Spec.Ignition.Config.Merge)
		if err != nil {
			return ignitiontypes.Config{}, fmt.Errorf("couldn't convert ignition merge label selector. Reason: %v", err)
		}

		ignitionList := metalv1alpha1.IgnitionV3List{}
		if err := r.List(ctx, &ignitionList, &client.ListOptions{LabelSelector: selector, Namespace: ign.Namespace}); err != nil {
			return ignitiontypes.Config{}, fmt.Errorf("couldn't list ignitions. Reason: %v", err)
		}

		indices := make([]int, len(ignitionList.Items))
		for i := range ignitionList.Items {
			indices[i] = i
		}
		sort.Slice(indices, func(i, j int) bool {
			return ignitionList.Items[i].Name < ignitionList.Items[j].Name
		})

		mergedConfig := ignitiontypes.Config{}
		for _, i := range indices {
			ignition := ignitionList.Items[i] // iterating through ignitions sorted by their name to ensure deterministic output
			cfg, err := r.createMergedConfig(ctx, &ignition, collectedIgns)
			if err != nil {
				return ignitiontypes.Config{}, err
			}
			mergedConfig = ignitionConfig.Merge(mergedConfig, cfg)
		}

		config = ignitionConfig.Merge(config, mergedConfig)
	}

	return config, err
}

func convert(spec metalv1alpha1.IgnitionV3Spec) (ignitiontypes.Config, error) {
	spec.Ignition.Config.Merge = nil
	spec.Ignition.Config.Replace = nil
	spec.TargetSecret = nil

	specByte, err := json.Marshal(spec)
	if err != nil {
		return ignitiontypes.Config{}, fmt.Errorf("couldn't marshal spec. Reason: %v", err)
	}

	cfg, report, err := ignitionConfig.Parse(specByte)
	if err != nil || report.IsFatal() {
		return ignitiontypes.Config{}, fmt.Errorf("couldn't parse spec into coreos ignition config. Error: %v, Report: %s", err, report.String())
	}
	return cfg, nil
}

func (r *IgnitionV3Reconciler) reconcileSecret(ctx context.Context, ignition *metalv1alpha1.IgnitionV3, configBytes []byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ignition.Spec.TargetSecret.Name,
			Namespace: ignition.Namespace,
		},
		Data: map[string][]byte{secretConfigData: configBytes},
	}
	res, err := controllerutil.CreateOrPatch(ctx, r.Client, secret, func() error {
		secret.Data[secretConfigData] = configBytes
		return controllerutil.SetOwnerReference(ignition, secret, r.Scheme)
	})

	if res == controllerutil.OperationResultNone {
		condition := metav1.Condition{
			Type:               metalv1alpha1.SecretType,
			LastTransitionTime: metav1.Now(),
			Status:             metav1.ConditionTrue,
			Reason:             "SecretReady",
		}
		if err := r.patchStatusIfNeeded(ctx, ignition, condition); err != nil {
			return err
		}
	}
	return err
}

func (r *IgnitionV3Reconciler) patchTargetIgnitionsStatus(ctx context.Context, ignitions map[string]struct{}, targetIgnition *metalv1alpha1.IgnitionV3) error {
	ignitionList := &metalv1alpha1.IgnitionV3List{}
	if err := r.List(ctx, ignitionList, &client.ListOptions{Namespace: targetIgnition.Namespace}); err != nil {
		return err
	}

	for _, ignition := range ignitionList.Items {
		if _, wasIgnitionUsedForMerging := ignitions[ignition.Name]; !wasIgnitionUsedForMerging {
			continue
		}
		ref := corev1.LocalObjectReference{Name: targetIgnition.Name}
		if ignition.Name == targetIgnition.Name || slices.Contains(ignition.Status.TargetIgnitions, ref) {
			continue
		}
		ignitionBase := ignition.DeepCopy()
		ignition.Status.TargetIgnitions = append(ignition.Status.TargetIgnitions, ref)
		if err := r.Status().Patch(ctx, &ignition, client.MergeFrom(ignitionBase)); err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IgnitionV3Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&metalv1alpha1.IgnitionV3{}).
		Owns(&corev1.Secret{}).
		Named("ignitionv3").
		Complete(r)
}
