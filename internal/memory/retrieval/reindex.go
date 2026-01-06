package retrieval

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/austiecodes/gomor/internal/client"
	"github.com/austiecodes/gomor/internal/memory/store"
	"github.com/austiecodes/gomor/internal/types"
)

// ReindexMemories re-calculates embeddings for all memories using the new model.
func ReindexMemories(ctx context.Context, s *store.Store, embeddingClient client.EmbeddingClient, model types.Model) error {
	// 1. Fetch all memories
	memories, err := s.GetAllMemories()
	if err != nil {
		return fmt.Errorf("failed to fetch memories for reindexing: %w", err)
	}

	total := len(memories)
	if total == 0 {
		return nil
	}

	log.Printf("Reindexing %d memories...", total)

	type reindexJob struct {
		item       store.MemoryItem
		retryCount int
		embedding  []float32
		err        error
	}

	// Channels
	// We use buffered channels to allow some pipeline overlap
	jobsCh := make(chan reindexJob, total)
	writeCh := make(chan reindexJob)
	retryCh := make(chan reindexJob)

	var wg sync.WaitGroup
	var failures []string
	var mu sync.Mutex

	// Create a cancellable context to allow us to stop workers when done
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 1. Embedder Goroutine: Initiates requests and receives responses
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-jobsCh:
				// Call embedding client
				emb, err := embeddingClient.Embed(ctx, model, job.item.Text)
				job.embedding = emb
				job.err = err

				// Send to writer (or retry handler via writer check)
				select {
				case <-ctx.Done():
					return
				case writeCh <- job:
				}
			}
		}
	}()

	// 2. Writer Goroutine: Responsible for writing to DB
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-writeCh:
				if job.err != nil {
					// Embedding failed, send to retry
					select {
					case <-ctx.Done():
						return
					case retryCh <- job:
					}
					continue
				}

				// Try to write to DB
				dim := embeddingClient.Dimensions(model)
				err := s.UpdateMemoryEmbedding(job.item.ID, job.embedding, model.ModelID, dim, model.Provider)
				if err != nil {
					job.err = fmt.Errorf("write failed: %w", err)
					select {
					case <-ctx.Done():
						return
					case retryCh <- job:
					}
					continue
				}

				// Success
				wg.Done()
			}
		}
	}()

	// 3. Retry Goroutine: Responsible for retrying
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case job := <-retryCh:
				if job.retryCount >= 5 {
					// Max retries reached, mark as failed
					mu.Lock()
					errMsg := fmt.Sprintf("- ID %s: %v", job.item.ID, job.err)
					failures = append(failures, errMsg)
					log.Printf("Failed to reindex memory %s after %d retries: %v", job.item.ID, job.retryCount, job.err)
					mu.Unlock()
					wg.Done()
					continue
				}

				// Backoff and retry
				// Google free tier rate limits can be strict (e.g. per minute quotas and delay requests).
				// We increase backoff significantly: 2s, 4s, 6s...
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(job.retryCount+1) * 2 * time.Second):
				}

				job.retryCount++
				job.err = nil

				select {
				case <-ctx.Done():
					return
				case jobsCh <- job:
				}
			}
		}
	}()

	// Initial load
	wg.Add(total)
	for _, m := range memories {
		select {
		case jobsCh <- reindexJob{item: m, retryCount: 0}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Wait for all items to be processed (either success or max retries)
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-doneCh:
		// Done successfully (or with some failures counted)
	}

	if len(failures) > 0 {
		return fmt.Errorf("%d memories failed to reindex:\n%s\nPlease try reindexing again later.", len(failures), strings.Join(failures, "\n"))
	}

	return nil
}
