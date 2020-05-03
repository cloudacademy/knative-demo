# AKS VoteApp Deployment Instructions

The following instructions are used to demonstrate Knative Serving and Eventing configurations.

# STEP 1:

Install Knative

## STEP 1.1:

Install Istio

```
ISTIO_VERSION=1.4.7

kubectl apply -f https://raw.githubusercontent.com/knative/serving/master/third_party/istio-$ISTIO_VERSION/istio-crds.yaml

kubectl apply -f https://raw.githubusercontent.com/knative/serving/master/third_party/istio-$ISTIO_VERSION/istio-minimal.yaml
```

## STEP 1.2:

Check pods have STATUS 'Running'

```
kubectl get pods -n istio-system
```

## STEP 1.3:

Install Serving

* Serving CRDs
* Serving Core Components
* Knative Istio Controller needed for Serving

```
KNATIVE_VERSION=0.13.0

kubectl apply -f https://github.com/knative/serving/releases/download/v$KNATIVE_VERSION/serving-crds.yaml

kubectl apply -f https://github.com/knative/serving/releases/download/v$KNATIVE_VERSION/serving-core.yaml

kubectl apply -f https://github.com/knative/serving/releases/download/v$KNATIVE_VERSION/serving-istio.yaml
```

## STEP 1.4:

Check Serving pods are running

```
kubectl get pods -n knative-serving
```

## STEP 1.5:

Setup xip.io for custom domain - provides wildcarded dynamic dns

```
kubectl apply -f https://github.com/knative/serving/releases/download/v$KNATIVE_VERSION/serving-default-domain.yaml
```

## STEP 1.6:

Install Eventing

* Eventing CRDs
* Eventing Core
* Default InMemory Channel (not suitable for production)
* Default Broker

```
KNATIVE_VERSION=0.13.0

kubectl apply -f https://github.com/knative/eventing/releases/download/v$KNATIVE_VERSION/eventing-crds.yaml

kubectl apply -f https://github.com/knative/eventing/releases/download/v$KNATIVE_VERSION/eventing-core.yaml

kubectl apply -f https://github.com/knative/eventing/releases/download/v$KNATIVE_VERSION/in-memory-channel.yaml

kubectl apply -f https://github.com/knative/eventing/releases/download/v$KNATIVE_VERSION/channel-broker.yaml
```

## STEP 1.7:

Check Eventing pods are running

```
kubectl get pods -n knative-eventing
```

# STEP 2:

Create the ```cloudacademy``` namespace

```
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Namespace
metadata:
  name: cloudacademy
  labels:
    name: cloudacademy
EOF
```

Configure the ```cloudacademy``` namespace to be the default

```
kubectl config set-context --current --namespace cloudacademy
```

# Step 3

Example Knative Service

# Step 3.1

Install example helloworld service

```
for version in {1..2}; do
cat << EOF | kubectl apply -f -
apiVersion: serving.knative.dev/v1alpha1
kind: Service
metadata:
  name: hellosvc
  namespace: cloudacademy
spec:
  template:
    metadata:
      name: hellosvc-v$version
    spec:
      containers:
        - image: docker.io/cloudacademydevops/helloworld:v1
          env:
            - name: SENDER
              value: "cloudacademy.knative.v$version"
EOF
done
```

Test the ```hellosvc``` knative service

```
kubectl get ksvc

HELLO_SVC_URL=$(kubectl get ksvc/hellosvc -o jsonpath="{.status.url}")
echo $HELLO_SVC_URL

curl $HELLO_SVC_URL/hello
```

Display current revisions

```
kubectl get revision
```

# Step 3.2

Install example helloworld service with traffic splitting

```
cat << EOF | kubectl apply -f -
apiVersion: serving.knative.dev/v1alpha1
kind: Service
metadata:
  name: hellosvc
  namespace: cloudacademy
spec:
  template:
    metadata:
      name: hellosvc-v3
    spec:
      containers:
        - image: docker.io/cloudacademydevops/helloworld:v1
          env:
            - name: SENDER
              value: "cloudacademy.knative.v3"
  traffic:
  - tag: prod
    revisionName: hellosvc-v3
    percent: 50
  - tag: staging
    revisionName: hellosvc-v2
    percent: 50
  - tag: latest
    latestRevision: true
    percent: 0
EOF
```

