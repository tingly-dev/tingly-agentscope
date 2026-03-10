package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/tingly-dev/tingly-agentscope/pkg/embedding/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	sidecarAddress = "localhost:50051"
)

// Category represents a work item category
type Category string

const (
	CategoryBugfix   Category = "bugfix"
	CategoryFeat     Category = "feat"
	CategorySpec     Category = "spec"
	CategoryRefactor Category = "refactor"
	CategoryReview   Category = "review"
	CategoryTest     Category = "test"
)

// CategoryPrompt stores the prompt and embedding for a category
type CategoryPrompt struct {
	Category  Category
	Prompt    string
	Embedding []float32
}

// Classifier classifies text into categories using embeddings
type Classifier struct {
	client         pb.LLMServiceClient
	categories     []CategoryPrompt
	embeddingCache map[string][]float32
}

// NewClassifier creates a new classifier
func NewClassifier(client pb.LLMServiceClient) *Classifier {
	return &Classifier{
		client: client,
		categories: []CategoryPrompt{
			{CategoryBugfix, "Task: fixing a bug, issue, defect, or error in the code. It involves repairing, patching, or resolving existing problems.", nil},
			{CategoryFeat, "Task: adding new features, functionality, or capabilities. It involves implementing something that didn't exist before.", nil},
			{CategorySpec, "Task: creating specifications, designs, or plans. It involves proposing architecture, defining requirements, or documenting technical decisions.", nil},
			{CategoryRefactor, "Task: refactoring or restructuring code. It involves improving code quality, maintainability, or organization without changing external behavior.", nil},
			{CategoryReview, "Task: code reviewing, auditing, or examining code.", nil},
			{CategoryTest, "Task: writing or adding tests for the code. It involves testing, verifying behavior, or ensuring correctness.", nil},
		},
		embeddingCache: make(map[string][]float32),
	}
}

// cosineSimilarity computes cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// getEmbedding retrieves or computes embedding for text
func (c *Classifier) getEmbedding(ctx context.Context, text string) ([]float32, error) {
	// Check cache
	if emb, ok := c.embeddingCache[text]; ok {
		return emb, nil
	}

	// Get from server
	resp, err := c.client.Embed(ctx, &pb.EmbedRequest{Text: text})
	if err != nil {
		return nil, err
	}

	// Cache and return
	c.embeddingCache[text] = resp.Vector
	return resp.Vector, nil
}

// Initialize initializes the classifier by computing category embeddings
func (c *Classifier) Initialize(ctx context.Context) error {
	fmt.Println("Initializing classifier with category embeddings...")
	fmt.Println()

	for i := range c.categories {
		emb, err := c.getEmbedding(ctx, c.categories[i].Prompt)
		if err != nil {
			return fmt.Errorf("failed to compute embedding for %s: %w", c.categories[i].Category, err)
		}
		c.categories[i].Embedding = emb
		fmt.Printf("  ✓ %s: %d dimensions\n", c.categories[i].Category, len(emb))
		fmt.Printf("    Prompt: \"%s\"\n", c.categories[i].Prompt)
	}

	fmt.Println("Classifier initialized!")
	fmt.Println()
	return nil
}

// Classify classifies the given text into a category
func (c *Classifier) Classify(ctx context.Context, text string) (Category, float32, error) {
	// Use the same prompt format for input text
	inputPrompt := "Task:: " + text
	textEmb, err := c.getEmbedding(ctx, inputPrompt)
	if err != nil {
		return "", 0, err
	}

	// Find best matching category
	var bestCategory Category
	var bestScore float32

	for _, cat := range c.categories {
		similarity := cosineSimilarity(textEmb, cat.Embedding)
		if similarity > bestScore {
			bestScore = similarity
			bestCategory = cat.Category
		}
	}

	return bestCategory, bestScore, nil
}

