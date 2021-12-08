# DigitalOceanChallenge2021
Challenge objective:
**Deploy a scalable message queue**

"A critical component of all the scalable architectures are message queues used to store and distribute messages to multiple parties and introduce buffering. Kafka is widely used in this space and there are multiple operators like Strimzi or to deploy it. For this project, use a sample app to demonstrate how your message queue works."

To start this challenge, we need a kubernetes cluster.  I always start out designing on a kind cluster, so let's do that.
```shell
❯ kind create cluster --config=./cluster.yaml
Creating cluster "kind" ...
 ✓ Ensuring node image (kindest/node:v1.21.1) 🖼
 ✓ Preparing nodes 📦 📦 📦 📦  
 ✓ Writing configuration 📜 
 ✓ Starting control-plane 🕹️ 
 ✓ Installing CNI 🔌 
 ✓ Installing StorageClass 💾 
 ✓ Joining worker nodes 🚜 
Set kubectl context to "kind-kind"
You can now use your cluster with:

kubectl cluster-info --context kind-kind

Thanks for using kind! 😊
❯ k get nodes
NAME                 STATUS   ROLES                  AGE   VERSION
kind-control-plane   Ready    control-plane,master   16m   v1.21.1
kind-worker          Ready    <none>                 15m   v1.21.1
kind-worker2         Ready    <none>                 15m   v1.21.1
kind-worker3         Ready    <none>                 15m   v1.21.1


```
With a cluster up and running, having something that will coordinate all the deployments and charts we'll want for this would be helpful.  Let's use flux for that.

Flux, https://fluxcd.io/, has a whole bunch of functionality that you can explore.  We're just going to use it here to keep the resource management sane in this example. 

The requirements for flux are pretty straight forward.  You need:
* a github or gitlab repo of your own (it will use the repo to keep track of the config). I'm using the repo where this README lives as mine. You could fork this one if you like, or just point at an empty one you create. Either would work.
* a kubernetes cluster (like the one we just spun up), 
* and install/set up the cli. See the "Getting Started" doc, https://fluxcd.io/docs/get-started/, for how to do that. Make sure you define the environment variables that identify your github/gitlab user and token as explained in that doc. 

Just to be sure, verify that flux is installed and feels good about the cluster.
```shell
❯ flux --version
flux version 0.24.0
❯ flux check --pre
► checking prerequisites
✔ Kubernetes 1.21.1 >=1.19.0-0
✔ prerequisites checks passed
```
Now let's install and configure flux for our cluster.  The command to do that is "bootstrap".  The specific args to pass are:
* owner -- this is your github user
* repository -- this is the repository you want flux to use to keep track of things. If you've set the token privs right, it'll even create it for you if it doesn't already exist.
* branch -- This is the branch you want flux to use. I will use the default branch for the repo for this example.
* path -- Path within the repo where flux will place its management directory structure. 
* personal -- This tells flux that the repo owner is a GitHub user

Here's how I'm going to bootstrap my cluster:
```shell
❯ flux bootstrap github \
  --owner=$GITHUB_USER \
  --repository=DigitalOceanChallenge2021 \
  --branch=development \
  --path=./ops \
  --personal
► connecting to github.com
► cloning branch "development" from Git repository "https://github.com/mdbdba/DigitalOceanChallenge2021.git"
✔ cloned repository
► generating component manifests
✔ generated component manifests
✔ committed sync manifests to "development" ("e4e344ae7abefb2a7d6d0a032531ad8299bf3f8a")
► pushing component manifests to "https://github.com/mdbdba/DigitalOceanChallenge2021.git"
✔ installed components
✔ reconciled components
► determining if source secret "flux-system/flux-system" exists
► generating source secret
✔ public key: ecdsa-sha2-nistp384 AAAAE2Vj...
✔ configured deploy key "flux-system-development-flux-system-./ops" for "https://github.com/mdbdba/DigitalOceanChallenge2021"
► applying source secret "flux-system/flux-system"
✔ reconciled source secret
► generating sync manifests
✔ generated sync manifests
✔ committed sync manifests to "development" ("0343a43f460d1485d89bae8deef1d33eb50005e6")
► pushing sync manifests to "https://github.com/mdbdba/DigitalOceanChallenge2021.git"
► applying sync manifests
✔ reconciled sync configuration
◎ waiting for Kustomization "flux-system/flux-system" to be reconciled
✔ Kustomization reconciled successfully
► confirming components are healthy
✔ helm-controller: deployment ready
✔ kustomize-controller: deployment ready
✔ notification-controller: deployment ready
✔ source-controller: deployment ready
✔ all components are healthy

```
One of the CRDs that gets added with flux is kustomize.  We can verify that things are good so far by checking if any kustomizations exist.
```shell
❯ k get ks -n flux-system
NAME          READY   STATUS                                                                   AGE
flux-system   True    Applied revision: development/833451eac2d2ef566ea0c9a7f5906ab14b0a11fc   6m38s
```
When flux sets things up it puts them in the flux-system namespace. When we look at that namespace for kustomizations with that command we see that one got applied. Great, it's not much yet, but it's a start.

