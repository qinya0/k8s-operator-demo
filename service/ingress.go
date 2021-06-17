package service

import (
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	appv1 "github.com/qinya0/k8s-operator-demo/api/v1"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type AppIngress struct {
	BaseService
	Old *v1beta1.Ingress
}

func (a *AppIngress) Name() string {
	return "Ingress"
}

func (a *AppIngress) Init(client *client.Client, appService *appv1.AppService, ctx *context.Context, request reconcile.Request, log *logr.Logger) error {
	_ = a.BaseService.Init(client, appService, ctx, request, log)

	// get ing
	ing := &v1beta1.Ingress{}
	if err := (*a.BaseService.Client).Get(context.TODO(), request.NamespacedName, ing); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// not found
		a.Old = nil
	} else {
		a.Old = ing
		if a.Old.Annotations == nil {
			a.Old.Annotations = map[string]string{}
		}
	}
	return nil
}
func (a *AppIngress) Configure() bool {
	return a.App.Spec.Ingress.Host != "" && a.App.Spec.Ingress.Path != "" && a.App.Spec.Port.Port != 0
}
func (a *AppIngress) Exist() (bool, error) {
	return a.Old != nil, nil
}
func (a *AppIngress) Create() error {
	ing := a.newIngress()
	if err := (*a.Client).Create(context.TODO(), ing); err != nil {
		return err
	}
	(*a.Log).Info("create ing:" + a.App.Name)
	return nil
}
func (a *AppIngress) NeedUpdate() (bool, error) {
	if a.Old == nil {
		return false, errors.NewServiceUnavailable("no old ing,can't update")
	}
	ing := a.newIngress()

	oldSpec := appv1.AppServiceSpec{}
	if oldSpecStr, ok := a.Old.Annotations[AnnotationName]; ok && oldSpecStr != "" {
		if err := json.Unmarshal([]byte(oldSpecStr), &oldSpec); err != nil {
			return false, err
		}
	}

	if !reflect.DeepEqual(ing.Spec, oldSpec) {
		return true, nil
	}
	return false, nil
}
func (a *AppIngress) Update() error {
	ing := a.newIngress()
	a.Old.Spec = ing.Spec
	if err := (*a.Client).Update(context.TODO(), a.Old); err != nil {
		return err
	}
	(*a.Log).Info("update ing")
	return nil
}
func (a *AppIngress) newIngress() *v1beta1.Ingress {
	spec := v1beta1.IngressSpec{
		Rules: []v1beta1.IngressRule{
			{
				Host: a.App.Spec.Ingress.Host,
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{
							{
								Path: a.App.Spec.Ingress.Path,
								Backend: v1beta1.IngressBackend{
									ServiceName: a.App.Name,
									ServicePort: a.App.Spec.Port.TargetPort,
								},
							},
						},
					},
				},
			},
		},
	}

	var annotations map[string]string
	if a.Old != nil {
		annotations = a.generateAnnotations(spec, a.Old.Annotations)
	} else {
		annotations = a.generateAnnotations(spec, nil)
	}

	return &v1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "networking.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        a.App.Name,
			Namespace:   a.App.Namespace,
			Annotations: annotations,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(a.App, schema.GroupVersionKind{
					Group:   v1.SchemeGroupVersion.Group,
					Version: v1.SchemeGroupVersion.Version,
					Kind:    "AppService",
				}),
			},
		},
		Spec: spec,
	}
}
