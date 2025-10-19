package controller

import (
	"context"
	"fmt"
	"time"

	autoscalev1 "github.com/Chen-Si-An/mongorouter-autoscaler/api/v1alpha1"
	"github.com/Chen-Si-An/mongorouter-autoscaler/pkg/promclient"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type MongoRouterAutoscalerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=autoscale.mongodb.io,resources=mongorouterautoscalers,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=autoscale.mongodb.io,resources=mongorouterautoscalers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch

func (r *MongoRouterAutoscalerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	mra := &autoscalev1.MongoRouterAutoscaler{}
	if err := r.Get(ctx, req.NamespacedName, mra); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var dep appsv1.Deployment
	if err := r.Get(ctx, types.NamespacedName{
		Name: mra.Spec.TargetRef.Name, Namespace: mra.Spec.TargetRef.Namespace,
	}, &dep); err != nil {
		log.Error(err, "cannot get target deployment")
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	prom, err := promclient.NewPromClient(mra.Spec.Prometheus.URL)
	if err != nil {
		log.Error(err, "failed to init Prometheus client")
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	avgCPU, err := prom.QueryAvgCPU(ctx, mra.Spec.TargetRef.Namespace, mra.Spec.TargetRef.Name, mra.Spec.Policy.Window)
	if err != nil {
		log.Error(err, "prometheus query failed")
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	curr := int32(1)
	if dep.Spec.Replicas != nil {
		curr = *dep.Spec.Replicas
	}

	min := int32(mra.Spec.ScaleBounds.MinReplicas)
	max := int32(mra.Spec.ScaleBounds.MaxReplicas)
	target := float64(mra.Spec.Policy.CpuTargetPercent)
	tol := float64(mra.Spec.Policy.TolerancePercent)
	step := int32(mra.Spec.Policy.Step)
	cooldown := time.Duration(mra.Spec.Policy.CooldownSeconds) * time.Second

	if time.Since(mra.Status.LastScaleTime.Time) < cooldown {
		return ctrl.Result{RequeueAfter: 45 * time.Second}, nil
	}

	var desired = curr
	switch {
	case avgCPU > target+tol && curr < max:
		desired = minInt32(curr+step, max)
		log.Info("Scaling UP mongos routers", "cpu", avgCPU, "old", curr, "new", desired)
	case avgCPU < target-tol && curr > min:
		desired = maxInt32(curr-step, min)
		log.Info("Scaling DOWN mongos routers", "cpu", avgCPU, "old", curr, "new", desired)
	default:
		log.Info("No scaling action", "cpu", avgCPU, "replicas", curr)
	}

	if desired != curr {
		dep.Spec.Replicas = &desired
		if err := r.Update(ctx, &dep); err != nil {
			log.Error(err, "failed to update deployment replicas")
		} else {
			mra.Status.LastScaleTime = metav1.Now()
			mra.Status.LastObservedCPU = fmt.Sprintf("%f", avgCPU)
			mra.Status.LastDesiredReplicas = desired
			_ = r.Status().Update(ctx, mra)
		}
	} else {
		mra.Status.LastObservedCPU = fmt.Sprintf("%f", avgCPU)
		_ = r.Status().Update(ctx, mra)
	}

	return ctrl.Result{RequeueAfter: 45 * time.Second}, nil
}

func (r *MongoRouterAutoscalerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&autoscalev1.MongoRouterAutoscaler{}).
		Complete(r)
}

func minInt32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
func maxInt32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
