# DigitalOceanChallenge2021
## Part 1: Overview ##
Challenge objective: **Deploy a scalable message queue**

"A critical component of all the scalable architectures are message queues used to store and distribute messages to multiple parties and introduce buffering. Kafka is widely used in this space and there are multiple operators like Strimzi or to deploy it. For this project, use a sample app to demonstrate how your message queue works."

Everything is more fun when there's a story attached to it, so here's ours for this project:


Our company, Conglombo Corp Limited, has a research team working on text to voice simulation.  They are currently testing the cadence and dexterity of their voices by having them perform various Epic Rap Battles of History. Researchers are watching the performances and rating them accordingly.  The ratings are being placed in a kafka topic. Our goal is to take topics off of the queue and do some simple analysis of the data from the topic.

The demo will consist of a couple of web services. One will pretend to be the researchers, putting ratings onto a kafka topic. The other will pull ratings out of the topic and sum up the ratings for the performances of each of the voices.
For the first parts of this we will be developing locally using:
* a [kind cluster](https://kind.sigs.k8s.io/) for our kubernetes cluster
* [fluxcd](https://github.com/fluxcd/flux2) to make setting up and maintaining the services we put in k8s easier.
* [helm](https://helm.sh/) is a nice wrapper for describing deployments to k8s and easier for humans to read.
* the [strimzi kafka operator helm chart](https://strimzi.io/documentation/)

## Part 2: Kind Cluster and GitOps Setup ##
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

## Part 3: Add Kafka using GitOps ##
With flux all set up, let's give it something to do.  We set out on this challenge to get an example app that touches kafka.  Getting Kafka installed seems like a good place to start.

Getting Kafka set up at a high level will be a two step process.  The first step is that we're going to set up an operator to handle the complicated bits around setting up Kafka. Then, for step two, we are going to convince the operator to create a Kafka instance for us. For each of the steps, We're going to be using helm charts.

Making things even easier, Flux is going to handle those pieces for us. For that to happen, we need to tell flux about where the helm charts (and any others we might need) live.  Let's create a kustomization in the flux-system directory telling flux to look for helm repositories.

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
This introduces a new subdirectory into our repository ./repos. That directory will get checked for repository definition yaml.  Update *./ops/kustomization.yaml* to add the file to its resources.
```yaml
...
resources:
- gotk-components.yaml
- gotk-sync.yaml
- repos-sync.yaml
```

And, let's give it a definition to look at. The following describes in yaml where to look for the Strimzi helm charts.

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
Once that is all checked into source control.  You'll see the new kustomization for "repos" got created. Do that by describing the k8s kustomizations.
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
So, with the kafka operator all set up, we just need to tell it what we want it to do.  For this example I just slightly changed (as in, changed the cluster name and namespace) the single node ephemeral example used by the official release, https://github.com/strimzi/strimzi-kafka-operator/blob/main/examples/kafka/kafka-ephemeral-single.yaml . We wouldn't want to use something like this for a prod environment, but for developing our sample app this will do just fine.
In the ./releases/kafka-cluster directory I created a kustomization that takes the definition for that kafka cluster and applies it. The setup is about the same generally as the kafka-operator. Points to note here though, for kafka-cluster we're not using a helm deploy (see kafka.yaml) and we have this kustomization (ks.yaml) being dependent on the operator.  After adding the kafka-cluster/* files and getting the repo updated, flux reconciles things and we soon have a kafka instance to play with.
```shell
❯ kga -n queuing
NAME                                           READY   STATUS    RESTARTS   AGE
pod/kafka-entity-operator-5c489ddf46-dgw2f     3/3     Running   0          29m
pod/kafka-kafka-0                              1/1     Running   0          29m
pod/kafka-zookeeper-0                          1/1     Running   0          31m
pod/kafka-zookeeper-1                          1/1     Running   0          31m
pod/kafka-zookeeper-2                          1/1     Running   2          31m
pod/strimzi-cluster-operator-76f95f787-hrdrn   1/1     Running   0          5d1h

NAME                             TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)                               AGE
service/kafka-kafka-bootstrap    ClusterIP   10.96.157.72   <none>        9091/TCP,9092/TCP,9093/TCP            29m
service/kafka-kafka-brokers      ClusterIP   None           <none>        9090/TCP,9091/TCP,9092/TCP,9093/TCP   29m
service/kafka-zookeeper-client   ClusterIP   10.96.42.93    <none>        2181/TCP                              31m
service/kafka-zookeeper-nodes    ClusterIP   None           <none>        2181/TCP,2888/TCP,3888/TCP            31m

NAME                                       READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/kafka-entity-operator      1/1     1            1           29m
deployment.apps/strimzi-cluster-operator   1/1     1            1           5d1h

NAME                                                 DESIRED   CURRENT   READY   AGE
replicaset.apps/kafka-entity-operator-5c489ddf46     1         1         1       29m
replicaset.apps/strimzi-cluster-operator-76f95f787   1         1         1       5d1h

NAME                               READY   AGE
statefulset.apps/kafka-kafka       1/1     29m
statefulset.apps/kafka-zookeeper   3/3     31m

```
Yay!  Let's test it out quick for a smoke test.  We'll spin up a producer pod to put some messages out there, then create a consumer. With some luck there will be no surprises.
```shell
❯ kubectl -n queuing run kafka-producer -ti --image=quay.io/strimzi/kafka:0.26.0-kafka-3.0.0 --rm=true --restart=Never -- bin/kafka-console-producer.sh --broker-list kafka-kafka-bootstrap:9092 --topic manual-add-topic
If you don't see a command prompt, try pressing enter.
>
[2021-12-13 03:59:42,757] WARN [Producer clientId=console-producer] Error while fetching metadata with correlation id 3 : {manual-add-topic=LEADER_NOT_AVAILABLE} (org.apache.kafka.clients.NetworkClient)
[2021-12-13 03:59:42,858] WARN [Producer clientId=console-producer] Error while fetching metadata with correlation id 4 : {manual-add-topic=LEADER_NOT_AVAILABLE} (org.apache.kafka.clients.NetworkClient)
>test
>this is only a test
>repeat this is only a test
>^C pod "kafka-producer" deleted
pod queuing/kafka-producer terminated (Error)

❯ kubectl -n queuing run kafka-consumer -ti --image=quay.io/strimzi/kafka:0.26.0-kafka-3.0.0 --rm=true --restart=Never -- bin/kafka-console-consumer.sh --bootstrap-server kafka-kafka-bootstrap:9092 --topic manual-add-topic --from-beginning
If you don't see a command prompt, try pressing enter.


test
this is only a test
repeat this is only a test
^CProcessed a total of 4 messages
pod "kafka-consumer" deleted
pod queuing/kafka-consumer terminated (Error)

```
Nice. Okay, so at this point we have all the infrastructure we need to write our own producer and consumer services in place, and we've kicked the tires on it a bit. 

Our company, Conglombo Corp Limited, has a research team working on text to voice simulation.  They are currently testing the cadence and dexterity of their voices by having them perform various Epic Rap Battles of History. Researchers are watching the performances and rating them accordingly.  The ratings are being placed in a kafka topic. Our goal is to take topics off of the queue and do some simple analysis of the data from the topic. 

Now, have a look in the src subdirectory.  In src/producer is a web service that simulates the researchers making ratings.  Hit the endpoint (or refresh the page) and it generates a number of ratings. The ratings are placed in kafka and await the consumer.   

The consumer pulls a number of ratings off the topic, crunches them up and shows the total rating values for the simulated voices.   

The game to be played there is to refresh the producer service, and once it's done refreshing, do the same for the consumer service, and notice that the rating values increase. The producer will look like this:
![Producer](doc/img/producer.png)
And, the consumer will look something like this:
![Consumer](doc/img/consumer.png)
As you can see in my ratings data, Joan of Arc has the highest sum of ratings.  Hope your favorite voice is at the top of your list!

A couple of notes if you were to try running the code locally.  To interact with kafka running in a cluster you'll need to make a way to get at it.  Do that by forwarding a port from the kafka bootstrap to localhost like this
```shell
❯ k port-forward -n queuing service/kafka-kafka-bootstrap 9092:9092
```
That will allow your producer or consumer to get at the kafka service, but there's one more step.  When a client talks with kafka it reports back the pod information to connect to.  Which, when you're outside the cluster presents a problem.  I got over that by adding the address the client complained about to my /etc/hosts file.  Like this:
```shell
❯ cat /etc/hosts
127.0.0.1	localhost kafka-kafka-0.kafka-kafka-brokers.queuing.svc
```
