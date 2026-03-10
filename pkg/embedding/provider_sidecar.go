// Package embedding provides a unified interface for embedding providers.
// This file contains the gRPC sidecar provider implementation.
package embedding

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/tingly-dev/tingly-agentscope/pkg/embedding/pb"
)

const (
	defaultSidecarAddress   = "localhost:50051"
	defaultSidecarModel     = "TaylorAI/bge-micro-v2"
	defaultSidecarTimeout   = 30 * time.Second
	defaultSidecarDimension = 384
)

// SidecarConfig holds configuration for the sidecar provider.
type SidecarConfig struct {
	Address   string        // gRPC address (default: localhost:50051)
	ModelName string        // Model identifier
	ModelPath string        // Path for model initialization
	Timeout   time.Duration // Request timeout
	Dimension int           // Embedding dimension (0 = auto-detect)
}

// SidecarProvider implements Provider via gRPC sidecar.
type SidecarProvider struct {
	client    pb.LLMServiceClient
	conn      *grpc.ClientConn
	modelName string
	dimension int
	timeout   time.Duration
}

// NewSidecarProvider creates a new sidecar provider with the given configuration.
func NewSidecarProvider(ctx context.Context, cfg *SidecarConfig) (*SidecarProvider, error) {
	if cfg == nil {
		cfg = &SidecarConfig{}
	}

	address := cfg.Address
	if address == "" {
		address = defaultSidecarAddress
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultSidecarTimeout
	}

	// Connect to sidecar
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sidecar at %s: %w", address, err)
	}

	client := pb.NewLLMServiceClient(conn)

	modelName := cfg.ModelName
	if modelName == "" {
		modelName = defaultSidecarModel
	}

	dimension := cfg.Dimension
	if dimension == 0 {
		dimension = defaultSidecarDimension
	}

	// Check health
	healthCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	health, err := client.Health(healthCtx, &pb.HealthRequest{})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("sidecar health check failed: %w", err)
	}
	if !health.Healthy {
		conn.Close()
		return nil, ErrUnavailable
	}

	// Initialize model if path provided
	if cfg.ModelPath != "" {
		initCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		_, err := client.InitModel(initCtx, &pb.InitRequest{
			ModelPath:   cfg.ModelPath,
			ContextSize: 2048,
		})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to initialize model: %w", err)
		}
	}

	return &SidecarProvider{
		client:    client,
		conn:      conn,
		modelName: modelName,
		dimension: dimension,
		timeout:   timeout,
	}, nil
}

// Embed generates an embedding for a single text.
func (p *SidecarProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, ErrInvalidInput
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	resp, err := p.client.Embed(ctx, &pb.EmbedRequest{Text: text})
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}

	// Update dimension if server returns different value
	if resp.Dim > 0 && int(resp.Dim) != p.dimension {
		p.dimension = int(resp.Dim)
	}

	return resp.Vector, nil
}

// EmbedBatch generates embeddings for multiple texts.
func (p *SidecarProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := p.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		result[i] = emb
	}
	return result, nil
}

// Dimension returns the embedding dimension.
func (p *SidecarProvider) Dimension() int {
	return p.dimension
}

// ModelName returns the model name.
func (p *SidecarProvider) ModelName() string {
	return p.modelName
}

// Close closes the gRPC connection.
func (p *SidecarProvider) Close() error {
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
