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

	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/extensions/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gitiflytekcomv1 "git.iflytek.com/iflearner-opeartor/api/v1"
)

// IflearnerJobReconciler reconciles a IflearnerJob object
type IflearnerJobReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	apiGVStr = gitiflytekcomv1.GroupVersion.String()
	labelApp = "app"

	podOwnerKey       = ".metadata.controller"
	podGrpcPort int32 = 50001
	podHttpPort int32 = 50002

	configmapName = "server-iflearner-crt"
	configmapFile = "server-iflearner-secret.crt"

	serviceGrpcPort int32 = 80
	serviceHttpPort int32 = 82

	// ingressHost        = ".server.iflearner.com"
	ingressSecretName  = "server-iflearner-secret"
	ingressAnnotations = map[string]string{
		"kubernetes.io/ingress.class":                  "nginx",
		"nginx.ingress.kubernetes.io/ssl-redirect":     "true",
		"nginx.ingress.kubernetes.io/backend-protocol": "GRPC",
		"nginx.ingress.kubernetes.io/proxy-body-size":  "1024m",
	}
)

//+kubebuilder:rbac:groups=git.iflytek.com,resources=iflearnerjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=git.iflytek.com,resources=iflearnerjobs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=git.iflytek.com,resources=iflearnerjobs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps;pods;services;endpoints;persistentvolumeclaims;events,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="extensions",resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the IflearnerJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *IflearnerJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var iflearnerJob gitiflytekcomv1.IflearnerJob
	if err := r.Get(ctx, req.NamespacedName, &iflearnerJob); err != nil {
		log.Error(err, "unable to fetch IflearnerJob")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var pods corev1.PodList
	if err := r.List(ctx, &pods, client.InNamespace(req.Namespace), client.MatchingFields{podOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list pods")
		return ctrl.Result{}, err
	}

	log.Info("list pods", "num", len(pods.Items))
	if len(pods.Items) == 0 {
		// name := fmt.Sprintf("%s-%d", iflearnerJob.Name, time.Now().Unix())
		name := iflearnerJob.Name

		log.Info("create pod")
		pod, err := constructPodForIflearnerJob(&iflearnerJob, r.Scheme, name)
		if err != nil {
			log.Error(err, "unable to construct Pod for IflearnerJob")
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, pod); err != nil {
			log.Error(err, "unable to create Pod for IflearnerJob", "pod", pod)
			return ctrl.Result{}, err
		}

		if iflearnerJob.Spec.Role == gitiflytekcomv1.RoleServer {
			log.Info("create service")
			svc, err := constructServiceForIflearnerJob(&iflearnerJob, r.Scheme, name)
			if err != nil {
				log.Error(err, "unable to construct Service for IflearnerJob")
				return ctrl.Result{}, err
			}

			if err := r.Create(ctx, svc); err != nil {
				log.Error(err, "unable to create Service for IflearnerJob", "service", svc)
				return ctrl.Result{}, err
			}

			log.Info("create ingress")
			ingress, err := constructIngressForIflearnerJob(&iflearnerJob, r.Scheme, name)
			if err != nil {
				log.Error(err, "unable to construct Ingress for IflearnerJob")
				return ctrl.Result{}, err
			}

			if err := r.Create(ctx, ingress); err != nil {
				log.Error(err, "unable to create Ingress for IflearnerJob", "ingress", svc)
				return ctrl.Result{}, err
			}
		}
	} else {
		log.Info("update status", "status", pods.Items[0].Status)
		pods.Items[0].Status.Conditions = make([]corev1.PodCondition, 0)
		iflearnerJob.Status.PodStatus = pods.Items[0].Status.DeepCopy()
		if err := r.Status().Update(ctx, &iflearnerJob); err != nil {
			log.Error(err, "unable to update status for IflearnerJob")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IflearnerJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, podOwnerKey, func(rawObj client.Object) []string {
		// grab the pod object, extract the owner...
		pod := rawObj.(*corev1.Pod)
		owner := metav1.GetControllerOf(pod)
		if owner == nil {
			return nil
		}
		// ...make sure it's a IflearnerJob...
		if owner.APIVersion != apiGVStr || owner.Kind != "IflearnerJob" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&gitiflytekcomv1.IflearnerJob{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

func constructPodForIflearnerJob(iflearnerJob *gitiflytekcomv1.IflearnerJob, scheme *runtime.Scheme, name string) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				labelApp: name,
			},
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   iflearnerJob.Namespace,
		},
		Spec: *iflearnerJob.Spec.Template.Spec.DeepCopy(),
	}
	for k, v := range iflearnerJob.Spec.Template.Annotations {
		pod.Annotations[k] = v
	}
	for k, v := range iflearnerJob.Spec.Template.Labels {
		pod.Labels[k] = v
	}
	if iflearnerJob.Spec.Role == gitiflytekcomv1.RoleClient {
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      configmapName,
			MountPath: "/etc/" + configmapFile,
			SubPath:   configmapFile,
			ReadOnly:  true,
		})
		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name: configmapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: configmapName},
				},
			},
		})
	}

	if err := ctrl.SetControllerReference(iflearnerJob, pod, scheme); err != nil {
		return nil, err
	}

	return pod, nil
}

func constructServiceForIflearnerJob(iflearnerJob *gitiflytekcomv1.IflearnerJob, scheme *runtime.Scheme, name string) (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				labelApp: name,
			},
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   iflearnerJob.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "grpc",
					Port:       serviceGrpcPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(int(podGrpcPort)),
				},
				{
					Name:       "http",
					Port:       serviceHttpPort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(int(podHttpPort)),
				},
			},
			Selector: map[string]string{
				labelApp: name,
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	if err := ctrl.SetControllerReference(iflearnerJob, svc, scheme); err != nil {
		return nil, err
	}

	return svc, nil
}

func constructIngressForIflearnerJob(iflearnerJob *gitiflytekcomv1.IflearnerJob, scheme *runtime.Scheme, name string) (*v1beta1.Ingress, error) {
	pathType := v1beta1.PathTypePrefix
	ingress := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: ingressAnnotations,
			Name:        name,
			Namespace:   iflearnerJob.Namespace,
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: iflearnerJob.Spec.Host,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: v1beta1.IngressBackend{
										ServiceName: name,
										ServicePort: intstr.FromInt(int(serviceGrpcPort)),
									},
								},
							},
						},
					},
				},
			},
			TLS: []v1beta1.IngressTLS{
				{
					Hosts:      []string{iflearnerJob.Spec.Host},
					SecretName: ingressSecretName,
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(iflearnerJob, ingress, scheme); err != nil {
		return nil, err
	}

	return ingress, nil
}
