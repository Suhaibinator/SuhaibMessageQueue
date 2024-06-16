package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"SuhaibMessageQueue/database"
	pb "SuhaibMessageQueue/proto"

	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedSuhaibMessageQueueServer
	driver *database.DBDriver
}

func (s *server) Produce(ctx context.Context, pr *pb.ProduceRequest) (*pb.ProduceResponse, error) {
	err := s.driver.AddMessageToTopic(pr.Topic, pr.Message)
	if err != nil {
		return nil, err
	}
	return &pb.ProduceResponse{}, nil
}

func (s *server) Consume(cr *pb.ConsumeRequest, cs pb.SuhaibMessageQueue_ConsumeServer) error {
	for {
		message, offset, err := s.driver.GetEarliestMessageFromTopic(cr.Topic)
		if message == nil && err == nil {
			fmt.Println("No messages")
		} else if err != nil {
			return err
		} else {
			// Send the message to the client
			err = cs.Send(&pb.ConsumeResponse{Message: message, Offset: offset})
			if err != nil {
				return err
			}
		}

	}
}

func main() {

	driver, err := database.NewDBDriver()
	if err != nil {
		log.Fatalf("failed to create database driver: %v", err)
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterSuhaibMessageQueueServer(s, &server{driver: driver})

	fmt.Println("Server is running on port 50051...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
