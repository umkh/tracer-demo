package main

import (
	"fmt"
	"github.com/alloykh/tracer-demo/demo/protos/genproto/client_service"
	"github.com/alloykh/tracer-demo/log"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"time"

	grpcRetry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	GRPCOpenTracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
)

type Clients struct {
	UserClient      client_service.ClientServiceClient
	TearDowns       []func(log *log.Factory)
}

func NewGRPClients() (clients *Clients, err error) {

	clients = &Clients{}

	retryOpts := []grpcRetry.CallOption{
		grpcRetry.WithBackoff(grpcRetry.BackoffLinear(100 * time.Millisecond)),
		grpcRetry.WithCodes(codes.NotFound, codes.Aborted),
	}

	tracingOpts := []GRPCOpenTracing.Option{
		GRPCOpenTracing.WithTracer(opentracing.GlobalTracer()), // setting tracer
		//GRPCOpenTracing.WithOpName(func(method string) string {    // changing operation name
		//	return "hell no"
		//}),
	}

	interceptors := grpc.WithChainUnaryInterceptor(grpcRetry.UnaryClientInterceptor(retryOpts...), GRPCOpenTracing.UnaryClientInterceptor(tracingOpts...))

	userClient, tr, err := callToUserClient(grpc.WithInsecure(), interceptors)

	clients.UserClient = userClient

	clients.TearDowns = append(clients.TearDowns, tr)

	return
}

func callToUserClient(opt ...grpc.DialOption) (client_service.ClientServiceClient, func(log *log.Factory), error) {

	connStr := fmt.Sprintf("%v%v", "localhost", ":7050")

	conn, err := grpc.Dial(
		connStr,
		opt...,
	)

	if err != nil {
		return nil, nil, errors.Wrap(err, "grpc-clients-callToUserClient()")
	}

	tr := func(log *log.Factory) {
		log.Default().Debug("shutting down grpc client") // add name of the client
		if err := conn.Close(); err != nil {
			log.Default().Error("grpc client connection close", zap.Any("err", err.Error()))
		}
	}

	return client_service.NewClientServiceClient(conn), tr, nil

}

