package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	smqv1 "github.com/Suhaibinator/SuhaibMessageQueue/api/smq/v1"
	"github.com/Suhaibinator/SuhaibMessageQueue/internal/queue"
	"github.com/Suhaibinator/SuhaibMessageQueue/internal/store/sqlite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// ServerOptions allows for programmatic configuration of the server.
type ServerOptions struct {
	EnableMTLS        bool
	TLSConfig         *tls.Config
	MaxRecvMsgSize    int
	MaxSendMsgSize    int
	AllowRemoteWrites bool
}

type Server struct {
	smqv1.UnimplementedSuhaibMessageQueueServer
	driver     queue.Store
	grpcServer *grpc.Server
	health     *health.Server

	port              string
	allowRemoteWrites bool
	closeOnce         sync.Once
}

func (s *Server) Produce(ctx context.Context, pr *smqv1.ProduceRequest) (*smqv1.ProduceResponse, error) {
	if !s.allowRemoteWrites {
		return nil, grpcError(queue.ErrRemoteWritesDisabled)
	}
	offset, err := s.driver.AddMessage(pr.Topic, pr.Message)
	if err != nil {
		return nil, grpcError(err)
	}
	return &smqv1.ProduceResponse{Offset: offset}, nil
}

func (s *Server) StreamProduce(sp smqv1.SuhaibMessageQueue_StreamProduceServer) error {
	if !s.allowRemoteWrites {
		return grpcError(queue.ErrRemoteWritesDisabled)
	}
	var lastOffset int64
	for {
		// Receive a message from the client
		message, err := sp.Recv()
		if err != nil {
			if err == io.EOF {
				// End of stream, send response
				return sp.SendAndClose(&smqv1.ProduceResponse{Offset: lastOffset})
			}
			return err
		}

		// Add the message to the topic
		lastOffset, err = s.driver.AddMessage(message.Topic, message.Message)
		if err != nil {
			return grpcError(err)
		}
	}
}

// Register registers the SMQ service, backed by store, on registrar. The
// caller owns both the store and the gRPC server lifecycle; closing the gRPC
// server does not close the store.
func Register(registrar grpc.ServiceRegistrar, store queue.Store, allowRemoteWrites bool) {
	smqv1.RegisterSuhaibMessageQueueServer(registrar, &Server{driver: store, allowRemoteWrites: allowRemoteWrites})
}

func (s *Server) GetDriver() queue.Store {
	return s.driver
}

func (s *Server) Consume(ctx context.Context, cr *smqv1.ConsumeRequest) (*smqv1.ConsumeResponse, error) {
	message, err := s.driver.GetMessageAtOffset(cr.Topic, cr.Offset)
	if err != nil {
		return nil, grpcError(err)
	}
	return &smqv1.ConsumeResponse{Message: message, Offset: cr.Offset}, nil
}

func (s *Server) StreamConsume(cr *smqv1.ConsumeRequest, cs smqv1.SuhaibMessageQueue_StreamConsumeServer) error {
	return s.streamMessages(cr.Topic, cs, cr.Offset)
}

func (s *Server) streamMessages(topic string, cs smqv1.SuhaibMessageQueue_StreamConsumeServer, startOffset int64) error {

	earliestOffset, err := s.driver.GetEarliestOffset(topic)

	if err != nil {
		return grpcError(err)
	}
	offset := startOffset
	if startOffset < earliestOffset {
		offset = earliestOffset
	}

	consecutiveNoMessages := 0
	for {
		// Get the message at the current offset
		message, err := s.driver.GetMessageAtOffset(topic, offset)
		if err != nil && !errors.Is(err, queue.ErrOffsetGreaterThanLatest) && !errors.Is(err, queue.ErrNotMessageAtOffset) {
			return grpcError(err)
		}

		if err != nil || message == nil {
			timer := time.NewTimer(time.Second)
			select {
			case <-cs.Context().Done():
				timer.Stop()
				return status.FromContextError(cs.Context().Err()).Err()
			case <-timer.C:
			}
			if consecutiveNoMessages < 10 {
				consecutiveNoMessages++
			} else {
				err = cs.Send(&smqv1.ConsumeResponse{Message: []byte{}, Offset: queue.HeartbeatOffset})
				if err != nil {
					return err
				}
			}
			continue
		}
		consecutiveNoMessages = 0

		// Send the message to the client
		err = cs.Send(&smqv1.ConsumeResponse{Message: message, Offset: offset})
		if err != nil {
			return err
		}
		// Increment the offset for the next iteration
		offset++
	}
}

