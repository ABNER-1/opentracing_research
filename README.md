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
