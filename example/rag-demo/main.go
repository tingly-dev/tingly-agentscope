package main

import (
	"context"
	"fmt"
	"log"

	"github.com/tingly-dev/tingly-agentscope/pkg/embedding"
	"github.com/tingly-dev/tingly-agentscope/pkg/rag"
	"github.com/tingly-dev/tingly-agentscope/pkg/rag/reader"
	"github.com/tingly-dev/tingly-agentscope/pkg/rag/store"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Tingly Scope RAG Demo ===\n")

	// 1. Create embedding model (using mock for demo)
	fmt.Println("1. Creating embedding model...")
	model := embedding.NewMockProvider(1536)
	fmt.Printf("   Model: %s (dimension: %d)\n\n", model.ModelName(), model.Dimension())

	// 2. Create vector store
	fmt.Println("2. Creating vector store...")
	store := store.NewMemoryStore()
	fmt.Println("   In-memory store created\n")

	// 3. Create knowledge base
	fmt.Println("3. Creating knowledge base...")
	kb := rag.NewSimpleKnowledge(model, store)
	fmt.Println("   Knowledge base initialized\n")

	// 4. Create document reader
	fmt.Println("4. Creating document reader...")
	textReader := reader.NewTextReader()
	// Use smaller chunks for demo
	textReader.SetChunkingStrategy(reader.NewFixedChunkingStrategy(200, 50, "\n\n"))
	fmt.Println("   Text reader with fixed chunking created\n")

	// 5. Add sample documents
	fmt.Println("5. Adding sample documents...")
	sampleDocuments := []string{
		"Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.\n\n" +
			"Go was developed at Google in 2007 to improve programming productivity in an era of multicore, networked machines and large codebases.\n\n" +
			"Go's key features include garbage collection, structural typing, and concurrency.",

		"The Go programming language was announced in November 2009. It became an open source project in 2010.\n\n" +
			"Go 1.0 was released in March 2012. Go is widely used in production at Google and in many other organizations and open-source projects.",

		"RAG (Retrieval-Augmented Generation) is a technique that enhances large language models with external knowledge retrieval.\n\n" +
			"RAG systems typically consist of: document ingestion, embedding generation, vector storage, and similarity search.\n\n" +
			"RAG helps reduce hallucinations and provides up-to-date information.",

		"Vector embeddings are numerical representations of text that capture semantic meaning.\n\n" +
			"Cosine similarity is commonly used to measure the similarity between embeddings.\n\n" +
			"Popular embedding models include OpenAI's text-embedding-3-small and text-embedding-3-large.",
	}

	for i, docText := range sampleDocuments {
		docs, err := textReader.Read(ctx, docText)
		if err != nil {
			log.Fatalf("Failed to read document %d: %v", i, err)
		}

		err = kb.AddDocuments(ctx, docs)
		if err != nil {
			log.Fatalf("Failed to add documents %d: %v", i, err)
		}

		fmt.Printf("   Added document %d (%d chunks)\n", i+1, len(docs))
	}

	size, _ := kb.Size(ctx)
	fmt.Printf("   Total chunks in knowledge base: %d\n\n", size)

	// 6. Perform similarity searches
	fmt.Println("6. Performing similarity searches...")

	queries := []string{
		"What is Go programming language?",
		"How does RAG work?",
		"Tell me about vector embeddings",
	}

	for _, query := range queries {
		fmt.Printf("\n   Query: \"%s\"\n", query)

		docs, err := kb.Retrieve(ctx, query, 2, nil)
		if err != nil {
			log.Fatalf("Failed to retrieve: %v", err)
		}

		if len(docs) == 0 {
			fmt.Println("   No results found")
			continue
		}

		for i, doc := range docs {
			score := ""
			if doc.Score != nil {
				score = fmt.Sprintf(" (score: %.4f)", *doc.Score)
			}
			content := doc.GetTextContent()
			if len(content) > 100 {
				content = content[:100] + "..."
			}
			fmt.Printf("   Result %d%s: %s\n", i+1, score, content)
		}
	}

	fmt.Println("\n=== Demo Complete ===")
}