// ShowClassificationDemo demonstrates classification on various examples
func (c *Classifier) ShowClassificationDemo(ctx context.Context) {
	fmt.Println("=== Classification Demo ===")
	fmt.Println()

	// Show category references
	fmt.Println("Category References:")
	for _, cat := range c.categories {
		fmt.Printf("  %s: \"%s\"\n", cat.Category, cat.Prompt)
	}
	fmt.Println()

	testCases := []struct {
		text     string
		expected Category
	}{
		// Short examples
		{"Fix memory leak in connection pool", CategoryBugfix},
		{"Add user authentication feature", CategoryFeat},
		{"Design the database schema for user management", CategorySpec},
		{"Refactor the authentication module for better maintainability", CategoryRefactor},
		{"Review pull request #123 for bug fixes", CategoryReview},

		// Longer examples
		{"The application crashes intermittently when processing large CSV files due to a buffer overflow that needs to be patched", CategoryBugfix},
		{"Implement a comprehensive dark mode support for the UI with automatic theme switching based on system preferences and user settings", CategoryFeat},
		{"Write a detailed technical specification for the payment gateway integration including API contracts, error handling strategies, security requirements, and retry logic", CategorySpec},
		{"Clean up unused imports across all modules, reorganize the project structure to follow domain-driven design principles, and improve overall code quality by removing technical debt", CategoryRefactor},
		{"Conduct a thorough code review of the authentication subsystem, verify that all security best practices are followed, check for potential edge cases in error handling, and ensure proper test coverage", CategoryReview},

		// Complex scenarios
		{"Patch the security vulnerability in the login system where users can bypass authentication via SQL injection, and add input validation to prevent similar attacks", CategoryBugfix},
		{"Create interactive dashboard widgets for real-time analytics visualization with customizable layouts, export functionality, and support for multiple data sources including REST APIs and WebSocket streams", CategoryFeat},
		{"Define the complete API contract for mobile app synchronization including versioning strategy, backward compatibility guarantees, conflict resolution mechanisms, offline support, and data consistency models", CategorySpec},
		{"Refactor the monolithic service into microservices architecture, implement proper separation of concerns, improve performance through better caching strategies, and enhance maintainability with clearer module boundaries", CategoryRefactor},
		{"Review the implementation of the new feature branch focusing on code quality, adherence to coding standards, proper error handling, documentation completeness, and potential performance bottlenecks before merging to main", CategoryReview},

		// Edge cases / ambiguous
		{"The connection timeout issue needs investigation and resolution", CategoryBugfix},
		{"Add support for GraphQL queries alongside existing REST endpoints", CategoryFeat},
		{"Document the current system architecture and identify areas for improvement", CategorySpec},
		{"Simplify the complex conditional logic in the order processing module", CategoryRefactor},
		{"Verify that all test cases pass and coverage metrics are met", CategoryReview},

		// Chinese examples
		{"修复连接池中的内存泄漏问题", CategoryBugfix},
		{"添加用户认证功能", CategoryFeat},
		{"设计用户管理数据库架构", CategorySpec},
		{"重构认证模块以提高可维护性", CategoryRefactor},
		{"审查关于错误修复的拉取请求", CategoryReview},

		// Chinese longer examples
		{"应用程序在处理大型CSV文件时间歇性崩溃，是由于缓冲区溢出导致的，需要打补丁修复", CategoryBugfix},
		{"实现全面的暗色模式支持，包括基于系统偏好和用户设置的自动主题切换", CategoryFeat},
		{"编写支付网关集成的详细技术规范，包括API契约、错误处理策略、安全要求和重试逻辑", CategorySpec},
		{"清理所有模块中未使用的导入，按领域驱动设计原则重组项目结构，通过消除技术债务提高代码质量", CategoryRefactor},
		{"对认证子系统进行彻底的代码审查，验证是否遵循所有安全最佳实践，检查错误处理中的潜在边缘情况", CategoryReview},
	}

	var correct int
	for i, tc := range testCases {
		category, score, err := c.Classify(ctx, tc.text)
		if err != nil {
			log.Printf("Error classifying '%s': %v\n", tc.text, err)
			continue
		}

		status := "✓"
		if category != tc.expected {
			status = "✗"
		} else {
			correct++
		}

		inputPrompt := "Task:: " + tc.text
		fmt.Printf("%2d. [%s] %s\n", i+1, status, tc.text)
		fmt.Printf("    Input: \"%s\"\n", inputPrompt)
		fmt.Printf("    Predicted: %s (confidence: %.4f)\n", category, score)
		if category != tc.expected {
			// Compute score for expected category
			textEmb, err := c.getEmbedding(ctx, inputPrompt)
			if err == nil {
				var expectedCat *CategoryPrompt
				for j := range c.categories {
					if c.categories[j].Category == tc.expected {
						expectedCat = &c.categories[j]
						break
					}
				}
				if expectedCat != nil {
					expectedScore := cosineSimilarity(textEmb, expectedCat.Embedding)
					fmt.Printf("    Expected:  %s (confidence: %.4f)\n", tc.expected, expectedScore)
				}
			}
		}
		fmt.Println()
	}

	accuracy := float64(correct) / float64(len(testCases)) * 100
	fmt.Printf("=== Accuracy: %d/%d (%.1f%%) ===\n", correct, len(testCases), accuracy)
}

func main() {
	// Parse flags
	modelPath := flag.String("model", "", "Model path (default: from EMBEDDING_MODEL env or TaylorAI/bge-micro-v2)")
	contextSize := flag.Int("context", 2048, "Context size")
	seed := flag.Int("seed", 42, "Random seed")
	flag.Parse()

	// Determine model path
	if *modelPath == "" {
		if envPath := os.Getenv("EMBEDDING_MODEL"); envPath != "" {
			*modelPath = envPath
		} else {
			*modelPath = "TaylorAI/bge-micro-v2"
		}
	}

	// Connect to Rust sidecar
	conn, err := grpc.NewClient(sidecarAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to sidecar: %v", err)
	}
	defer conn.Close()

	client := pb.NewLLMServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Check health
	health, err := client.Health(ctx, &pb.HealthRequest{})
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	fmt.Printf("Sidecar Health: %v\n", health.Healthy)

	// Initialize model
	initResp, err := client.InitModel(ctx, &pb.InitRequest{
		ModelPath:   *modelPath,
		ContextSize: int32(*contextSize),
		Seed:        int32(*seed),
	})
	if err != nil {
		log.Fatalf("Failed to initialize model: %v", err)
	}
	fmt.Printf("Model Init: %s\n", initResp.Message)

	// Get model info
	info, err := client.ModelInfo(ctx, &pb.ModelInfoRequest{})
	if err != nil {
		log.Fatalf("Failed to get model info: %v", err)
	}
	fmt.Printf("Model Info:\n  Name: %s\n  Vocab: %d\n  Context: %d\n  Backend: %s\n\n",
		info.ModelName, info.VocabSize, info.ContextSize, info.Backend)

	// Create and initialize classifier
	classifier := NewClassifier(client)
	if err := classifier.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize classifier: %v", err)
	}

	// Run classification demo
	classifier.ShowClassificationDemo(ctx)
}
