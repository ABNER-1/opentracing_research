package pkg

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
	"io"
)

func StarASpan(operationName string) opentracing.Span {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(operationName)
	return span
}

func StarAChildSpanFromID(id string, operationName string) opentracing.Span {
	tracer := opentracing.GlobalTracer()
	parentSpan, err := tracer.Extract(opentracing.TextMap, opentracing.TextMapCarrier{"": id})
	if err != nil {
		fmt.Println("error: ", err)
	}
	span := tracer.StartSpan(operationName, opentracing.ChildOf(parentSpan))
	return span
}

func GetSpanID(span opentracing.Span) string{
	tracer := opentracing.GlobalTracer()
	carrier := opentracing.TextMapCarrier{}
	tracer.Inject(span.Context(), opentracing.TextMap, carrier)
	requestID := carrier[""]
	return requestID
}

func InitTracer(serviceName string) (io.Closer, error) {
	cfg := jaegercfg.Configuration{
		ServiceName: serviceName,
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans: true,
		},
	}

	jLogger := jaegerlog.StdLogger
	tracer, closer, err := cfg.NewTracer(
		jaegercfg.Logger(jLogger),
		jaegercfg.Injector(opentracing.TextMap, jaeger.NewTextMapPropagator(&jaeger.HeadersConfig{}, jaeger.Metrics{})),
		jaegercfg.Extractor(opentracing.TextMap, jaeger.NewTextMapPropagator(&jaeger.HeadersConfig{}, jaeger.Metrics{})),
	)
	if err != nil {
		fmt.Println("error", err)
	}
	// Set the singleton opentracing.Tracer with the Jaeger tracer.
	opentracing.SetGlobalTracer(tracer)

	return closer, err
}
