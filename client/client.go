package client

import (
	"context"
	"io"
	"time"

	"github.com/Suhaibinator/SuhaibMessageQueue/config"
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
	conn, err := grpc.NewClient(address+":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*1024), grpc.MaxCallSendMsgSize(1024*1024*1024)))
	if err != nil {
		return nil, err
	}
	client := pb.NewSuhaibMessageQueueClient(conn)
	return &Client{conn: conn,
		client: client}, nil
}

// Produce sends a message to the server
func (c *Client) Produce(topic string, message []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	_, err := c.client.Produce(ctx, &pb.ProduceRequest{Topic: topic, Message: message})
	return err
}

// StreamProduce sends a stream of messages to the server
func (c *Client) StreamProduce(topic string, function func() ([]byte, error)) error {
	ctx := context.Background()

	stream, err := c.client.StreamProduce(ctx)
	if err != nil {
		return err
	}

	for {
		message, err := function()
		if err != nil {
			stream.CloseSend()
			return err
		}

		err = stream.Send(&pb.ProduceRequest{Topic: topic, Message: message})
		if err != nil {
			return err
		}
	}

}

func (c *Client) ConsumeEarliest(topic string) ([]byte, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	response, err := c.client.GetEarliestOffset(ctx, &pb.GetEarliestOffsetRequest{Topic: topic})
	if err != nil {
		return nil, -1, err
	}

	messageResponse, err := c.client.Consume(ctx, &pb.ConsumeRequest{Topic: topic, Offset: response.Offset})
	if err != nil {
		return nil, -1, err
	}
	return messageResponse.Message, response.Offset, err
}

func (c *Client) ConsumeLatest(topic string) ([]byte, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	response, err := c.client.GetLatestOffset(ctx, &pb.GetLatestOffsetRequest{Topic: topic})
	if err != nil {
		return nil, -1, err
	}

	messageResponse, err := c.client.Consume(ctx, &pb.ConsumeRequest{Topic: topic, Offset: response.Offset})
	if err != nil {
		return nil, -1, err
	}
	return messageResponse.Message, response.Offset, err
}

// Consume retrieves a message from the server
func (c *Client) Consume(topic string, offset int64) ([]byte, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	response, err := c.client.Consume(ctx, &pb.ConsumeRequest{Topic: topic, Offset: offset})
	if err != nil {
		return nil, -1, err
	}
	return response.Message, response.Offset, nil
}

func (c *Client) StreamConsume(topic string, startOffset int64, function func([]byte, int64) error) error {
	ctx := context.Background()

	stream, err := c.client.StreamConsume(ctx, &pb.ConsumeRequest{Topic: topic, Offset: startOffset})
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
		if response.Offset == config.SPECIAL_OFFSET_HEARTBEAT {
			// Heartbeat
			continue
		}
		// Process the message here.
		// For example, you can print it to the console:
		err = function(response.Message, response.Offset)
		if err != nil {
			stream.CloseSend()
			return err
		}

	}

	return nil
}

// CreateTopic creates a new topic on the server
func (c *Client) CreateTopic(topic string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := c.client.CreateTopic(ctx, &pb.CreateTopicRequest{Topic: topic})
	return err
}

// Close closes the connection to the server
func (c *Client) Close() {
	c.conn.Close()
}

// DeleteUntilOffset deletes all messages in a topic until a certain offset
func (c *Client) DeleteUntilOffset(topic string, offset int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := c.client.DeleteUntilOffset(ctx, &pb.DeleteUntilOffsetRequest{Topic: topic, Offset: offset})
	return err
}
