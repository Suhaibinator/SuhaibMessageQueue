package server

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"time"

	"github.com/Suhaibinator/SuhaibMessageQueue/config"
	smqerrors "github.com/Suhaibinator/SuhaibMessageQueue/errors"
	"github.com/Suhaibinator/SuhaibMessageQueue/internal/server/database"
	pb "github.com/Suhaibinator/SuhaibMessageQueue/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	pb.UnimplementedSuhaibMessageQueueServer
	driver     database.DBDriverInterface
	grpcServer *grpc.Server

	port              string
	allowRemoteWrites bool
}

func (s *Server) Produce(ctx context.Context, pr *pb.ProduceRequest) (*pb.ProduceResponse, error) {
	if !s.allowRemoteWrites {
		return nil, smqerrors.ErrRemoteWritesDisabled
	}
	err := s.driver.AddMessageToTopic(pr.Topic, pr.Message)
	if err != nil {
		return nil, err
	}
	return &pb.ProduceResponse{}, nil
}

func (s *Server) StreamProduce(sp pb.SuhaibMessageQueue_StreamProduceServer) error {
	if !s.allowRemoteWrites {
		return smqerrors.ErrRemoteWritesDisabled
	}
	for {
		// Receive a message from the client
		message, err := sp.Recv()
		if err != nil {
			if err == io.EOF {
				// End of stream, send response
				return sp.SendAndClose(&pb.ProduceResponse{})
			}
			return err
		}

		// Add the message to the topic
		err = s.driver.AddMessageToTopic(message.Topic, message.Message)
		if err != nil {
			return err
		}
	}
}

func (s *Server) GetDriver() database.DBDriverInterface {
	return s.driver
}

func (s *Server) Consume(ctx context.Context, cr *pb.ConsumeRequest) (*pb.ConsumeResponse, error) {
	message, err := s.driver.GetMessageAtOffset(cr.Topic, cr.Offset)
	if err != nil {
		return nil, err
	}
	return &pb.ConsumeResponse{Message: message, Offset: cr.Offset}, nil
}

func (s *Server) StreamConsume(cr *pb.ConsumeRequest, cs pb.SuhaibMessageQueue_StreamConsumeServer) error {
	return s.streamMessages(cr.Topic, cs, cr.Offset)
}

func (s *Server) streamMessages(topic string, cs pb.SuhaibMessageQueue_StreamConsumeServer, startOffset int64) error {

	earliestOffset, err := s.driver.GetEarliestOffset(topic)

	if err != nil {
		return err
	}
	offset := startOffset
	if startOffset < earliestOffset {
		offset = earliestOffset
	}

	consecutiveNoMessages := 0
	for {
		// Get the message at the current offset
		message, err := s.driver.GetMessageAtOffset(topic, offset)
		if err != nil {
			return err
		}

		if message == nil {
			log.Println("No messages at offset", offset)
			time.Sleep(time.Second) // Wait for a second before trying again
			if consecutiveNoMessages < 10 {
				consecutiveNoMessages++
			} else {
				err = cs.Send(&pb.ConsumeResponse{Message: []byte{}, Offset: config.SPECIAL_OFFSET_HEARTBEAT})
				if err != nil {
					return err
				}
			}
			continue
		}

		// Send the message to the client
		err = cs.Send(&pb.ConsumeResponse{Message: message, Offset: offset})
		if err != nil {
			return err
		}
		// Increment the offset for the next iteration
		offset++
	}
}

func (s *Server) CreateTopic(ctx context.Context, tr *pb.CreateTopicRequest) (*pb.CreateTopicResponse, error) {
	if !s.allowRemoteWrites {
		return nil, smqerrors.ErrRemoteWritesDisabled
	}
	err := s.driver.CreateTopic(tr.Topic)
	if err != nil {
		return nil, err
	}
	return &pb.CreateTopicResponse{}, nil
}

func (s *Server) GetEarliestMessageFromTopic(ctx context.Context, gr *pb.GetEarliestOffsetRequest) (*pb.GetEarliestOffsetResponse, error) {
	_, offset, err := s.driver.GetEarliestMessageFromTopic(gr.Topic)
	if err != nil {
		return nil, err
	}
	return &pb.GetEarliestOffsetResponse{Offset: offset}, nil
}

func (s *Server) GetLatestMessageFromTopic(ctx context.Context, gr *pb.GetLatestOffsetRequest) (*pb.GetLatestOffsetResponse, error) {
	_, offset, err := s.driver.GetLatestMessageFromTopic(gr.Topic)
	if err != nil {
		return nil, err
	}
	return &pb.GetLatestOffsetResponse{Offset: offset}, nil
}

