package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/example/iot-device-management/internal/config"
	"google.golang.org/grpc"
)

type Server struct {
	cfg    config.GRPCConfig
	server *grpc.Server
	svc    *Service
}

func NewServer(cfg config.GRPCConfig, svc *Service) *Server {
	return &Server{cfg: cfg, svc: svc}
}

func (s *Server) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.cfg.Listen)
	if err != nil {
		return fmt.Errorf("listen grpc %s: %w", s.cfg.Listen, err)
	}
	opts := []grpc.ServerOption{
		grpc.ForceServerCodec(jsonCodec{}),
		grpc.MaxRecvMsgSize(s.cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(s.cfg.MaxSendMsgSize),
		grpc.ConnectionTimeout(s.cfg.ConnectionTimeout),
	}
	s.server = grpc.NewServer(opts...)
	s.server.RegisterService(&grpc.ServiceDesc{
		ServiceName: "iot.v1.IoTService",
		HandlerType: (*any)(nil),
		Methods: []grpc.MethodDesc{
			{MethodName: "Health", Handler: unary(s.svc.Health)},
			{MethodName: "GetDevice", Handler: unary(s.svc.GetDevice)},
			{MethodName: "IngestTelemetry", Handler: unary(s.svc.IngestTelemetry)},
			{MethodName: "EnqueueCommand", Handler: unary(s.svc.EnqueueCommand)},
			{MethodName: "GetTwin", Handler: unary(s.svc.GetTwin)},
			{MethodName: "QueryTimeSeries", Handler: unary(s.svc.QueryTimeSeries)},
		},
	}, s.svc)
	go func() {
		<-ctx.Done()
		done := make(chan struct{})
		go func() {
			s.server.GracefulStop()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			s.server.Stop()
		}
	}()
	return s.server.Serve(listener)
}
