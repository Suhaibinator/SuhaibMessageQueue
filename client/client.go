package client

import (
	"context"
	"io"
	"time"

	pb "github.com/Suhaibinator/SuhaibMessageQueue/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a struct that holds the connection to the server
type Client struct {
	conn   *grpc.ClientConn
	client pb.SuhaibMessageQueueClient
}

// NewClient creates a new client
func NewClient(address, port string) (*Client, error) {
	conn, err := grpc.NewClient(address+":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	client := pb.NewSuhaibMessageQueueClient(conn)
	return &Client{conn: conn,
		client: client}, nil
}

// Produce sends a message to the server
func (c *Client) Produce(topic string, message []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := c.client.Produce(ctx, &pb.ProduceRequest{Topic: topic, Message: message})
	return err
}

func (c *Client) Consume(topic string, function func([]byte, int64) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	stream, err := c.client.Consume(ctx, &pb.ConsumeRequest{Topic: topic})
	if err != nil {
		return err
	}

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			// The stream has been closed by the server.
			break
		}
		if err != nil {
			return err
		}

		// Process the message here.
		// For example, you can print it to the console:
		err = function(response.Message, response.Offset)
		if err != nil {
			return err
		}

	}

	return nil
}