By the way, the flux command line can get information like this too.  Check it out:
```shell
❯ flux get kustomizations
NAME       	READY	MESSAGE                                                               	REVISION                                            	SUSPENDED 
flux-system	True 	Applied revision: development/833451eac2d2ef566ea0c9a7f5906ab14b0a11fc	development/833451eac2d2ef566ea0c9a7f5906ab14b0a11fc	False  
```

A bit more about what happened with the bootstrap command. Have a look at the repo you should see that the path used in the bootstrap command has been built out.
```shell
❯ tree ops
ops
└── flux-system
    ├── gotk-components.yaml
    ├── gotk-sync.yaml
    └── kustomization.yaml
```
These files will govern how we add/maintain the functionality we add to the cluster with a minimum of fuss. 

With flux all set up, let's give it something to do.  We set out on this challenge to get an example app that touches kafka.  Getting Kafka installed seems like a good place to start.

We're going to use a helm chart to install the Strimzi operator and that's what we will convince to set up kafka for us.

Flux is going to handle those pieces for us. For that to happen, we need to tell flux about where the helm chart (and any others we might need) lives.  Let's create a kustomization in the flux-system directory telling flux to look for helm repositories.

*flux-system/repo-sync.yaml*
```yaml
---
apiVersion: kustomize.toolkit.fluxcd.io/v1beta1
kind: Kustomization
metadata:
  name: repos
  namespace: flux-system
spec:
  interval: 10m0s
  path: ./repos
  prune: true
  sourceRef:
    kind: GitRepository
    name: flux-system
  validation: client
```
This introduces a new subdirectory into our repository ./repos. That directory will get checked for repository definition yaml.  Update *./ops/kustomization.yaml* to add the file to it's resources.
```yaml
...
resources:
- gotk-components.yaml
- gotk-sync.yaml
- repos-sync.yaml
```

And, let's give it a definition to look at.

*./repos/strimzi.yaml*
```yaml
---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: HelmRepository
metadata:
name: strimzi
namespace: flux-system
spec:
interval: 10m0s
url: https://strimzi.io/charts/
```
Once that is all checked into source control.  You'll see the new kustomization for "repos" get created by describing the kustomizations.
```yaml
❯ k get ks -n flux-system
  NAME          READY   STATUS                                                                   AGE
flux-system   True    Applied revision: development/e2531334fc795e3ede86817993838706bd100ece   3d20h
repos         True    Applied revision: development/e2531334fc795e3ede86817993838706bd100ece   2m5s

❯ k describe ks -n flux-system
Name:         flux-system
Namespace:    flux-system
Labels:       kustomize.toolkit.fluxcd.io/name=flux-system
              kustomize.toolkit.fluxcd.io/namespace=flux-system
...
Status:
  Conditions:
    Last Transition Time:  2021-12-08T00:50:18Z
    Message:               Applied revision: development/e2531334fc795e3ede86817993838706bd100ece
    Reason:                ReconciliationSucceeded
    Status:                True
    Type:                  Ready
  Inventory:
    Entries:
      Id:                   _alerts.notification.toolkit.fluxcd.io_apiextensions.k8s.io_CustomResourceDefinition
      V:                    v1
      ...
      Id:                   flux-system_flux-system_source.toolkit.fluxcd.io_GitRepository
      V:                    v1beta1
  Last Applied Revision:    development/e2531334fc795e3ede86817993838706bd100ece
  Last Attempted Revision:  development/e2531334fc795e3ede86817993838706bd100ece
  Observed Generation:      1
Events:
  Type    Reason  Age    From                  Message
  ----    ------  ----   ----                  -------
  Normal  info    8m30s  kustomize-controller  Reconciliation finished in 395.845917ms, next run in 10m0s
  Normal  info    4m49s  kustomize-controller  Reconciliation finished in 430.769842ms, next run in 10m0s
  Normal  info    84s    kustomize-controller  Kustomization/flux-system/repos created
  Normal  info    84s    kustomize-controller  Reconciliation finished in 400.801338ms, next run in 10m0s

Name:         repos
Namespace:    flux-system
Labels:       kustomize.toolkit.fluxcd.io/name=flux-system
              kustomize.toolkit.fluxcd.io/namespace=flux-system
...
Status:
  Conditions:
    Last Transition Time:  2021-12-08T00:50:18Z
    Message:               Applied revision: development/e2531334fc795e3ede86817993838706bd100ece
    Reason:                ReconciliationSucceeded
    Status:                True
    Type:                  Ready
  Inventory:
    Entries:
      Id:                   flux-system_strimzi_source.toolkit.fluxcd.io_HelmRepository
      V:                    v1beta1
  Last Applied Revision:    development/e2531334fc795e3ede86817993838706bd100ece
  Last Attempted Revision:  development/e2531334fc795e3ede86817993838706bd100ece
  Observed Generation:      1
Events:
  Type    Reason  Age   From                  Message
  ----    ------  ----  ----                  -------
  Normal  info    84s   kustomize-controller  HelmRepository/flux-system/strimzi created
  Normal  info    84s   kustomize-controller  Reconciliation finished in 37.122392ms, next run in 10m0s
  
  ❯ k get helmRepositories -A
  NAMESPACE     NAME      URL                          READY   STATUS                                                                               AGE
flux-system   strimzi   https://strimzi.io/charts/   True    Fetched revision: fe5f69ab3ee9d0810754153212089610d7f136a2a77c00f0784fde74c38e8736   2m28s
```
Now, it knows where to look. Let's give it something to look for.  I did the same thing we did with the repos directory for the releases directory.  Adding *./flux-system/releases-sync.yaml* and updating *./flux-system/kustomization.yaml*.

