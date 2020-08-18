# how to use
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