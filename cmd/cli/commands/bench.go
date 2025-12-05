package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/model-runner/cmd/cli/commands/completion"
	"github.com/docker/model-runner/cmd/cli/desktop"
	"github.com/docker/model-runner/pkg/inference"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const defaultBenchPrompt = "Write a short story about a robot learning to paint."

// benchResult holds the result of a single benchmark request
type benchResult struct {
	Duration         time.Duration
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	Error            error
}

// benchStats holds aggregated statistics for benchmark results
type benchStats struct {
	Concurrency      int
	TotalRequests    int
	SuccessfulReqs   int
	FailedReqs       int
	TotalDuration    time.Duration
	MeanDuration     time.Duration
	MinDuration      time.Duration
	MaxDuration      time.Duration
	StdDevDuration   time.Duration
	TotalTokens      int
	CompletionTokens int
	TokensPerSecond  float64
}

func newBenchCmd() *cobra.Command {
	var prompt string
	var concurrencies []int
	var numRequests int

	const cmdArgs = "MODEL"
	c := &cobra.Command{
		Use:   "bench " + cmdArgs,
		Short: "Benchmark a model's performance with concurrent requests",
		Long: `Benchmark a model's performance by measuring tokens per second with varying levels of concurrency.

This command provides a hyperfine-like experience for benchmarking LLM inference performance.
It runs the specified model with 1, 2, 4, and 8 concurrent requests by default and reports
timing statistics including tokens per second.`,
		Example: `  # Benchmark with default prompt and concurrency levels
  docker model bench llama3.2

  # Benchmark with custom prompt
  docker model bench llama3.2 --prompt "Explain quantum computing"

  # Benchmark with specific concurrency levels
  docker model bench llama3.2 --concurrency 1,4,8

  # Run more requests per concurrency level
  docker model bench llama3.2 --requests 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			model := args[0]

			if _, err := ensureStandaloneRunnerAvailable(cmd.Context(), asPrinter(cmd), false); err != nil {
				return fmt.Errorf("unable to initialize standalone model runner: %w", err)
			}

			// Check if model exists locally
			_, err := desktopClient.Inspect(model, false)
			if err != nil {
				if !errors.Is(err, desktop.ErrNotFound) {
					return handleClientError(err, "Failed to inspect model")
				}
				cmd.Println("Unable to find model '" + model + "' locally. Pulling from the server.")
				if err := pullModel(cmd, desktopClient, model, false); err != nil {
					return err
				}
			}

			return runBenchmark(cmd, model, prompt, concurrencies, numRequests)
		},
		ValidArgsFunction: completion.ModelNames(getDesktopClient, 1),
	}
	c.Args = requireExactArgs(1, "bench", cmdArgs)

	c.Flags().StringVar(&prompt, "prompt", defaultBenchPrompt, "Prompt to use for benchmarking")
	c.Flags().IntSliceVarP(&concurrencies, "concurrency", "c", []int{1, 2, 4, 8}, "Concurrency levels to test")
	c.Flags().IntVarP(&numRequests, "requests", "n", 3, "Number of requests per concurrency level")

	return c
}

func runBenchmark(cmd *cobra.Command, model, prompt string, concurrencies []int, numRequests int) error {
	boldCyan := color.New(color.FgCyan, color.Bold)
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	boldCyan.Fprintf(cmd.OutOrStdout(), "Benchmark: %s\n", model)
	cmd.Printf("  Prompt: %s\n", truncateString(prompt, 50))
	cmd.Printf("  Requests per concurrency level: %d\n\n", numRequests)

	// Warm-up run
	cmd.Print("Warming up...")
	_, err := runSingleBenchmark(cmd.Context(), model, prompt)
	if err != nil {
		cmd.Println(" failed!")
		return fmt.Errorf("warm-up failed: %w", err)
	}
	cmd.Println(" done")

	var allStats []benchStats

	for _, concurrency := range concurrencies {
		bold.Fprintf(cmd.OutOrStdout(), "Running with %d concurrent request(s)...\n", concurrency)

		stats, err := runConcurrentBenchmarks(cmd.Context(), model, prompt, concurrency, numRequests)
		if err != nil {
			return fmt.Errorf("benchmark failed at concurrency %d: %w", concurrency, err)
		}

		allStats = append(allStats, stats)

		// Print progress
		if stats.FailedReqs > 0 {
			yellow.Fprintf(cmd.OutOrStdout(), "  Completed: %d/%d requests (%.1f%% success rate)\n",
				stats.SuccessfulReqs, stats.TotalRequests,
				float64(stats.SuccessfulReqs)/float64(stats.TotalRequests)*100)
		} else {
			green.Fprintf(cmd.OutOrStdout(), "  Completed: %d/%d requests\n",
				stats.SuccessfulReqs, stats.TotalRequests)
		}
		cmd.Printf("  Mean: %s ± %s\n", formatDuration(stats.MeanDuration), formatDuration(stats.StdDevDuration))
		cmd.Printf("  Range: [%s ... %s]\n", formatDuration(stats.MinDuration), formatDuration(stats.MaxDuration))
		cmd.Printf("  Tokens/sec: %.2f\n\n", stats.TokensPerSecond)
	}

	// Print summary table
	printBenchmarkSummary(cmd, allStats)

	return nil
}

func runConcurrentBenchmarks(ctx context.Context, model, prompt string, concurrency, numRequests int) (benchStats, error) {
	results := make([]benchResult, numRequests)
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	startTime := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := runSingleBenchmark(ctx, model, prompt)
			if err != nil {
				results[idx] = benchResult{Error: err}
				return
			}
			results[idx] = result
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	return calculateStats(results, concurrency, totalDuration), nil
}

func runSingleBenchmark(ctx context.Context, model, prompt string) (benchResult, error) {
	start := time.Now()

	reqBody := desktop.OpenAIChatRequest{
		Model: model,
		Messages: []desktop.OpenAIChatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return benchResult{}, fmt.Errorf("error marshaling request: %w", err)
	}

	completionsPath := inference.InferencePrefix + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, modelRunner.URL(completionsPath), bytes.NewReader(jsonData))
	if err != nil {
		return benchResult{}, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "docker-model-cli/"+desktop.Version)

	resp, err := modelRunner.Client().Do(req)
	if err != nil {
		return benchResult{}, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return benchResult{}, fmt.Errorf("error response: status=%d body=%s", resp.StatusCode, body)
	}

	// Read and parse the streaming response
	var finalUsage struct {
		CompletionTokens int `json:"completion_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		TotalTokens      int `json:"total_tokens"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return benchResult{}, fmt.Errorf("error reading response: %w", err)
	}

	// Parse SSE events to get the usage
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var streamResp desktop.OpenAIChatResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			continue
		}

		if streamResp.Usage != nil {
			finalUsage.CompletionTokens = streamResp.Usage.CompletionTokens
			finalUsage.PromptTokens = streamResp.Usage.PromptTokens
			finalUsage.TotalTokens = streamResp.Usage.TotalTokens
		}
	}

	duration := time.Since(start)

	return benchResult{
		Duration:         duration,
		PromptTokens:     finalUsage.PromptTokens,
		CompletionTokens: finalUsage.CompletionTokens,
		TotalTokens:      finalUsage.TotalTokens,
	}, nil
}