The *./releases/strimzi-kafka-operator* directory holds the files that define making the namespace, the helm chart install, and the kustomization that keeps an eye on that helm chart definition.

After checking all of that in, flux deploys all that for us.
```shell
❯ kgea
NAMESPACE     LAST SEEN   TYPE     REASON   OBJECT                                 MESSAGE
flux-system   68s       Normal   info     kustomization/releases                 Namespace/queuing configured
HelmRelease/flux-system/kafka-operator configured
Kustomization/flux-system/kafka-operator configured
flux-system   67s         Normal   info                gitrepository/flux-system                       Fetched revision: development/2d4e927fe90b0227a089624228e9a80998c8b132
flux-system   67s         Normal   info                kustomization/repos                             Reconciliation finished in 112.25794ms, next run in 10m0s
flux-system   67s         Normal   info                kustomization/kafka-operator                    Reconciliation finished in 186.166124ms, next run in 10m0s
flux-system   67s         Normal   info                kustomization/releases                          HelmRelease/flux-system/kafka-operator configured
flux-system   67s         Normal   info                kustomization/releases                          Reconciliation finished in 97.338797ms, next run in 10m0s
flux-system   66s         Normal   info                kustomization/flux-system                       Reconciliation finished in 568.4968ms, next run in 10m0s
flux-system   66s         Normal   info                helmrelease/kafka-operator                      Helm install has started
flux-system   66s         Normal   info                helmchart/flux-system-kafka-operator            Pulled 'strimzi-kafka-operator' chart with version '0.26.0'.
queuing       65s         Normal   SuccessfulCreate    replicaset/strimzi-cluster-operator-76f95f787   Created pod: strimzi-cluster-operator-76f95f787-hrdrn
queuing       65s         Normal   Scheduled           pod/strimzi-cluster-operator-76f95f787-hrdrn    Successfully assigned queuing/strimzi-cluster-operator-76f95f787-hrdrn to kind-worker3
queuing       65s         Normal   ScalingReplicaSet   deployment/strimzi-cluster-operator             Scaled up replica set strimzi-cluster-operator-76f95f787 to 1
queuing       64s         Normal   Pulling             pod/strimzi-cluster-operator-76f95f787-hrdrn    Pulling image "quay.io/strimzi/operator:0.26.0"
queuing       22s         Normal   Pulled              pod/strimzi-cluster-operator-76f95f787-hrdrn    Successfully pulled image "quay.io/strimzi/operator:0.26.0" in 41.915345909s
queuing       22s         Normal   Created             pod/strimzi-cluster-operator-76f95f787-hrdrn    Created container strimzi-cluster-operator
queuing       21s         Normal   Started             pod/strimzi-cluster-operator-76f95f787-hrdrn    Started container strimzi-cluster-operator
flux-system   2s          Normal   info                helmrelease/kafka-operator                      Helm install succeeded

❯ kga -n queuing
NAME                                           READY   STATUS    RESTARTS   AGE
pod/strimzi-cluster-operator-76f95f787-hrdrn   1/1     Running   0          19m

NAME                                       READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/strimzi-cluster-operator   1/1     1            1           19m

NAME                                                 DESIRED   CURRENT   READY   AGE
replicaset.apps/strimzi-cluster-operator-76f95f787   1         1         1       19m

```

