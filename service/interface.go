package service

import (
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	appv1 "github.com/qinya0/k8s-operator-demo/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type AppInterface interface {
	Name() string
	Init(Client *client.Client, appService *appv1.AppService, ctx *context.Context, request reconcile.Request, log *logr.Logger) error
	Configure() bool // 是否需要创建和更新
	Exist() (bool, error)
	Create() error
	NeedUpdate() (bool, error)
	Update() error
	Delete() error // todo 同步删除
}

type BaseService struct {
	App     *appv1.AppService
	Client  *client.Client
	Ctx     *context.Context
	Request reconcile.Request
	Log     *logr.Logger
}

func (a *BaseService) Name() string {
	return "BaseService"
}

func (a *BaseService) Init(client *client.Client, appService *appv1.AppService, ctx *context.Context, request reconcile.Request, log *logr.Logger) error {
	a.Client = client
	a.Ctx = ctx
	a.App = appService
	a.Request = request
	a.Log = log

	return nil
}

func (a *BaseService) Configure() bool {
	return false
}

func (a *BaseService) Exist() (bool, error) {
	return false, nil
}
func (a *BaseService) Create() error {
	return nil
}
func (a *BaseService) NeedUpdate() (bool, error) {
	return false, nil
}
func (a *BaseService) Update() error {
	return nil
}

func (a *BaseService) Delete() error {
	return nil
}

func (a *BaseService) generateAnnotations(o interface{}, old map[string]string) map[string]string {
	if old == nil {
		old = map[string]string{}
	}
	// set annotations
	specData, _ := json.Marshal(o)
	old[AnnotationName] = string(specData)

	return old
}

const AnnotationName = "app.service.spec"