func calculateStats(results []benchResult, concurrency int, totalDuration time.Duration) benchStats {
	stats := benchStats{
		Concurrency:   concurrency,
		TotalRequests: len(results),
		MinDuration:   time.Duration(math.MaxInt64),
		TotalDuration: totalDuration,
	}

	var durations []time.Duration

	for _, r := range results {
		if r.Error != nil {
			stats.FailedReqs++
			continue
		}

		stats.SuccessfulReqs++
		stats.TotalTokens += r.TotalTokens
		stats.CompletionTokens += r.CompletionTokens
		durations = append(durations, r.Duration)

		if r.Duration < stats.MinDuration {
			stats.MinDuration = r.Duration
		}
		if r.Duration > stats.MaxDuration {
			stats.MaxDuration = r.Duration
		}
	}

	if len(durations) == 0 {
		stats.MinDuration = 0
		return stats
	}

	// Calculate mean
	var totalDur time.Duration
	for _, d := range durations {
		totalDur += d
	}
	stats.MeanDuration = totalDur / time.Duration(len(durations))

	// Calculate standard deviation
	var sumSquares float64
	meanFloat := float64(stats.MeanDuration)
	for _, d := range durations {
		diff := float64(d) - meanFloat
		sumSquares += diff * diff
	}
	variance := sumSquares / float64(len(durations))
	stats.StdDevDuration = time.Duration(math.Sqrt(variance))

	// Calculate tokens per second (based on completion tokens generated during the total wall-clock time)
	if stats.TotalDuration > 0 {
		stats.TokensPerSecond = float64(stats.CompletionTokens) / stats.TotalDuration.Seconds()
	}

	return stats
}

func printBenchmarkSummary(cmd *cobra.Command, allStats []benchStats) {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)

	bold.Fprintln(cmd.OutOrStdout(), "Summary")
	cmd.Println(strings.Repeat("─", 70))
	cmd.Printf("%-12s  %-15s  %-15s  %-15s\n", "Concurrency", "Mean Time", "Tokens/sec", "Success Rate")
	cmd.Println(strings.Repeat("─", 70))

	// Find the best tokens per second
	var bestTPS float64
	for _, s := range allStats {
		if s.TokensPerSecond > bestTPS {
			bestTPS = s.TokensPerSecond
		}
	}

	for _, s := range allStats {
		successRate := float64(s.SuccessfulReqs) / float64(s.TotalRequests) * 100
		meanStr := fmt.Sprintf("%s ± %s", formatDuration(s.MeanDuration), formatDuration(s.StdDevDuration))
		tpsStr := fmt.Sprintf("%.2f", s.TokensPerSecond)
		successStr := fmt.Sprintf("%.0f%%", successRate)

		if s.TokensPerSecond == bestTPS {
			cmd.Printf("%-12d  %-15s  ", s.Concurrency, meanStr)
			green.Fprintf(cmd.OutOrStdout(), "%-15s", tpsStr)
			cmd.Printf("  %-15s\n", successStr)
		} else {
			cmd.Printf("%-12d  %-15s  %-15s  %-15s\n", s.Concurrency, meanStr, tpsStr, successStr)
		}
	}

	cmd.Println(strings.Repeat("─", 70))

	// Find optimal concurrency
	sort.Slice(allStats, func(i, j int) bool {
		return allStats[i].TokensPerSecond > allStats[j].TokensPerSecond
	})

	if len(allStats) > 0 {
		best := allStats[0]
		green.Fprintf(cmd.OutOrStdout(), "\nOptimal concurrency: %d (%.2f tokens/sec)\n", best.Concurrency, best.TokensPerSecond)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Nanoseconds())/1e3)
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
