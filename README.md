[toc]
## how to run
1. run jaeger docker all in on in localhost
```bash
docker run -d --name jaeger \
  -e COLLECTOR_ZIPKIN_HTTP_PORT=9411 \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 9411:9411 \
  jaegertracing/all-in-one:1.18
```

2. run server b

```bash
go run main.go b
```
start another bash to run server c

```bash
go run main.go c
```

3. run client a

```bash
go run main.go a 10
```

4. open jaeger url to watch result `localhost:16686`

## how to use

### in the same process

1. init tracer from config
    ```go
    import (
     "testOpentracing/pkg/tracing"
    )
    serviceName := "service_a"
    agentEndpoint := "localhost:5775"
    debug := false
    closer, err := tracing.InitTracer(serviceName, agentEndpoint, debug)
    if err != nil {
        // handle error and log
    }
    // invoke closer.Close when
    defer closer.Close()
    ```

2. create a span
    ```go
    operationName := "function name or other op name"
    span := tracing.CreateSpan(operationName)
    // finish when exit the scope
    defer span.Finish()
    ```

3. create a child span when invoke a function
    ```go
    func DoHandleStep1(span opentracing.Span)  {
        childSpan := tracing.CreateChildFromSC(span.Context())
        defer childSpan.Finish()
    }
    ```
4. create a follower span when invoke a function
    ```go
    func DoHandleStep2(span opentracing.Span)  {
        followerSpan := tracing.CreateFollowerFromSC(span.Context())
        defer followerSpan.Finish()
    }
    ```

### tracing in IPC

1. init tracer in each process

2. create a span in first process and propagate carrier to the next process
    ```go
    operationName := "func 1"
    span := tracing.CreateSpan(operationName)
    defer span.Finish()
    carrier := GetCarrier(span)
    // propagate carrier to the next process
    ```

3. create a child span in the next process
    ```go
    operationName := "func 2"
    childSpan := tracing.CreateChildFromCarrier(operationName, carrier)
    defer childSpan.Finish()
    ```

4. or create a follower span in the next process
    ```go
    operationName := "func 3"
    followerSpan := tracing.CreateFollowerFromCarrier(operationName, carrier)
    defer followerSpan.Finish()
    ```

### log something or set some tags

1. log something
    ```go
    span.LogKV("output", result)
    ```
2. set tags
    ```go
    span.SetTag("server", "c")
    ```

## how to deploy in distributed system

There were 4 machines in the disturbed environment, called A, B, C, D.

We deployed services and jaeger agent in A, B, C machine, and deployed jaeger collector and queries in D machine for tracing debug.

![arch](https://live.staticflickr.com/65535/50244154736_01ac06c9ab_o.png)

### deploy without k8s version

These version use es storage for storing log and data. (Distributed jaeger version supports ES or Cassandra for storage.)

1. start an es instance in a machine or start an es cluster for more stable storage if need

```bash
docker run -d -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" docker.elastic.co/elasticsearch/elasticsearch:7.9.0
```

2. start agent in each machine

```bash
docker run -d -p 5775:5775/udp jaegertracing/jaeger-agent:latest --reporter.grpc.host-port=${COLLOCTOR-HOST-IP}:14250
```

3. start collector in a machine

```bash
docker run -d -e SPAN_STORAGE_TYPE=elasticsearch -e ES_SERVER_URLS=http://${ES-HOST-IP}:9200 -p 14250:14250/tcp jaegertracing/jaeger-collector:latest --es.index-prefix=openstracing
```

4. start query in a machine for showing tracing result

```bash
docker run -d -p 16686:16686 -e SPAN_STORAGE_TYPE=elasticsearch -e ES_SERVER_URLS=http://${ES-HOST-IP}:9200 jaegertracing/jaeger-query:latest --es.index-prefix=openstracing
```

### deploy with k8s

1. deploy jaeger operator in k8s environment.

```bash
kubectl create namespace observability # <1>
kubectl create -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/crds/jaegertracing.io_jaegers_crd.yaml # <2>
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/service_account.yaml
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/role.yaml
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/role_binding.yaml
kubectl create -n observability -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/operator.yaml

kubectl create -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/cluster_role.yaml
kubectl create -f https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/cluster_role_binding.yaml
```

2. prepare external storage such as stable ES cluster.
Use Es operator or deploy by docker image.

3. deploy production component (collector / ingester / query)

```yaml
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: simple-prod
spec:
  strategy: production
  collector:
    image: jaegertracing/jaeger-collector:latest # <1>
  query:
    image: jaegertracing/jaeger-query:latest # <1>
  agent: # delete these two line if you want to use sidecar mode
    strategy: DaemonSet
  storage:
    type: elasticsearch
    options:
      es:
        server-urls: http://${ES-HOST}:9200
```

4. [Optional] prepare MQ such as Kafka in k8s if need (need ingester) *[not tried]*
This is suitable for streaming mode.

5. set agent with application as sidecar or DaemonSet
    1. add agent as sidecar container and use it

    ```yaml
     - name: jaeger-agent
       image: jaegertracing/jaeger-agent:latest # it's best to keep this version in sync with the operator's
       env:
       - name: POD_NAMESPACE
         valueFrom:
           fieldRef:
             fieldPath: metadata.namespace
       args:
       - --reporter.grpc.host-port=dns:///jaeger-collector-headless.$(POD_NAMESPACE).svc.cluster.local:14250
       ports:
       - containerPort: 5775
         name: jg-compact-trft
         protocol: UDP
   ```

   2. (1) deploy agent as DaemonSet

   ```yaml
   apiVersion: jaegertracing.io/v1
   kind: Jaeger
   metadata:
     name: my-jaeger
   spec:
     agent: # add these two line to collector & query yaml
       strategy: DaemonSet
   ```
   
   2. (2) use agent in application k8s yaml

   ```yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: myapp
   spec:
     selector:
       matchLabels:
         app: myapp
     template:
       metadata:
         labels:
           app: myapp
       spec:
         containers:
         - name: myapp
           image: acme/myapp:myversion
           env:
           - name: JAEGER_AGENT_HOST  # use env in the application config code or yaml config file
             valueFrom:
               fieldRef:
                 fieldPath: status.hostIP
   ```

PS: 其实 agent 和 jaeger client 可以不在同一个 host 上，但因为协议使用 udp ， 有存在丢失的可能性。 所有最佳实践上都要求部署在同一个 host 上。如果是局域网，其实还是可以容忍的。 故开发环境下，是可以部署在不同的host上以供快速调试。