func (s *Server) CreateTopic(ctx context.Context, tr *smqv1.CreateTopicRequest) (*smqv1.CreateTopicResponse, error) {
	if !s.allowRemoteWrites {
		return nil, grpcError(queue.ErrRemoteWritesDisabled)
	}
	err := s.driver.CreateTopic(tr.Topic)
	if err != nil {
		return nil, grpcError(err)
	}
	return &smqv1.CreateTopicResponse{}, nil
}

func (s *Server) GetEarliestMessageFromTopic(ctx context.Context, gr *smqv1.GetEarliestOffsetRequest) (*smqv1.GetEarliestOffsetResponse, error) {
	_, offset, err := s.driver.GetEarliestMessageFromTopic(gr.Topic)
	if err != nil {
		return nil, grpcError(err)
	}
	return &smqv1.GetEarliestOffsetResponse{Offset: offset}, nil
}

func (s *Server) GetLatestMessageFromTopic(ctx context.Context, gr *smqv1.GetLatestOffsetRequest) (*smqv1.GetLatestOffsetResponse, error) {
	_, offset, err := s.driver.GetLatestMessageFromTopic(gr.Topic)
	if err != nil {
		return nil, grpcError(err)
	}
	return &smqv1.GetLatestOffsetResponse{Offset: offset}, nil
}

func (s *Server) GetEarliestOffset(ctx context.Context, gr *smqv1.GetEarliestOffsetRequest) (*smqv1.GetEarliestOffsetResponse, error) {
	offset, err := s.driver.GetEarliestOffset(gr.Topic)
	if err != nil {
		return nil, grpcError(err)
	}
	return &smqv1.GetEarliestOffsetResponse{Offset: offset}, nil
}

func (s *Server) GetLatestOffset(ctx context.Context, gr *smqv1.GetLatestOffsetRequest) (*smqv1.GetLatestOffsetResponse, error) {
	offset, err := s.driver.GetLatestOffset(gr.Topic)
	if err != nil {
		return nil, grpcError(err)
	}
	return &smqv1.GetLatestOffsetResponse{Offset: offset}, nil
}

func (s *Server) Close() {
	s.closeOnce.Do(func() {
		s.health.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
		s.grpcServer.Stop()
		if err := s.driver.Close(); err != nil {
			log.Printf("close store: %v", err)
		}
	})
}

