package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	smqv1 "github.com/Suhaibinator/SuhaibMessageQueue/api/smq/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is a struct that holds the connection to the server
type Client struct {
	conn           *grpc.ClientConn
	client         smqv1.SuhaibMessageQueueClient
	requestTimeout time.Duration
}

// NewClient creates a new client
// It accepts an optional tlsCfg to enable mTLS.
// If tlsCfg is nil, it defaults to an insecure connection.
func NewClient(address, port string, tlsCfg *TLSConfig) (*Client, error) {
	return New(Options{Address: address, Port: port, TLS: tlsCfg})
}

// New creates a client from explicit options.
func New(options Options) (*Client, error) {
	options, err := options.withDefaults()
	if err != nil {
		return nil, err
	}

	var creds credentials.TransportCredentials

	if options.TLS != nil {
		// mTLS is enabled, load credentials
		clientCert, errLoad := tls.LoadX509KeyPair(options.TLS.CertFile, options.TLS.KeyFile)
		if errLoad != nil {
			return nil, fmt.Errorf("client: failed to load client certificate/key: %w", errLoad)
		}

		caCertPEM, errLoad := os.ReadFile(options.TLS.CAFile)
		if errLoad != nil {
			return nil, fmt.Errorf("client: failed to read CA certificate: %w", errLoad)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCertPEM) {
			return nil, fmt.Errorf("client: failed to append CA certificate to pool")
		}

		// Determine the hostname for TLS verification. The address may not
		// contain a port, so attempt to split host and port. If splitting
		// fails, use the address as-is.
		host := options.Address
		if strings.Contains(options.Address, ":") {
			if h, _, err := net.SplitHostPort(options.Address); err == nil {
				host = h
			}
		}

		tlsCredentials := &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{clientCert},
			RootCAs:      caCertPool,
			ServerName:   host, // Use the resolved host for certificate verification
		}
		creds = credentials.NewTLS(tlsCredentials)
	} else {
		// mTLS is not enabled, use insecure credentials
		creds = insecure.NewCredentials()
	}

	// Establish the gRPC connection
	conn, err := grpc.NewClient(net.JoinHostPort(options.Address, options.Port),
		grpc.WithTransportCredentials(creds),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(options.MaxReceiveMsgSize),
			grpc.MaxCallSendMsgSize(options.MaxSendMsgSize),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("client: failed to connect to server: %w", err)
	}

	client := smqv1.NewSuhaibMessageQueueClient(conn)
	return &Client{conn: conn, client: client, requestTimeout: options.RequestTimeout}, nil
}

// Produce sends a message to the server
func (c *Client) Produce(topic string, message []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.requestTimeout)
	defer cancel()

	_, err := c.client.Produce(ctx, &smqv1.ProduceRequest{Topic: topic, Message: message})
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

		err = stream.Send(&smqv1.ProduceRequest{Topic: topic, Message: message})
		if err != nil {
			return err
		}
	}

}

func (c *Client) ConsumeEarliest(topic string) ([]byte, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.requestTimeout)
	defer cancel()

	response, err := c.client.GetEarliestOffset(ctx, &smqv1.GetEarliestOffsetRequest{Topic: topic})
	if err != nil {
		return nil, -1, err
	}

	messageResponse, err := c.client.Consume(ctx, &smqv1.ConsumeRequest{Topic: topic, Offset: response.Offset})
	if err != nil {
		return nil, -1, err
	}
	return messageResponse.Message, response.Offset, err
}

func (c *Client) ConsumeLatest(topic string) ([]byte, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.requestTimeout)
	defer cancel()

	response, err := c.client.GetLatestOffset(ctx, &smqv1.GetLatestOffsetRequest{Topic: topic})
	if err != nil {
		return nil, -1, err
	}

	messageResponse, err := c.client.Consume(ctx, &smqv1.ConsumeRequest{Topic: topic, Offset: response.Offset})
	if err != nil {
		return nil, -1, err
	}
	return messageResponse.Message, response.Offset, err
}

// Consume retrieves a message from the server
func (c *Client) Consume(topic string, offset int64) ([]byte, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.requestTimeout)
	defer cancel()

	response, err := c.client.Consume(ctx, &smqv1.ConsumeRequest{Topic: topic, Offset: offset})
	if err != nil {
		return nil, -1, err
	}
	return response.Message, response.Offset, nil
}

func (c *Client) StreamConsume(topic string, startOffset int64, function func([]byte, int64) error) error {
	ctx := context.Background()

	stream, err := c.client.StreamConsume(ctx, &smqv1.ConsumeRequest{Topic: topic, Offset: startOffset})
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
		if response.Offset == HeartbeatOffset {
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

	_, err := c.client.CreateTopic(ctx, &smqv1.CreateTopicRequest{Topic: topic})
	return err
}

// Close closes the connection to the server
func (c *Client) Close() error {
	return c.conn.Close()
}

// DeleteUntilOffset deletes all messages in a topic until a certain offset
func (c *Client) DeleteUntilOffset(topic string, offset int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := c.client.DeleteUntilOffset(ctx, &smqv1.DeleteUntilOffsetRequest{Topic: topic, Offset: offset})
	return err
}

// BulkRetrieve retrieves a batch of messages from the server.
func (c *Client) BulkRetrieve(topic string, startOffset int64, limit int32) (*smqv1.BulkRetrieveResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.requestTimeout)
	defer cancel()

	response, err := c.client.BulkRetrieve(ctx, &smqv1.BulkRetrieveRequest{
		Topic:       topic,
		StartOffset: startOffset,
		Limit:       limit,
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}
