package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	apiv1 "github.com/tiamxu/k8s-operator/deploy-operator/api/v1"
	"github.com/tiamxu/k8s-operator/deploy-operator/internal/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	appList         map[string]string
	namespace       string
	resourceBuilder resource.DeployStackBuild
	resources       client.Object
)

type DeployStackReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=gopron.online,resources=deploystacks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gopron.online,resources=deploystacks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gopron.online,resources=deploystacks/finalizers,verbs=update

func (r *DeployStackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// ctx = context.Background()
	logger := r.Log.WithValues("DeployStack", req.NamespacedName)

	deployStackInstance, err := r.getDeployStack(ctx, req.NamespacedName)
	// fmt.Println("########DeployStack配置:", deployStackInstance.Spec, "###########")
	if err != nil {
		logger.Error(err, "Not Found DeployStack Resource ,Please Create Kind DeployStack resource")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("########Kind DeployStack Resource Normal...######") //说明deploystack Kind已经创建
	logger.Info("######Start reconciling#########")

	//序列化depoystack 配置
	instanceSpec, err := json.Marshal(deployStackInstance.Spec)
	if err != nil {
		logger.Error(err, "Failed to marshal cluster spec")
	}
	logger.V(1).Info("DeployStackInstance", "spec", string(instanceSpec))

	//声明并初始化一个DeployStackBuild的结构体变量
	// deploymentBuilder = resource.DeployStackBuild{Instance: deployStackInstance, Scheme: r.Scheme}
	resourceBuilder = resource.DeployStackBuild{Instance: deployStackInstance, Scheme: r.Scheme}

	appList = deployStackInstance.Spec.AppsList
	namespace = deployStackInstance.Spec.Namespace
	if appList == nil {
		appList = map[string]string{"test": "latest"}
	}
	for name, tag := range appList {
		//builders 相当于[]ResourceBuilder 接口类型
		builders := resourceBuilder.ResourceBuilds()
		for _, builder := range builders {
			//resources为对应资源Kind的Object
			if resources, err = builder.Build(name, tag); err != nil {
				return ctrl.Result{}, err
			}

			//判断所对应的资源类型属于那个Kind，之后进入对于的逻辑中处理
			switch resourceObj := resources.(type) {
			case *appsv1.Deployment:
				logger.Info("Fetch Kind Deployment", "Name", name, "Kind", reflect.TypeOf(resourceObj))
				//查询服务是否存在Get
				if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, resourceObj); err != nil {
					//服务不存在就创建Create
					//当error等于nil 时返回false
					if errors.IsNotFound(err) {
						logger.Info("NotFound resource Deployment for DeployStack, Create one", "Name", name)
						if err := r.Client.Create(ctx, resourceObj); err != nil {
							logger.Error(err, "Create Resource Deployment Failed")
							return ctrl.Result{}, err
						}
						r.Recorder.Eventf(resourceObj, corev1.EventTypeNormal, "Created", "Created deployment %q", name)
					}
					// if err := r.Client.Status().Update(ctx, resourceObj); err != nil {
					// 	logger.Error(err, "Failed to update status")
					// 	return ctrl.Result{}, err
					// }

				} else {
					logger.Info("Kind Deployment resource already", "Name", name, "Kind", reflect.TypeOf(resourceObj))
					oldResourceVersion, err := r.getResourceObj(ctx, namespace, name, resourceObj)
					if err != nil {
						return ctrl.Result{}, err
					}
					//Modify Resource
					if err := builder.Update(resourceObj, name, tag); err != nil {
						return ctrl.Result{}, err
					}
					//Update
					if err := r.Client.Update(ctx, resourceObj); err != nil {
						logger.Error(err, "Update Resource Deployment Failed")
						return ctrl.Result{}, err
					}
					newResourceVersion, err := r.getResourceObj(ctx, namespace, name, resourceObj)
					if err != nil {
						return ctrl.Result{}, err
					}
					fmt.Printf("deployment:%s,oldResourceVersion:%s,newResourceVersion:%s\n", name, oldResourceVersion, newResourceVersion)
					// 比较新旧资源对象的 resourceVersion 字段的值
					if oldResourceVersion != newResourceVersion {
						logger.Info("Kind Service Updated", "Name", name, "Kind", reflect.TypeOf(resourceObj))
						r.Recorder.Eventf(resourceObj, corev1.EventTypeNormal, "Update", "Update Service %q", name)
					}
				}
				logger.Info("exec end deployment")

			case *corev1.Service:
				logger.Info("Fetch Kind Service", "Name", name, "Kind", reflect.TypeOf(resourceObj))
				if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, resourceObj); err != nil {
					if errors.IsNotFound(err) {
						logger.Info("NotFound resource Service for DeployStack, Create one", "Name", name)
						if err := r.Client.Create(ctx, resourceObj); err != nil {
							logger.Error(err, "Create Resource Service Failed")
							return ctrl.Result{}, err
						}
						r.Recorder.Eventf(resourceObj, corev1.EventTypeNormal, "Created", "Created Service %q", name)
						// if err := r.Client.Status().Update(ctx, resourceObj); err != nil {
						// 	logger.Error(err, "Failed to update status")
						// 	return ctrl.Result{}, err
						// }
					}

				} else {
					logger.Info("Kind Service resource already", "Name", name, "Kind", reflect.TypeOf(resourceObj))
					oldResourceVersion, err := r.getResourceObj(ctx, namespace, name, resourceObj)
					if err != nil {
						return ctrl.Result{}, err
					}
					//Update
					if err := builder.Update(resourceObj, name, tag); err != nil {
						return ctrl.Result{}, err
					}
					if err := r.Client.Update(ctx, resourceObj); err != nil {
						logger.Error(err, "Update Resource Service Failed")
						return ctrl.Result{}, err
					}
					newResourceVersion, err := r.getResourceObj(ctx, namespace, name, resourceObj)
					if err != nil {
						return ctrl.Result{}, err
					}
					fmt.Printf("Service:%s,oldResourceVersion:%s,newResourceVersion:%s\n", name, oldResourceVersion, newResourceVersion)
					// 比较新旧资源对象的 resourceVersion 字段的值
					if oldResourceVersion != newResourceVersion {
						logger.Info("Kind Service Updated", "Name", name, "Kind", reflect.TypeOf(resourceObj))
						r.Recorder.Eventf(resourceObj, corev1.EventTypeNormal, "Update", "Update Service %q", name)
					}

				}
				logger.Info("exec end service")
			case *corev1.Secret:
				logger.Info("Fetch Kind Secret", "Name", name, "Kind", reflect.TypeOf(resourceObj))
				if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: "global-secret"}, resourceObj); err != nil {
					if errors.IsNotFound(err) {
						logger.Info("NotFound resource Secret for DeployStack, Create one")
						if err := r.Client.Create(ctx, resourceObj); err != nil {
							logger.Error(err, "Create Resource Secret Failed")
							return ctrl.Result{}, err
						}
						r.Recorder.Eventf(resourceObj, corev1.EventTypeNormal, "Created", "Created Secret ")

					}
				} else {
					logger.Info("Kind Secret resource already", "Name", name, "Kind", reflect.TypeOf(resourceObj))

				}

			case *corev1.ConfigMap:
			case *v1.Ingress:
				logger.Info("Fetch Kind Ingress", "Name", name, "Kind", reflect.TypeOf(resourceObj))
				for _, ingress := range resourceBuilder.Instance.Spec.Ingress {
					if name == ingress.Name {
						if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: resource.StringCombin(name, "-", "ingress")}, resourceObj); err != nil {
							if errors.IsNotFound(err) {
								logger.Info("NotFound resource Ingress for DeployStack, Create one")
								if err := r.Client.Create(ctx, resourceObj); err != nil {
									logger.Error(err, "Create Resource ingress Failed")
									return ctrl.Result{}, err
								}
								r.Recorder.Eventf(resourceObj, corev1.EventTypeNormal, "Created", "Created ingress ")

							}
						} else {
							logger.Info("Kind Ingress resource already", "Name", name, "Kind", reflect.TypeOf(resourceObj))
						}
					}
				}

			default:
				logger.Info("Other Kind Type")
			}

		}
		logger.Info("#####end分割线####", "Name", name)
	}

	return ctrl.Result{}, nil
}

// 查询DeployStack Kind
func (r *DeployStackReconciler) getDeployStack(ctx context.Context, namespaceName types.NamespacedName) (*apiv1.DeployStack, error) {
	deployStackInstance := &apiv1.DeployStack{}
	if err := r.Get(ctx, namespaceName, deployStackInstance); err != nil {
		return deployStackInstance, err
	}
	return deployStackInstance, nil
}

// 查询资源，返回资源版本
func (r *DeployStackReconciler) getResourceObj(ctx context.Context, namespace, name string, obj client.Object) (string, error) {
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, obj); err != nil {
		return "", err
	}
	resourceVersion := obj.GetResourceVersion()
	return resourceVersion, nil
}

// 查询Deployment
func (r *DeployStackReconciler) Deployment(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, deployment); err != nil {
		return nil, err
	}
	return deployment, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeployStackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.DeployStack{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&v1.Ingress{}).
		Complete(r)
}
