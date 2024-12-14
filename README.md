# kstatus-rbac-test

I setup this repo to figure out which RBAC permissions were needed to use kstatus

Conclusion:
this is the minimal set of permissions needed to run a kstatus check on a deployment
```yaml
rules:
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["list", "watch"]
  - apiGroups: ["apps"]
    resources: ["replicasets"]
    verbs: ["list"]
```