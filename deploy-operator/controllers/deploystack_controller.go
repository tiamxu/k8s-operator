package controllers

import (
	"context"
	"encoding/json"
	"reflect"

	"github.com/go-logr/logr"
	apiv1 "github.com/tiamxu/k8s-operator/deploy-operator/api/v1"
	"github.com/tiamxu/k8s-operator/deploy-operator/internal/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// var (
// 	// appList         map[string]string
// 	// resourceBuilder resource.DeployStackBuild
// 	resources client.Object
// )

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
	if err != nil {
		// 如果资源不存在，则忽略
		if errors.IsNotFound(err) {
			logger.Error(err, "Not Found DeployStack Resource ,Please Create Kind DeployStack resource")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		logger.Error(err, "Failed to get DeployStack resource")
		return ctrl.Result{}, err
	}
	logger.Info("Kind DeployStack Resource Normal...") //说明deploystack Kind已经创建
	logger.Info("Start reconciling")

	//序列化depoystack 配置
	instanceSpec, err := json.Marshal(deployStackInstance.Spec)
	if err != nil {
		logger.Error(err, "Failed to marshal cluster spec")
	}
	logger.V(1).Info("DeployStackInstance", "spec", string(instanceSpec))

	//声明并初始化一个DeployStackBuild的结构体变量
	// deploymentBuilder = resource.DeployStackBuild{Instance: deployStackInstance, Scheme: r.Scheme}
	resourceBuilder := resource.DeployStackBuild{Instance: deployStackInstance, Scheme: r.Scheme}

	appList := deployStackInstance.Spec.AppsList
	namespace := deployStackInstance.Spec.Namespace
	if appList == nil {
		// appList = map[string]string{"test": "latest"}
		return ctrl.Result{}, nil
	}
	for name, tag := range appList {
		builders := resourceBuilder.ResourceBuilds()
		fondResourceName := name
		var resources client.Object
		for _, builder := range builders {
			//获取对于资源类型
			if resources, err = builder.GetObjectKind(); err != nil {
				return ctrl.Result{}, err
			}

			if _, ok := resources.(*corev1.ConfigMap); ok {
				fondResourceName = "global-config"

			} else if _, ok := resources.(*corev1.Secret); ok {
				fondResourceName = "global-secret"

			} else if _, ok := resources.(*v1.Ingress); ok {
				for _, ingress := range resourceBuilder.Instance.Spec.Ingress {
					if ingress.Name == name {
						fondResourceName = resource.StringCombin(name, "-", "ingress")
						break
					}

				}
				if fondResourceName == "global-secret" {
					continue
				}
			} else {
				fondResourceName = name
			}
			//获取对于资源类型currentResourcesObj:= &appsv1.Deployment{}
			// resourceObj, ok := r.getObjectKind(resources)
			// if !ok {
			// 	continue
			// }
			// logger.Info("Begin Fetch getResourceObj")
			currentResourceObj, err := r.getResourceObj(ctx, namespace, fondResourceName, resources)
			if client.IgnoreNotFound(err) != nil {
				return ctrl.Result{}, err
			}
			// 如果 对于 资源对象不存在，则创建
			resourceObj := resources
			if errors.IsNotFound(err) {
				logger.Info("NotFound Resource for DeployStack, Create one", "Name", fondResourceName, "Kind", reflect.TypeOf(resourceObj))
				//Create Resource
				if resourceObj, err = builder.Build(name, tag); err != nil {
					return ctrl.Result{}, err
				}
				if err := r.Client.Create(ctx, resourceObj); err != nil {
					logger.Error(err, "Create Resource  Failed", "Name", fondResourceName, "Kind", reflect.TypeOf(resourceObj))
					return ctrl.Result{}, err
				}
				r.Recorder.Eventf(resourceObj, corev1.EventTypeNormal, "Created", "Created resource %T", resourceObj)
			} else {
				logger.Info("Kind  resource already", "Name", fondResourceName, "Kind", reflect.TypeOf(currentResourceObj))
				// 如果资源对象存在，且需要更新，则更新
				var newResourceObj client.Object
				if newResourceObj, err = builder.Update(currentResourceObj, name, tag); err != nil {
					return ctrl.Result{}, err
				}
				// if !reflect.DeepEqual(newResourceObj, currentResourceObj) {
				if err := r.Client.Update(ctx, newResourceObj); err != nil {
					logger.Error(err, "Update Resource  Failed", "Name", fondResourceName, "Kind", reflect.TypeOf(newResourceObj))
					return ctrl.Result{}, err
				}
				logger.Info("Kind Resource Updated", "Name", fondResourceName, "Kind", reflect.TypeOf(newResourceObj))
				r.Recorder.Eventf(newResourceObj, corev1.EventTypeNormal, "Update", "Update Resource %T", newResourceObj)
				// }

			}
		}
		logger.Info("#####end分割线####", "Name", name)
	}
	//删除多余服务,通过资源标签过滤
	if err := r.resourcesDelete(ctx, deployStackInstance); err != nil {
		logger.Error(err, "Failed to Delete DeployStack resource")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
func (r *DeployStackReconciler) resourcesDelete(ctx context.Context, deployStack *apiv1.DeployStack) error {
	resourceBuilder := resource.DeployStackBuild{Instance: deployStack, Scheme: r.Scheme}
	builders := resourceBuilder.ResourceBuilds()
	var (
		err       error
		resources client.Object
	)
	labelSelector := labels.SelectorFromSet(map[string]string{"app.kubernetes.io/name": "deploystack"})
	listOps := &client.ListOptions{Namespace: deployStack.Spec.Namespace, LabelSelector: labelSelector}
	for _, builder := range builders {
		if resources, err = builder.GetObjectKind(); err != nil {
			return err
		}
		switch resources.(type) {
		case *appsv1.Deployment:
			resourceObjList := &appsv1.DeploymentList{}
			if err := r.List(ctx, resourceObjList, listOps); err != nil {
				return err
			}
			for _, resourceObj := range resourceObjList.Items {
				if _, ok := deployStack.Spec.AppsList[resourceObj.Name]; !ok {
					//deployment no longer exists in the deploystack spec, so delete it
					if err := r.Delete(ctx, &resourceObj); err != nil {
						return err
					}
					r.Recorder.Eventf(&resourceObj, corev1.EventTypeNormal, "Deleted", "Deleted Resource %T", resourceObj)
				}
			}
		case *corev1.Service:
			resourceObjList := &corev1.ServiceList{}
			if err := r.List(ctx, resourceObjList, listOps); err != nil {
				return err
			}
			for _, resourceObj := range resourceObjList.Items {
				if _, ok := deployStack.Spec.AppsList[resourceObj.Name]; !ok {
					if err := r.Delete(ctx, &resourceObj); err != nil {
						return err
					}
					r.Recorder.Eventf(&resourceObj, corev1.EventTypeNormal, "Deleted", "Deleted Resource %T", resourceObj)
				}
			}
		case *corev1.Secret:
			resourceObjList := &corev1.SecretList{}
			if err := r.List(ctx, resourceObjList, listOps); err != nil {
				return err
			}
			for _, resourceObj := range resourceObjList.Items {
				if resourceObj.Name == "global-secret" {
					continue
				}
				if _, ok := deployStack.Spec.AppsList[resourceObj.Name]; !ok {
					if err := r.Delete(ctx, &resourceObj); err != nil {
						return err
					}
					r.Recorder.Eventf(&resourceObj, corev1.EventTypeNormal, "Deleted", "Deleted Resource %T", resourceObj)
				}
			}
		case *corev1.ConfigMap:
			resourceObjList := &corev1.ConfigMapList{}
			if err := r.List(ctx, resourceObjList, listOps); err != nil {
				return err
			}
			for _, resourceObj := range resourceObjList.Items {
				if resourceObj.Name == "global-config" {
					continue
				}
				if _, ok := deployStack.Spec.AppsList[resourceObj.Name]; !ok {
					if err := r.Delete(ctx, &resourceObj); err != nil {
						return err
					}
					r.Recorder.Eventf(&resourceObj, corev1.EventTypeNormal, "Deleted", "Deleted Resource %T", resourceObj)
				}
			}
		case *v1.Ingress:
			resourceObjList := &v1.IngressList{}
			if err := r.List(ctx, resourceObjList, listOps); err != nil {
				return err
			}
			for _, resourceObj := range resourceObjList.Items {
				if _, ok := deployStack.Spec.AppsList[resourceObj.Name]; !ok {
					if err := r.Delete(ctx, &resourceObj); err != nil {
						return err
					}
					r.Recorder.Eventf(&resourceObj, corev1.EventTypeNormal, "Deleted", "Deleted Resource %T", resourceObj)
				}
			}
		}

	}

	return nil
}

// func (r *DeployStackReconciler) getObjectResourceList(resources client.Object) (client.ObjectList, bool) {
// 	//判断所对应的资源类型属于那个Kind，之后进入对于的逻辑中处理
// 	switch resources.(type) {
// 	case *appsv1.Deployment:
// 		// logger.Info("Resource Kind Type", "Kind", reflect.TypeOf(resourceObj))
// 		resourceObjList := &appsv1.DeploymentList{}
// 		return resourceObjList, true
// 	case *corev1.Service:
// 		// logger.Info("Resource Kind Type", "Kind", reflect.TypeOf(resourceObj))
// 		resourceObjList := &corev1.ServiceList{}
// 		return resourceObjList, true
// 	case *corev1.Secret:
// 		resourceObjList := &corev1.SecretList{}
// 		return resourceObjList, true
// 	case *corev1.ConfigMap:
// 		resourceObjList := &corev1.ConfigMapList{}
// 		return resourceObjList, true
// 	case *v1.Ingress:
// 		resourceObjList := &v1.IngressList{}
// 		return resourceObjList, true
// 	default:
// 		// logger.Info("Other Kind Type")
// 		return nil, false
// 	}

// }

// 查询DeployStack Kind
func (r *DeployStackReconciler) getDeployStack(ctx context.Context, namespaceName types.NamespacedName) (*apiv1.DeployStack, error) {
	deployStackInstance := &apiv1.DeployStack{}
	if err := r.Get(ctx, namespaceName, deployStackInstance); err != nil {
		return deployStackInstance, err
	}
	return deployStackInstance, nil
}

// 查询资源，返回资源版本
// func (r *DeployStackReconciler) getResourceVersion(ctx context.Context, namespace, name string, obj client.Object) (string, error) {
// 	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, obj); err != nil {
// 		return "", err
// 	}
// 	resourceVersion := obj.GetResourceVersion()
// 	return resourceVersion, nil
// }

// 查询对应资源
func (r *DeployStackReconciler) getResourceObj(ctx context.Context, namespace, name string, resourceObj client.Object) (client.Object, error) {
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, resourceObj); err != nil {
		return resourceObj, err
	}
	return resourceObj, nil
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
