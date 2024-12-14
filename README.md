# kstatus-rbac-test

I setup this repo to figure out which RBAC permissions were needed to use kstatus. 

To replicate this test, run `./script.sh`, then exec into the pod and copy and paste the contents of `pod-script.sh`

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