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

### deploy with k8s (untried)

1. deploy jaeger operator in k8s environment.

2. prepare external storage such as stable ES cluster.

3. deploy production component (agent / collector / ingester / query)

4. prepare MQ such as Kafka in k8s if need (need ingester)