func (s *Server) GetEarliestOffset(ctx context.Context, gr *pb.GetEarliestOffsetRequest) (*pb.GetEarliestOffsetResponse, error) {
	offset, err := s.driver.GetEarliestOffset(gr.Topic)
	if err != nil {
		return nil, err
	}
	return &pb.GetEarliestOffsetResponse{Offset: offset}, nil
}

func (s *Server) GetLatestOffset(ctx context.Context, gr *pb.GetLatestOffsetRequest) (*pb.GetLatestOffsetResponse, error) {
	offset, err := s.driver.GetLatestOffset(gr.Topic)
	if err != nil {
		return nil, err
	}
	return &pb.GetLatestOffsetResponse{Offset: offset}, nil
}

func (s *Server) Close() {
	s.grpcServer.Stop()
	s.driver.Close()
}

func NewServer(port, dbPath string, opts *ServerOptions) *Server {
	driver, err := database.NewDBDriver(dbPath)
	if err != nil {
		log.Fatalf("failed to create database driver: %v", err)
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

	// Configure mTLS
	if opts != nil {
		log.Println("Using mTLS configuration from ServerOptions.")
		if opts.EnableMTLS {
			if opts.TLSConfig != nil {
				creds := credentials.NewTLS(opts.TLSConfig)
				serverOpts = append(serverOpts, grpc.Creds(creds))
				log.Println("mTLS is enabled for the server via options.")
			} else {
				log.Println("Warning: Server mTLS is enabled in options, but TLSConfig is nil. Server will start without mTLS.")
			}
		} else {
			log.Println("mTLS is not enabled for the server via options.")
		}
	} else {
		log.Println("Using mTLS configuration from global config.")
		if config.ServerEnableMTLS {
			tlsConfig := config.LoadServerTLSConfig()
			if tlsConfig != nil {
				creds := credentials.NewTLS(tlsConfig)
				serverOpts = append(serverOpts, grpc.Creds(creds))
				log.Println("mTLS is enabled for the server via global config.")
			} else {
				log.Println("Warning: Server mTLS is enabled in global config, but failed to load TLS configuration. Server will start without mTLS.")
			}
		} else {
			log.Println("mTLS is not enabled for the server via global config.")
		}
	}

	grpcServer := grpc.NewServer(serverOpts...)
	allowWrites := config.AllowRemoteWrites
	if opts != nil {
		allowWrites = opts.AllowRemoteWrites
	}

	srv := &Server{
		driver:            driver,
		grpcServer:        grpcServer,
		port:              port,
		allowRemoteWrites: allowWrites,
	}
	pb.RegisterSuhaibMessageQueueServer(grpcServer, srv)
	return srv
}

func (s *Server) Start() {
	lis, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		log.Printf("failed to listen: %v\n", err)
	}
	log.Printf("Server is running on port %s...\n", s.port)
	if err := s.grpcServer.Serve(lis); err != nil {
		log.Printf("failed to serve: %v\n", err)
	}
}

func (s *Server) DeleteUntilOffset(ctx context.Context, dr *pb.DeleteUntilOffsetRequest) (*pb.DeleteUntilOffsetResponse, error) {
	if !s.allowRemoteWrites {
		return nil, smqerrors.ErrRemoteWritesDisabled
	}
	err := s.driver.DeleteMessagesUntilOffset(dr.Topic, dr.Offset)
	if err != nil {
		return nil, err
	}
	return &pb.DeleteUntilOffsetResponse{}, nil
}

func (s *Server) Debug() {
	s.driver.Debug()
}

// Connect is a placeholder implementation for the Connect method required by the interface
func (s *Server) Connect(ctx context.Context, req *pb.ConnectRequest) (*pb.ConnectResponse, error) {
	return &pb.ConnectResponse{}, nil
}

func (s *Server) BulkRetrieve(ctx context.Context, brr *pb.BulkRetrieveRequest) (*pb.BulkRetrieveResponse, error) {
	dbMessages, nextOffset, err := s.driver.GetMessagesAfterOffsetWithLimit(brr.Topic, brr.StartOffset, int(brr.Limit))
	if err != nil {
		return nil, err
	}

	retrievedMessages := make([]*pb.RetrievedMessage, len(dbMessages))
	for i, dbMsg := range dbMessages {
		retrievedMessages[i] = &pb.RetrievedMessage{
			Message: dbMsg.Message,
			Offset:  dbMsg.Offset,
		}
	}

	return &pb.BulkRetrieveResponse{
		Messages:   retrievedMessages,
		Count:      int32(len(retrievedMessages)),
		NextOffset: nextOffset,
	}, nil
}
