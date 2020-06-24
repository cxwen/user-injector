# user-injector

Inject user info in label or annotation for certain resources of Kubernetes by default。

为Kubernetes某类资源默认在label或annotation中注入用户名。

# 使用方法

1、设置注入类型

通过环境变量设置注入类型，以逗号分隔，默认值为label

- label表示往label中注入
- annotation表示往annotation中注入

``` shell
env:
- name: INJECTIONT_TYPE
  value: "label,annotation" # 设置注入的类型: label, annotation
```

2、设置注入资源类型规则

deployment/mutatingwebhook.yaml

``` yaml
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: user-injector-mutate
  labels:
    app: user-injector
webhooks:
  - name: user-injector.cxwen.com
    clientConfig:
      service:
        name: user-injector
        namespace: kube-system
        path: "/mutate"
      caBundle: ${CA_BUNDLE}
    # 以下为规则列表，下面两个为示例
    rules:
    - operations: [ "CREATE" ]
      apiGroups: [""]
      apiVersions: ["*"]
      resources: ["*"]
    - operations: [ "CREATE" ]
      apiGroups: ["apps"]
      apiVersions: ["*"]
      resources: ["*"]
```

3、部署

``` shell
sh deployment/webhook-create-signed-cert.sh
sh deployment/webhook-patch-ca-bundle.sh
kubectl apply -f deployment
```

注意更改 deployment/deployment.yaml中的镜像tag

``` shell
image: xwcheng/user-injector:latest
```

4、注入效果

添加用户label

例：

``` shell
username.cxwen.com: kubernetes-admin
```

