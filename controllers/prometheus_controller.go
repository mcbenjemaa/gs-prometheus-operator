/*
Copyright 2022.

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

package controllers

import (
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrltypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	crlog "sigs.k8s.io/controller-runtime/pkg/log"

	monitoringv1alpha1 "github.com/mcbenjemaa/gs-prometheus-operator/api/v1alpha1"
)

// PrometheusReconciler reconciles a Prometheus object
type PrometheusReconciler struct {
	client.Client
	recorder record.EventRecorder

	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=monitoring.giantswarm.io,resources=prometheuses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.giantswarm.io,resources=prometheuses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.giantswarm.io,resources=prometheuses/finalizers,verbs=update

//+kubebuilder:rbac:groups=apps,resources=StatefulSets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=StatefulSets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=StatefulSet/finalizers,verbs=update

//+kubebuilder:rbac:groups=,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=,resources=services/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=,resources=services/finalizers,verbs=update

//+kubebuilder:rbac:groups=,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=,resources=configmaps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=,resources=configmaps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Prometheus object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *PrometheusReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := crlog.FromContext(ctx)

	// Retrieve Prometheus object
	var prometheus monitoringv1alpha1.Prometheus
	if err := r.Get(ctx, req.NamespacedName, &prometheus); err != nil {
		log.Error(err, "unable to fetch Prometheus")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// ensurePrometheus
	err := r.ensurePrometheus(ctx, &prometheus)
	if err != nil {
		log.Error(err, "unable to ensure Prometheus %v")
		r.recorder.Eventf(&prometheus, core.EventTypeWarning, "FailedInitializePrometheus", "error initializing prometheus, %v", err)
	}

	return ctrl.Result{}, nil
}

// ensurePrometheus ensures Prometheus(Statefulset, Service, ConfigMap) exists
func (r *PrometheusReconciler) ensurePrometheus(ctx context.Context, p *monitoringv1alpha1.Prometheus) error {

	err := r.reconcileStatefulSet(ctx, p)
	if err != nil {
		return fmt.Errorf("unable to reconcile StatefulSet: %v", err)
	}

	err = r.reconcileService(ctx, p)
	if err != nil {
		return fmt.Errorf("unable to reconcile Service: %v", err)
	}

	err = r.reconcileConfigMap(ctx, p)
	if err != nil {
		return fmt.Errorf("unable to reconcile ConfigMap: %v", err)
	}

	return nil
}

func (r *PrometheusReconciler) reconcileStatefulSet(ctx context.Context, p *monitoringv1alpha1.Prometheus) error {
	log := crlog.FromContext(ctx)

	// Retrieve StatefulSet
	var sts appsv1.StatefulSet
	nn := ctrltypes.NamespacedName{Namespace: p.ObjectMeta.Namespace, Name: p.ObjectMeta.Name}
	if err := r.Get(ctx, nn, &sts); err != nil {
		log.Info("unable to get StatefulSet")
		desiredSts := desiredStatefulSet(p)
		if apierrors.IsNotFound(err) {
			// Create StatefulSet
			if err := ctrl.SetControllerReference(p, &desiredSts, r.Scheme); err != nil {
				return err
			}
			if err := r.Create(ctx, &desiredSts); err != nil {
				return err
			} else if err == nil {
				r.recorder.Eventf(p, core.EventTypeNormal, "PrometheusStatefulSetCreated", "StatefulSet %v is created", p.Name)
			}
		}
		return err
	} else {

		desiredSts := desiredStatefulSet(p)
		// Update Prometheus Status
		// TODO: readyReplicas
		if p.Status.ReadyReplicas != sts.Status.ReadyReplicas {
			p.Status.ReadyReplicas = sts.Status.ReadyReplicas
			err := r.Status().Update(ctx, p)
			if err != nil {
				return err
			}
		}
		// Check Diff & Update StatefulSet
		if !cmp.Equal(sts.Spec, desiredSts.Spec) {
			log.Info("Update Prometheus StatefulSet")

			return r.Update(ctx, &desiredSts)
		}

	}
	return nil
}

func (r *PrometheusReconciler) reconcileService(ctx context.Context, p *monitoringv1alpha1.Prometheus) error {

	// Retrieve Service
	var svc core.Service
	nn := ctrltypes.NamespacedName{Namespace: p.ObjectMeta.Namespace, Name: p.ObjectMeta.Name}
	if err := r.Get(ctx, nn, &svc); err != nil {
		desiredSvc := desiredService(p)
		if apierrors.IsNotFound(err) {
			// Create StatefulSet
			if err := ctrl.SetControllerReference(p, &desiredSvc, r.Scheme); err != nil {
				return err
			}
			if err := r.Create(ctx, &desiredSvc); err != nil {
				return err
			} else if err == nil {
				r.recorder.Eventf(p, core.EventTypeNormal, "PrometheusServiceCreated", "Service %v is created", p.Name)
			}
		}
		return err
	} else {
		desiredSvc := desiredService(p)
		// Check Diff & Update Service
		if !cmp.Equal(svc.Spec, desiredSvc.Spec) {
			//TODO: Ignore:::
			//return r.Update(ctx, &desiredSvc)
			return nil
		}
	}
	return nil
}

func (r *PrometheusReconciler) reconcileConfigMap(ctx context.Context, p *monitoringv1alpha1.Prometheus) error {
	log := crlog.FromContext(ctx)

	// reconcile Prometheus ConfigMap
	var cm core.ConfigMap
	nn := ctrltypes.NamespacedName{Namespace: p.ObjectMeta.Namespace, Name: p.ObjectMeta.Name + prometheusConfigMapSuffix}
	if err := r.Get(ctx, nn, &cm); err != nil {
		log.Info("unable to get ConfigMap")
		if apierrors.IsNotFound(err) {
			desiredCm, err := desiredPrometheusConfigMap(p)
			if err != nil {
				return err
			}
			// Create ConfigMap
			if err := ctrl.SetControllerReference(p, &desiredCm, r.Scheme); err != nil {
				return err
			}
			if err := r.Create(ctx, &desiredCm); err != nil {
				return err
			} else if err == nil {
				r.recorder.Eventf(p, core.EventTypeNormal, "PrometheusConfigCreated", "ConfigMap %v is created", p.Name)
			}
		} else {
			return err
		}
	} else {
		desiredCm, err := desiredPrometheusConfigMap(p)
		if err != nil {
			return err
		}
		// Check Diff & Update
		if !cmp.Equal(cm.Data, desiredCm.Data) {
			return r.Update(ctx, &desiredCm)
		}
	}

	// reconcile targets ConfigMap
	tcmName := p.ObjectMeta.Name + prometheusConfigMapTargetsSuffix
	var tcm core.ConfigMap
	nncm := ctrltypes.NamespacedName{Namespace: p.ObjectMeta.Namespace, Name: tcmName}
	if err := r.Get(ctx, nncm, &tcm); err != nil {
		log.Info("unable to get Targets ConfigMap")
		if apierrors.IsNotFound(err) {
			desiredCm, err := desiredTargetsConfigMap(p)
			if err != nil {
				return err
			}
			// Create ConfigMap
			if err := ctrl.SetControllerReference(p, &desiredCm, r.Scheme); err != nil {
				return err
			}
			if err := r.Create(ctx, &desiredCm); err != nil {
				return err
			} else if err == nil {
				r.recorder.Eventf(p, core.EventTypeNormal, "TargetsConfigCreated", "ConfigMap %v is created", tcmName)
			}
		}
		return err
	} else {
		desiredCm, err := desiredTargetsConfigMap(p)
		if err != nil {
			return err
		}
		// Check Diff & Update
		if !cmp.Equal(tcm.Data, desiredCm.Data) {
			log.Info("Update targets ConfigMap")
			return r.Update(ctx, &desiredCm)
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrometheusReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("gs-prometheus-operator")

	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.Prometheus{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&core.Service{}).
		Owns(&core.ConfigMap{}).
		Complete(r)
}
