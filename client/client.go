package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/Suhaibinator/SuhaibMessageQueue/config"
	pb "github.com/Suhaibinator/SuhaibMessageQueue/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a struct that holds the connection to the server
type Client struct {
	conn   *grpc.ClientConn
	client pb.SuhaibMessageQueueClient
}

// NewClient creates a new client
// It accepts an optional tlsCfg to enable mTLS.
// If tlsCfg is nil, it defaults to an insecure connection.
func NewClient(address, port string, tlsCfg *config.ClientTLSConfig) (*Client, error) {
	var creds credentials.TransportCredentials
	var err error

	if tlsCfg != nil {
		// mTLS is enabled, load credentials
		clientCert, errLoad := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
		if errLoad != nil {
			return nil, fmt.Errorf("client: failed to load client certificate/key: %w", errLoad)
		}

		caCertPEM, errLoad := os.ReadFile(tlsCfg.CAFile)
		if errLoad != nil {
			return nil, fmt.Errorf("client: failed to read CA certificate: %w", errLoad)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCertPEM) {
			return nil, fmt.Errorf("client: failed to append CA certificate to pool")
		}

		// Extract the hostname from the address for ServerName
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("client: invalid address format: %w", err)
		}

		tlsCredentials := &tls.Config{
			Certificates: []tls.Certificate{clientCert},
			RootCAs:      caCertPool,
			ServerName:   host, // Set the ServerName for proper certificate verification
		}
		creds = credentials.NewTLS(tlsCredentials)
	} else {
		// mTLS is not enabled, use insecure credentials
		creds = insecure.NewCredentials()
	}

	// Establish the gRPC connection
	conn, err := grpc.NewClient(address+":"+port,
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(1024*1024*1024), // 1GB
			grpc.MaxCallSendMsgSize(1024*1024*1024), // 1GB
		),
	)
	if err != nil {
		return nil, fmt.Errorf("client: failed to connect to server: %w", err)
	}

	client := pb.NewSuhaibMessageQueueClient(conn)
	return &Client{conn: conn, client: client}, nil
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

// BulkRetrieve retrieves a batch of messages from the server.
func (c *Client) BulkRetrieve(topic string, startOffset int64, limit int32) (*pb.BulkRetrieveResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	response, err := c.client.BulkRetrieve(ctx, &pb.BulkRetrieveRequest{
		Topic:       topic,
		StartOffset: startOffset,
		Limit:       limit,
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}
