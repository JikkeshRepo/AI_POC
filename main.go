package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	

type ModelConfig struct {
	Model   string
	TimeOut time.Duration
}

type SearchTool struct {
	name        string
	description string
	Func        func(string) (string, error)
}

func (t SearchTool) Call(ctx context.Context, input string) (string, error) {
	return t.Func(input)
}

func (t SearchTool) Name() string {
	return t.name
}

func (t SearchTool) Description() string {
	return t.description
}

func GenerateFromLLM(ctx context.Context, llm llms.LLM, prompt string, mc ModelConfig) (string, error) {
	var fullResponse strings.Builder
	isStreaming := false

	completion, err := llm.Call(ctx,
		prompt,
		llms.WithTemperature(0.5),
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			isStreaming = true
			text := string(chunk)
			fullResponse.WriteString(text)
			return nil
		}),
	)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	if !isStreaming {
		fullResponse.WriteString(completion)
	}

	return fullResponse.String(), nil
}

func performSearchQuery(query string) (string, error) {
	searchQueryURL := "https://duckduckgo.com/html/?q=" + url.QueryEscape(query)

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest("GET", searchQueryURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to perform search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search request failed with status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if strings.Contains(string(body), "CAPTCHA") {
		return "", fmt.Errorf("CAPTCHA detected, unable to proceed")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("failed to parse search results: %w", err)
	}

	var results []string

	doc.Find(".result__body").Each(func(i int, s *goquery.Selection) {
		if i < 5 {
			title := s.Find(".result__title").Text()
			link, _ := s.Find(".result__title a").Attr("href")
			snippet := s.Find(".result__snippet").Text()
			results = append(results, fmt.Sprintf("Title: %s\nLink: %s\nSnippet: %s\n", title, link, snippet))
		}
	})
	if len(results) == 0 {
		return "No results found.", nil
	}

	return strings.Join(results, "\n"), nil
}

func rateLimitedPerformSearch(query string) (string, error) {
	time.Sleep(2 * time.Second)
	return performSearchQuery(query)
}

func createSearchAgent() (tools.Tool, error) {
	searchTool := SearchTool{
		name:        "search",
		description: "Searches from DuckDuckGo",
		Func: func(query string) (string, error) {
			return performSearchQuery(query)
		},
	}

	return searchTool, nil
}

func main() {
	log.Println("Hello Rr")

	config := ModelConfig{
		Model:   "llama3.1",
		TimeOut: 2 * time.Minute,
	}

	reader := bufio.NewReader(os.Stdin)

	llm, err := ollama.New(ollama.WithModel(config.Model))
	if err != nil {
		log.Fatalf("Failed to initialize LLM: %v", err)
	}

	searchTool, err := createSearchAgent()
	if err != nil {
		log.Fatalf("Failed to create search tool: %v", err)
	}

	var history []string

	for {
		fmt.Print("Enter your question (or 'quit' to exit): ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		if strings.ToLower(userInput) == "quit" {
			break
		}

		if len(history) > 0 {
			userInput = strings.Join(history, "\n") + "\n" + userInput
		}

		ctx, cancel := context.WithTimeout(context.Background(), config.TimeOut)

		searchResult, err := searchTool.Call(ctx, userInput)
		if err != nil {
			if strings.Contains(err.Error(), "CAPTCHA") {
				fmt.Println("CAPTCHA detected. Please try again later or use a different IP.")
			} else {
				fmt.Printf("Search failed: %v\n", err)
			}
			cancel()
			continue
		}

		prompt := fmt.Sprintf(`You are an AI assistant that uses search results to answer questions accurately. 
		Base your answers on the provided search results and conversation history.
		If the search results don't contain relevant information, say so.
		
		Conversation history:
		%s
		
		Current question: %s
		
		Search results:
		%s
		
		Instructions:
		1. Analyze the search results and conversation history carefully.
		2. Provide a comprehensive answer based on the information in the search results and relevant context from the conversation history.
		3. If the search results don't contain relevant information to answer the question, state that clearly.
		4. Keep your answer concise and to the point.
		
		Your answer:`, strings.Join(history, "\n"), userInput, searchResult)

		fmt.Println("\nGenerating response...")

		response, err := GenerateFromLLM(ctx, llm, prompt, config)
		cancel()

		if err != nil {
			if err == context.DeadlineExceeded {
				fmt.Printf("Operation timed out after %v seconds\n", config.TimeOut.Seconds())
			} else {
				fmt.Printf("Error: %v\n", err)
			}
		} else {
			fmt.Println(response)

			history = append(history, fmt.Sprintf("User: %s", truncateString(userInput, 100)))
			history = append(history, fmt.Sprintf("AI: %s", truncateString(response, 200)))
			if len(history) > 10 {
				history = history[len(history)-10:]
			}
		}
	}
}

func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength-3] + "..."
}