Test the ```hellosvc``` knative service traffic splitting

```
kubectl get ksvc

HELLO_SVC_URL=$(kubectl get ksvc/hellosvc -o jsonpath="{.status.url}")
echo $HELLO_SVC_URL

for i in {1..10}; do
curl $HELLO_SVC_URL/hello
done
```

Test tag based urls

```
HELLO_SVC_URL_PROD=${HELLO_SVC_URL/hellosvc/prod-hellosvc}
HELLO_SVC_URL_STAGING=${HELLO_SVC_URL/hellosvc/staging-hellosvc}

echo $HELLO_SVC_URL_PROD
echo $HELLO_SVC_URL_STAGING

curl $HELLO_SVC_URL_PROD/hello
curl $HELLO_SVC_URL_STAGING/hello
```

```
kubectl get revision
```

# Step 4

Configure knative autoscaling using kpa and 2 requests in-flight per pod

```
cat << EOF | kubectl apply -f -
apiVersion: serving.knative.dev/v1alpha1
kind: Service
metadata:
  name: hellosvc
  namespace: cloudacademy
spec:
  template:
    metadata:
      name: hellosvc-v4
      annotations:
        # 2 requests in-flight per pod - for testing
        autoscaling.knative.dev/class:  kpa.autoscaling.knative.dev
        autoscaling.knative.dev/metric: concurrency
        autoscaling.knative.dev/target: "2"
        autoscaling.knative.dev/minScale: "0"
        autoscaling.knative.dev/maxScale: "20"
    spec:
      containers:
        - image: docker.io/cloudacademydevops/helloworld:v4
          imagePullPolicy: Always
          env:
            - name: SENDER
              value: "cloudacademy.knative.v4"
EOF
```

Send 10 concurrent curl requests to service

```
for i in {1..10}; do echo $HELLO_SVC_URL/hello?id=$i; done | xargs -P 10 -n 1 curl
```

Examine pods

```
kubectl get pods --watch
```

# Step 5

Install Eventing - Source to Sink

## Step 5.1

Install PingSource

```
cat << EOF | kubectl apply -f -
apiVersion: sources.knative.dev/v1alpha2
kind: PingSource
metadata:
  name: ping-cloudacademy
  namespace: cloudacademy
spec:
  schedule: "* * * * *"
  jsonData: '{"message": "knative rocks!!"}'
  sink:
    ref:
      apiVersion: serving.knative.dev/v1
      kind: Service
      name: cloudacademy-service
EOF
```

## Step 5.2

Install Service - SimpleLogger

```
cat << EOF | kubectl apply -f -
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: cloudacademy-service
  namespace: cloudacademy
spec:
  template:
    metadata:
      name: cloudacademy-service-v1
    spec:
      containers:
      - image: cloudacademydevops/simplelogger:v1
        ports:
        - containerPort: 8080
EOF
```

## Step 5.3

Examine SimpleLogger pod log

```
SIMPLELOGGER_POD=$(kubectl get pod -l app=cloudacademy-service-v1 --no-headers=true -o custom-columns=:metadata.name)
echo $SIMPLELOGGER_POD

kubectl logs $SIMPLELOGGER_POD -c user-container --follow
```

# Step 6

Install Eventing - Channel and Subscription

## Step 6.1

Install InMemoryChannel

```
cat << EOF | kubectl apply -f -
apiVersion: messaging.knative.dev/v1alpha1
kind: InMemoryChannel
metadata:
  name: cloudacademy-channel
  namespace: cloudacademy
EOF
```

## Step 6.2

Install PingSource

