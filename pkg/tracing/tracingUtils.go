package tracing

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	jaegerConfig "github.com/uber/jaeger-client-go/config"
	jaegerLog "github.com/uber/jaeger-client-go/log"
	metrics "github.com/uber/jaeger-lib/metrics"
	"gopkg.in/yaml.v2"
	"io"
	"testOpentracing/pkg"
)

const defaultTraceContextHeaderName = "trace_header"

var headerConfig = jaeger.HeadersConfig{}

//todo : all fmt.print should be replaced by logging
func InitTracer(serviceName string, localAgentHostPort string, debug bool) (io.Closer, error) {
	reporter := jaegerConfig.ReporterConfig{
		LogSpans:           false,
		LocalAgentHostPort: localAgentHostPort,
	}
	sampler := jaegerConfig.SamplerConfig{}
	if debug {
		sampler.Type = jaeger.SamplerTypeConst
		sampler.Param = 1
		reporter.LogSpans = true
	} else {
		// in production environment, const sampler will affect app performance
		sampler.Type = jaeger.SamplerTypeProbabilistic
		sampler.Param = 0.2
	}
	// will be edited after definite log and metrics, this is all consistent in system,
	// so have no need to configure it
	jLogger := jaegerLog.StdLogger
	jMetrics := metrics.NullFactory
	return initTracer(serviceName, sampler, reporter, jLogger, jMetrics)
}

func initTracer(serviceName string, sampler jaegerConfig.SamplerConfig,
	reporter jaegerConfig.ReporterConfig, jLogger jaeger.Logger,
	jMetrics metrics.Factory) (io.Closer, error) {
	headerConfig = jaeger.HeadersConfig{
		TraceContextHeaderName: defaultTraceContextHeaderName,
	}
	cfg := jaegerConfig.Configuration{
		ServiceName: serviceName,
		Sampler:     &sampler,
		Reporter:    &reporter,
		Headers:     &headerConfig,
	}

	tracer, closer, err := cfg.NewTracer(
		jaegerConfig.Logger(jLogger),
		jaegerConfig.Metrics(jMetrics),
	)
	if err != nil {
		fmt.Println("error", err)
	}
	opentracing.SetGlobalTracer(tracer)
	return closer, err
}

func InitTracerFromYAML(yamlPath string) (io.Closer, error) {
	cfg := jaegerConfig.Configuration{}
	str := pkg.ReadYamlFromFile(yamlPath)
	err := yaml.Unmarshal([]byte(str), &cfg)
	tracer, closer, err := cfg.NewTracer()
	headerConfig = *cfg.Headers
	if err != nil {
		fmt.Println("error", err)
	}
	opentracing.SetGlobalTracer(tracer)
	return closer, err
}

func CreateSpan(operationName string) opentracing.Span {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(operationName)
	return span
}

func CreateChildFromSC(operationName string, sc opentracing.SpanContext) opentracing.Span {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(operationName, opentracing.ChildOf(sc))
	return span
}

func CreateFollowerFromSC(operationName string, sc opentracing.SpanContext) opentracing.Span {
	tracer := opentracing.GlobalTracer()
	span := tracer.StartSpan(operationName, opentracing.FollowsFrom(sc))
	return span
}

func CreateChildFromCarrier(operationName string, carrier string) opentracing.Span {
	parentSC := extractSCFromCarrier(carrier)
	return CreateChildFromSC(operationName, parentSC)
}

func CreateFollowerFromCarrier(operationName string, carrier string) opentracing.Span {
	parentSC := extractSCFromCarrier(carrier)
	return CreateFollowerFromSC(operationName, parentSC)
}

/* get carrier for IPC span relationship construction
	for example:
	1. get carrier of span
	2. transfer carrier to the next process by sockets / http / grpc and so on
    3. in next process, invoke CreateChildFromCarrier to construct a child span of this
*/
func GetCarrier(span opentracing.Span) string {
	tracer := opentracing.GlobalTracer()
	textMap := opentracing.TextMapCarrier{}

	err := tracer.Inject(span.Context(), opentracing.TextMap, textMap)
	if err != nil {
		fmt.Println("error: ", err)
	}
	carrier := textMap[headerConfig.TraceContextHeaderName]
	return carrier
}

func extractSCFromCarrier(carrier string) opentracing.SpanContext {
	tracer := opentracing.GlobalTracer()
	parentSpan, err := tracer.Extract(opentracing.TextMap, opentracing.TextMapCarrier{headerConfig.TraceContextHeaderName: carrier})
	if err != nil {
		fmt.Println("error: ", err)
	}
	return parentSpan
}
