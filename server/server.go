package server

import (
	"context"
	"io"
	"log"
	"net"
	"time"

	"github.com/Suhaibinator/SuhaibMessageQueue/config"
	pb "github.com/Suhaibinator/SuhaibMessageQueue/proto"
	"github.com/Suhaibinator/SuhaibMessageQueue/server/database"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Server struct {
	pb.UnimplementedSuhaibMessageQueueServer
	driver     database.DBDriverInterface
	grpcServer *grpc.Server

	port string
}

func (s *Server) Produce(ctx context.Context, pr *pb.ProduceRequest) (*pb.ProduceResponse, error) {
	err := s.driver.AddMessageToTopic(pr.Topic, pr.Message)
	if err != nil {
		return nil, err
	}
	return &pb.ProduceResponse{}, nil
}

func (s *Server) StreamProduce(sp pb.SuhaibMessageQueue_StreamProduceServer) error {
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

func NewServer(port, dbPath string) *Server {
	driver, err := database.NewDBDriver(dbPath)
	if err != nil {
		log.Fatalf("failed to create database driver: %v", err)
	}

	var serverOpts []grpc.ServerOption
	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(1024*1024*1024))
	serverOpts = append(serverOpts, grpc.MaxSendMsgSize(1024*1024*1024))

	if config.ServerEnableMTLS {
		tlsConfig := config.LoadServerTLSConfig()
		if tlsConfig != nil {
			creds := credentials.NewTLS(tlsConfig)
			serverOpts = append(serverOpts, grpc.Creds(creds))
			log.Println("mTLS is enabled for the server.")
		} else {
			log.Println("Warning: Server mTLS is enabled in config, but failed to load TLS configuration. Server will start without mTLS.")
		}
	} else {
		log.Println("mTLS is not enabled for the server.")
	}

	grpcServer := grpc.NewServer(serverOpts...)
	pb.RegisterSuhaibMessageQueueServer(grpcServer, &Server{driver: driver})

	return &Server{
		driver:     driver,
		grpcServer: grpcServer,
		port:       port,
	}
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
