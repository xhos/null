package receiptparser

import (
	arian "ariand/internal/gen/arian/v1"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client interface {
	Parse(
		ctx context.Context,
		file io.Reader,
		filename string,
		contentType string,
		engine *arian.ReceiptEngine,
	) (*arian.Receipt, error)

	GetStatus(ctx context.Context) (*arian.GetStatusResponse, error)
	TestConnection(ctx context.Context) error
}

// grpcClient is the implementation of Client using gRPC
type grpcClient struct {
	client arian.ReceiptParsingServiceClient
	conn   *grpc.ClientConn
}

func New(address string, timeout time.Duration) (Client, error) {
	grpcAddress := address
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		u, err := url.Parse(address)
		if err != nil {
			return nil, fmt.Errorf("failed to parse receipt parser URL %s: %w", address, err)
		}
		grpcAddress = u.Host
	}

	conn, err := grpc.NewClient(grpcAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection to receipt parsing service at %s: %w", grpcAddress, err)
	}

	client := arian.NewReceiptParsingServiceClient(conn)

	return &grpcClient{
		client: client,
		conn:   conn,
	}, nil
}

func (c *grpcClient) TestConnection(ctx context.Context) error {
	_, err := c.GetStatus(ctx)
	return err
}

func (c *grpcClient) Close() error {
	return c.conn.Close()
}

// Parse sends an image to the gRPC service and returns the parsed receipt
func (c *grpcClient) Parse(
	ctx context.Context,
	file io.Reader,
	filename string,
	contentType string,
	engine *arian.ReceiptEngine,
) (*arian.Receipt, error) {
	// Read the file data
	imageData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Create the gRPC request
	req := &arian.ParseImageRequest{
		ImageData:   imageData,
		ContentType: contentType,
		Engine:      engine,
	}

	// Call the gRPC service
	resp, err := c.client.ParseImage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to parse image via gRPC: %w", err)
	}

	return resp.Receipt, nil
}

// GetStatus returns the status of available parsing providers
func (c *grpcClient) GetStatus(ctx context.Context) (*arian.GetStatusResponse, error) {
	req := &arian.GetStatusRequest{}

	resp, err := c.client.GetStatus(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get status via gRPC: %w", err)
	}

	return resp, nil
}
