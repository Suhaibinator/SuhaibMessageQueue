package server

import (
	"context"
	"log"
	"net"
	"time"

	pb "github.com/Suhaibinator/SuhaibMessageQueue/proto"
	"github.com/Suhaibinator/SuhaibMessageQueue/server/database"
	"google.golang.org/grpc"
)

type Server struct {
	pb.UnimplementedSuhaibMessageQueueServer
	driver     *database.DBDriver
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

func (s *Server) StreamProducer(sp pb.SuhaibMessageQueue_StreamProduceServer) error {
	for {
		// Receive a message from the client
		message, err := sp.Recv()
		if err != nil {
			return err
		}

		// Add the message to the topic
		err = s.driver.AddMessageToTopic(message.Topic, message.Message)
		if err != nil {
			return err
		}
	}
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
	offset := startOffset

	for {
		// Get the message at the current offset
		message, err := s.driver.GetMessageAtOffset(topic, offset)
		if err != nil {
			return err
		}

		if message == nil {
			log.Println("No messages at offset", offset)
			time.Sleep(time.Second) // Wait for a second before trying again
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

func (s *Server) Close() {
	s.grpcServer.Stop()
	s.driver.Close()
}

func NewServer(port string) *Server {
	driver, err := database.NewDBDriver()
	if err != nil {
		log.Fatalf("failed to create database driver: %v", err)
	}

	grpcServer := grpc.NewServer()
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