```
cat << EOF | kubectl apply -f -
apiVersion: sources.knative.dev/v1alpha2
kind: PingSource
metadata:
  name: ping-cloudacademy
  namespace: cloudacademy
spec:
  schedule: "* * * * *"
  jsonData: '{"message": "knative rocks!!", "from": "pingsource - channelsub"}'
  sink:
    ref:
      apiVersion: messaging.knative.dev/v1alpha1
      kind: InMemoryChannel
      name: cloudacademy-channel
EOF
```

## Step 6.3

Install 2x Service - SimpleLogger

```
cat << EOF | kubectl apply -f -
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: cloudacademy-service1
  namespace: cloudacademy
spec:
  template:
    metadata:
      name: cloudacademy-service1-v1
      annotations:
        autoscaling.knative.dev/minScale: "1"
        autoscaling.knative.dev/maxScale: "1"
    spec:
      containers:
      - image: cloudacademydevops/simplelogger:v1
        ports:
        - containerPort: 8080
---
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: cloudacademy-service2
  namespace: cloudacademy
spec:
  template:
    metadata:
      name: cloudacademy-service2-v1
      annotations:
        autoscaling.knative.dev/minScale: "1"
        autoscaling.knative.dev/maxScale: "1"
    spec:
      containers:
      - image: cloudacademydevops/simplelogger:v1
        ports:
        - containerPort: 8080
EOF
```

## Step 6.4

Install 2x Subscription

```
cat << EOF | kubectl apply -f -
apiVersion: messaging.knative.dev/v1alpha1
kind: Subscription
metadata:
  name: cloudacademy-sub1
  namespace: cloudacademy
spec:
  channel:
    apiVersion: messaging.knative.dev/v1alpha1
    kind: InMemoryChannel
    name: cloudacademy-channel
  subscriber:
    ref:
      apiVersion: serving.knative.dev/v1
      kind: Service
      name: cloudacademy-service1
---
apiVersion: messaging.knative.dev/v1alpha1
kind: Subscription
metadata:
  name: cloudacademy-sub2
  namespace: cloudacademy
spec:
  channel:
    apiVersion: messaging.knative.dev/v1alpha1
    kind: InMemoryChannel
    name: cloudacademy-channel
  subscriber:
    ref:
      apiVersion: serving.knative.dev/v1
      kind: Service
      name: cloudacademy-service2
EOF
```

## Step 6.5

Examine SimpleLogger pods - log

```
SIMPLELOGGER_SVC1_POD=$(kubectl get pod -l app=cloudacademy-service1-v1 --no-headers=true -o custom-columns=:metadata.name)

SIMPLELOGGER_SVC2_POD=$(kubectl get pod -l app=cloudacademy-service2-v1 --no-headers=true -o custom-columns=:metadata.name)

echo $SIMPLELOGGER_SVC1_POD
echo $SIMPLELOGGER_SVC2_POD

kubectl logs $SIMPLELOGGER_SVC1_POD -c user-container --follow
kubectl logs $SIMPLELOGGER_SVC2_POD -c user-container --follow
```

# Step 7

Install Eventing - Broker and Trigger

## Step 7.1

Configure automatic knative eventing injection

```
kubectl label ns cloudacademy knative-eventing-injection=enabled
```

Get the broker url for the cloudacademy namespace

```
kubectl config set-context --current --namespace cloudacademy
kubectl get broker
```

## Step 7.2

Install PingSource

```
cat << EOF | kubectl apply -f -
apiVersion: sources.knative.dev/v1alpha2
kind: PingSource
metadata:
  name: ping-cloudacademy
  namespace: cloudacademy
spec:
  schedule: "* * * * *"
  jsonData: '{"message": "knative rocks!!", "from": "pingsource - brokertrigger"}'
  sink:
    ref:
      apiVersion: eventing.knative.dev/v1alpha1
      kind: Broker
      name: default
EOF
```

## Step 7.3

Install 3x Service - SimpleLogger

