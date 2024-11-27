/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"

	ignitionConfig "github.com/coreos/ignition/v2/config/v3_5"
	ignitiontypes "github.com/coreos/ignition/v2/config/v3_5/types"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	metalv1alpha1 "github.com/cobaltcore-dev/khalkeon/api/v1alpha1"
)

// IgnitionV3Reconciler reconciles a IgnitionV3 object
type IgnitionV3Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=metal.cobaltcore.dev,resources=ignitionv3s,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=metal.cobaltcore.dev,resources=ignitionv3s/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=metal.cobaltcore.dev,resources=ignitionv3s/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;create;update;delete

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
	_ = log.FromContext(ctx)

	ignition := &metalv1alpha1.IgnitionV3{}
	if err := r.Get(ctx, req.NamespacedName, ignition); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !ignition.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	if err := r.setStatus(ctx, ignition); err != nil {
		return ctrl.Result{}, err
	}

	//TODO fix recurrence labels and replace
	ignitions, err := r.getIgnitions(ctx, &ignition.Spec.Ignition.Config.Merge)
	if err != nil {
		return ctrl.Result{}, err
	}

	mergedConfigBytes, err := r.mergeIgnitionConfig(ignitions)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileSecret(ctx, ignition, mergedConfigBytes); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *IgnitionV3Reconciler) setStatus(ctx context.Context, ignition *metalv1alpha1.IgnitionV3) error {
	ignitionBase := ignition.DeepCopy()
	condition := metav1.Condition{
		Type: "valid configuration",
	}
	if _, err := convert(ignition.Spec); err != nil {
		condition.Status = metav1.ConditionTrue
	} else {
		condition.Status = metav1.ConditionFalse
	}
	if changed := meta.SetStatusCondition(&ignition.Status.Conditions, condition); changed {
		if err := r.Status().Patch(ctx, ignition, client.MergeFrom(ignitionBase)); err != nil {
			return fmt.Errorf("failed to patch IgnitionV3 status: %w", err)
		}
	}
	return nil
}

// type IgnitionV3List = []metalcobaltcoredevv1alpha1.IgnitionV3

// func Merge(list1, list2 IgnitionV3List) (IgnitionV3List, bool) {
//  ret := list1
//  for _, ignition2 := range list2 {
//      if slices.IndexFunc(list1, func(ignition1 metalcobaltcoredevv1alpha1.IgnitionV3) bool {
//          return reflect.DeepEqual(ignition1.Spec.Ignition.Config.Merge, ignition2.Spec.Ignition.Config.Merge)
//      }) == -1 {
//          ret = append(ret, ignition2)
//      }
//  }
//  return ret, len(ret) > max(len(list1), len(list2))
// }

func (r *IgnitionV3Reconciler) getIgnitions(ctx context.Context, labelSelector *metav1.LabelSelector) ([]metalv1alpha1.IgnitionV3, error) {
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, err
	}
	// labels.Eq
	// req, _ := selector.Requirements()
	// combinedSelector := labels.NewSelector()
	ignitionList := metalv1alpha1.IgnitionV3List{}
	if err := r.List(ctx, &ignitionList, &client.ListOptions{LabelSelector: selector}); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	// for _, ignition := range ignitionList.Items {
	//  if !reflect.DeepEqual(ignition.Spec.Ignition.Config.Merge, *labelSelector) {

	//  }
	// }
	return ignitionList.Items, nil
}

func (r *IgnitionV3Reconciler) mergeIgnitionConfig(ignitions []metalv1alpha1.IgnitionV3) ([]byte, error) {
	indices := make([]int, len(ignitions))
	for i := range ignitions {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return ignitions[i].Name < ignitions[j].Name
	})

	mergedSpec := ignitiontypes.Config{}
	for _, i := range indices {
		ignition := ignitions[i] // iterating through ignitions sorted by their name to ensure deterministic output
		cfg, err := convert(ignition.Spec)
		if err != nil {
			return []byte{}, fmt.Errorf("couldn't convert spec of %s. Reason: %v", client.ObjectKeyFromObject(&ignition).String(), err)
		}

		mergedSpec = ignitionConfig.Merge(mergedSpec, cfg)
	}

	return json.Marshal(mergedSpec)
}

func convert(spec metalv1alpha1.IgnitionV3Spec) (ignitiontypes.Config, error) {
	spec.Ignition.Config.Merge = metav1.LabelSelector{}
	spec.Ignition.Config.Replace = v1.LocalObjectReference{}
	spec.TargetSecret = v1.LocalObjectReference{}

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

func (r *IgnitionV3Reconciler) reconcileSecret(ctx context.Context, ignition *metalv1alpha1.IgnitionV3, cofigBytes []byte) error {
	if ignition.Spec.TargetSecret.Name == "" {
		return nil
	}

	secret, err := r.buildSecret(ctx, ignition, cofigBytes)
	if err != nil {
		return err
	}

	return r.createOrUpdate(ctx, secret, ignition)
}

func (r *IgnitionV3Reconciler) buildSecret(ctx context.Context, ignition *metalv1alpha1.IgnitionV3, cofigBytes []byte) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ignition.Spec.TargetSecret.Name,
			Namespace: ignition.Namespace,
		},
		Data: map[string][]byte{"config": cofigBytes},
	}
	return secret, nil
}

func (r *IgnitionV3Reconciler) createOrUpdate(ctx context.Context, secret *corev1.Secret, owner metav1.Object) error {
	foundSecret := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(secret), foundSecret); apierrors.IsNotFound(err) {
		controllerutil.SetOwnerReference(owner, secret, r.Scheme)
		if err = r.Create(ctx, secret); err != nil {
			return fmt.Errorf("couldn't create a secrete %s. Reason: %w", client.ObjectKeyFromObject(secret).String(), err)
		}

	} else if err == nil && bytes.Compare(foundSecret.Data["config"], secret.Data["config"]) != 0 {
		foundSecret.Data = secret.Data
		if err = r.Update(ctx, foundSecret); err != nil {
			return fmt.Errorf("couldn't update a secrete %s. Reason: %w", client.ObjectKeyFromObject(secret).String(), err)
		}
	} else {
		return err
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
