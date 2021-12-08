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
This introduces a new subdirectory into our repository ./repos. That directory will get checked for repository definition yaml.  And, let's give it a definition to look at.

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