```
cat << EOF | kubectl apply -f -
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: cloudacademy-service1
  namespace: cloudacademy
spec:
  template:
    metadata:
      name: cloudacademy-service1-v1
      annotations:
        autoscaling.knative.dev/minScale: "1"
        autoscaling.knative.dev/maxScale: "1"
    spec:
      containers:
      - image: cloudacademydevops/simplelogger:v1
        ports:
        - containerPort: 8080
---
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: cloudacademy-service2
  namespace: cloudacademy
spec:
  template:
    metadata:
      name: cloudacademy-service2-v1
      annotations:
        autoscaling.knative.dev/minScale: "1"
        autoscaling.knative.dev/maxScale: "1"
    spec:
      containers:
      - image: cloudacademydevops/simplelogger:v1
        ports:
        - containerPort: 8080
---
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: cloudacademy-service3
  namespace: cloudacademy
spec:
  template:
    metadata:
      name: cloudacademy-service3-v1
      annotations:
        autoscaling.knative.dev/minScale: "1"
        autoscaling.knative.dev/maxScale: "1"
    spec:
      containers:
      - image: cloudacademydevops/simplelogger:v1
        ports:
        - containerPort: 8080
EOF
```

## Step 7.4

Install 3x Trigger

```
cat << EOF | kubectl apply -f -
apiVersion: eventing.knative.dev/v1alpha1
kind: Trigger
metadata:
  name: cloudacademy-trigger1
  namespace: cloudacademy
spec:
  filter:
    attributes:
      type: dev.knative.sources.ping
  subscriber:
    ref:
      apiVersion: serving.knative.dev/v1
      kind: Service
      name: cloudacademy-service1
---
apiVersion: eventing.knative.dev/v1alpha1
kind: Trigger
metadata:
  name: cloudacademy-trigger2
  namespace: cloudacademy
spec:
  filter:
    attributes:
      type: dev.knative.sources.ping
  subscriber:
    ref:
      apiVersion: serving.knative.dev/v1
      kind: Service
      name: cloudacademy-service2
---
apiVersion: eventing.knative.dev/v1alpha1
kind: Trigger
metadata:
  name: cloudacademy-trigger3
  namespace: cloudacademy
spec:
  filter:
    attributes:
      type: cloudacademy.app.blah
  subscriber:
    ref:
      apiVersion: serving.knative.dev/v1
      kind: Service
      name: cloudacademy-service3
EOF
```

## Step 7.5

Create Curler Pod

```
cat << EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  labels:
    run: curler
  name: curler
  namespace: cloudacademy
spec:
  containers:
  - name: curler
    image: fedora:latest
    tty: true
EOF
```

## Step 7.6

Retrieve broker url

```
BROKER_URL=$(kubectl get broker -o jsonpath='{.items[0].status.address.url}')
echo BROKER_URL: $BROKER_URL
```

## Step 7.7

Perform an HTTP POST - send CloudEvent message 

```
kubectl exec -it curler -- curl -v $BROKER_URL \
-X POST \
-H "Ce-Id: say-hello" \
-H "Ce-Specversion: 1.0" \
-H "Ce-Type: cloudacademy.app.blah" \
-H "Ce-Source: mycurl" \
-H "Content-Type: application/json" \
-d '{"key":"curl cloudevent message!!"}'
```

## Step 7.8

Examine SimpleLogger pods - log

```
SIMPLELOGGER_SVC1_POD=$(kubectl get pod -l app=cloudacademy-service1-v1 --no-headers=true -o custom-columns=:metadata.name)

SIMPLELOGGER_SVC2_POD=$(kubectl get pod -l app=cloudacademy-service2-v1 --no-headers=true -o custom-columns=:metadata.name)

SIMPLELOGGER_SVC3_POD=$(kubectl get pod -l app=cloudacademy-service3-v1 --no-headers=true -o custom-columns=:metadata.name)

echo $SIMPLELOGGER_SVC1_POD
echo $SIMPLELOGGER_SVC2_POD
echo $SIMPLELOGGER_SVC3_POD

kubectl logs $SIMPLELOGGER_SVC1_POD -c user-container --follow
kubectl logs $SIMPLELOGGER_SVC2_POD -c user-container --follow
kubectl logs $SIMPLELOGGER_SVC3_POD -c user-container --follow
```