// NewServer constructs the storage and gRPC server. Startup errors are
// returned to the command instead of terminating the process from a library.
func NewServer(port, dbPath string, opts *ServerOptions) (*Server, error) {
	driver, err := sqlite.NewDBDriver(dbPath)
	if err != nil {
		return nil, fmt.Errorf("create SQLite store: %w", err)
	}

	var serverOpts []grpc.ServerOption

	// Configure MaxRecvMsgSize and MaxSendMsgSize
	maxRecvMsgSize := 1024 * 1024 * 1024 // Default 1GB
	maxSendMsgSize := 1024 * 1024 * 1024 // Default 1GB

	if opts != nil {
		if opts.MaxRecvMsgSize > 0 {
			maxRecvMsgSize = opts.MaxRecvMsgSize
			log.Printf("MaxRecvMsgSize set to %d from options.", maxRecvMsgSize)
		}
		if opts.MaxSendMsgSize > 0 {
			maxSendMsgSize = opts.MaxSendMsgSize
			log.Printf("MaxSendMsgSize set to %d from options.", maxSendMsgSize)
		}
	}
	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(maxRecvMsgSize))
	serverOpts = append(serverOpts, grpc.MaxSendMsgSize(maxSendMsgSize))

	if opts != nil && opts.EnableMTLS {
		if opts.TLSConfig == nil {
			driver.Close()
			return nil, errors.New("mTLS is enabled but TLSConfig is nil")
		}
		serverOpts = append(serverOpts, grpc.Creds(credentials.NewTLS(opts.TLSConfig)))
	}

	grpcServer := grpc.NewServer(serverOpts...)
	healthServer := health.NewServer()
	allowWrites := true
	if opts != nil {
		allowWrites = opts.AllowRemoteWrites
	}

	srv := &Server{
		driver:            driver,
		grpcServer:        grpcServer,
		health:            healthServer,
		port:              port,
		allowRemoteWrites: allowWrites,
	}
	smqv1.RegisterSuhaibMessageQueueServer(grpcServer, srv)
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	return srv, nil
}

// Serve listens until the context is canceled or gRPC returns an error.
func (s *Server) Serve(ctx context.Context) error {
	lis, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return fmt.Errorf("listen on port %s: %w", s.port, err)
	}
	log.Printf("Server is running on port %s...\n", s.port)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- s.grpcServer.Serve(lis)
	}()

	select {
	case err := <-serveErr:
		s.Close()
		if err != nil {
			return fmt.Errorf("serve gRPC: %w", err)
		}
		return nil
	case <-ctx.Done():
		stopped := make(chan struct{})
		go func() {
			s.grpcServer.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-time.After(10 * time.Second):
			s.grpcServer.Stop()
		}
		s.Close()
		return nil
	}
}

func (s *Server) DeleteUntilOffset(ctx context.Context, dr *smqv1.DeleteUntilOffsetRequest) (*smqv1.DeleteUntilOffsetResponse, error) {
	if !s.allowRemoteWrites {
		return nil, grpcError(queue.ErrRemoteWritesDisabled)
	}
	err := s.driver.DeleteMessagesUntilOffset(dr.Topic, dr.Offset)
	if err != nil {
		return nil, grpcError(err)
	}
	return &smqv1.DeleteUntilOffsetResponse{}, nil
}

// Connect is a placeholder implementation for the Connect method required by the interface
func (s *Server) Connect(ctx context.Context, req *smqv1.ConnectRequest) (*smqv1.ConnectResponse, error) {
	return &smqv1.ConnectResponse{}, nil
}

func (s *Server) BulkRetrieve(ctx context.Context, brr *smqv1.BulkRetrieveRequest) (*smqv1.BulkRetrieveResponse, error) {
	dbMessages, nextOffset, err := s.driver.GetMessagesAfterOffsetWithLimit(brr.Topic, brr.StartOffset, int(brr.Limit))
	if err != nil {
		return nil, grpcError(err)
	}

	retrievedMessages := make([]*smqv1.RetrievedMessage, len(dbMessages))
	for i, dbMsg := range dbMessages {
		retrievedMessages[i] = &smqv1.RetrievedMessage{
			Message: dbMsg.Message,
			Offset:  dbMsg.Offset,
		}
	}

	return &smqv1.BulkRetrieveResponse{
		Messages:   retrievedMessages,
		Count:      int32(len(retrievedMessages)),
		NextOffset: nextOffset,
	}, nil
}

func grpcError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	case errors.Is(err, queue.ErrTopicNotFound), errors.Is(err, queue.ErrNotMessageAtOffset):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, queue.ErrTopicAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, queue.ErrOffsetGreaterThanLatest):
		return status.Error(codes.OutOfRange, err.Error())
	case errors.Is(err, queue.ErrTopicIsEmpty):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, queue.ErrRemoteWritesDisabled):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, queue.ErrInvalidTopic):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, queue.ErrStoreClosed):
		return status.Error(codes.Unavailable, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
