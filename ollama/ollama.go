// Package ollama implements access to offline Ollama model.
//
// [Client] implements [llm.Embedder]. Use [NewClient] to connect.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/superryanguo/rmcp/llm"
	. "github.com/superryanguo/rmcp/openai"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
)

const (
	DefaultEmbeddingModel = "mxbai-embed-large"
	EmbedUrl              = "/api/embed"
	DefaultGenModel       = "llama3.2:3b"
	DefaultGenModel3      = "deepseek-v2:16b"
	GenUrl                = "/api/generate"
	DefaultGenModel2      = "deepseek-r1:7b"
	maxBatch              = 512 // default physical batch size in ollama
)

// A Client represents a connection to Ollama.
type Client struct {
	slog  *slog.Logger
	hc    *http.Client
	url   *url.URL // url of the ollama server
	model string
}

type Response struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

func AssembleRsp(d []byte) (string, error) {
	var s string
	var err error

	lines := strings.Split(string(d), "\n")
	for _, line := range lines {
		var resp Response
		err = json.Unmarshal([]byte(line), &resp)
		if err != nil {
			return s, err
		}
		s += resp.Response
	}
	return s, nil
}

// NewClient returns a connection to Ollama server. If empty, the
// server is assumed to be hosted at http://127.0.0.1:11434.
// The model is the model name to use for embedding.
// A typical model for embedding is "mxbai-embed-large".
func NewClient(lg *slog.Logger, hc *http.Client, server string, model string) (*Client, error) {
	if server == "" {
		host := os.Getenv("OLLAMA_HOST")
		if host == "" {
			host = "127.0.0.1"
		}
		server = "http://" + host + ":11434"
	}
	u, err := url.Parse(server)
	if err != nil {
		return nil, err
	}
	return &Client{slog: lg, hc: hc, url: u, model: model}, nil
}

// EmbedDocs returns the vector embeddings for the docs,
// implementing [llm.Embedder].
func (c *Client) EmbedDocs(ctx context.Context, docs []llm.EmbedDoc) ([]llm.Vector, error) {
	embedURL := c.url.JoinPath(EmbedUrl) // ollama embed endpoint
	var vecs []llm.Vector
	for docs := range slices.Chunk(docs, maxBatch) {
		var inputs []string
		for _, doc := range docs {
			// ollama does not support adding content with title
			input := doc.Title + "\n\n" + doc.Text
			inputs = append(inputs, input)
		}
		vs, err := embed(ctx, c.hc, embedURL, inputs, c.model)
		if err != nil {
			return nil, err
		}
		vecs = append(vecs, vs...)
	}
	return vecs, nil
}

func (c *Client) Prompt(ctx context.Context, input string) ([]byte, error) {
	u := c.url.JoinPath(GenUrl)
	rsp, err := prompt(ctx, c.hc, u, input, c.model)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}

func prompt(ctx context.Context, hc *http.Client, u *url.URL, inputs string, model string) ([]byte, error) {
	/////////
	tools := []ToolDefinition{
		ReadFileDefinition,
		ListFilesDefinition,
	}

	openaiTools := []OpenAIChatCompletionTool{}
	for _, toolDef := range tools {
		openaiTools = append(openaiTools, OpenAIChatCompletionTool{
			Type: "function",
			Function: OpenAIChatCompletionFunctionDefinition{
				Name:        toolDef.Name,
				Description: toolDef.Description,
				Parameters:  toolDef.InputSchema,
			},
		})
	}

	systemPrompt := "You are a helpful Go programmer assistant. You have access to tools to interact with the local filesystem (read, list, edit files). Use them when appropriate to fulfill the user's request. When editing, be precise about the changes. Respond ONLY with tool calls if you need to use tools, otherwise respond with text."
	conversation := []OpenAIChatCompletionMessage{
		{Role: "system", Content: systemPrompt}, // Start with system prompt
	}

	// Build request payload
	requestPayload := OpenAIChatCompletionRequest{
		Model:       model,
		Messages:    conversation,
		Tools:       openaiTools,
		ToolChoice:  "auto", // Let the model decide when to use tools
		MaxTokens:   2048,   // Or make configurable
		Temperature: 0.7,    // Reasonable default
	}

	// Marshal payload to JSON
	jsonPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	/////////
	//request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(erj))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(jsonPayload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := hc.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	rsp, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Gai get rsp1: %s\n", string(rsp))

	ask := struct {
		Model  string `json:"model"`
		Input  string `json:"prompt"`
		Stream bool   `json:"stream"`
	}{
		Model:  model,
		Input:  "what do you see in the directory?",
		Stream: true,
	}

	erj, err := json.Marshal(ask)
	if err != nil {
		return nil, err
	}
	request, err = http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(erj))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err = hc.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	rsp, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Gai get rsp2: %s\n", string(rsp))
	return rsp, nil
}

func embed(ctx context.Context, hc *http.Client, embedURL *url.URL, inputs []string, model string) ([]llm.Vector, error) {
	embReq := struct {
		Model string   `json:"model"`
		Input []string `json:"input"`
	}{
		Model: model,
		Input: inputs,
	}
	erj, err := json.Marshal(embReq)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, embedURL.String(), bytes.NewReader(erj))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := hc.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	embResp, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if err := embedError(response, embResp); err != nil {
		return nil, err
	}
	return embeddings(embResp)
}

// embedError extracts error from ollama's response, if any.
func embedError(resp *http.Response, embResp []byte) error {
	if resp.StatusCode == 200 {
		return nil
	}
	if resp.StatusCode == 400 {
		var e struct {
			Error string `json:"error"`
		}
		// ollama returns JSON with error field set for bad requests.
		if err := json.Unmarshal(embResp, &e); err != nil {
			return err
		}
		return fmt.Errorf("ollama response error: %s", e.Error)
	}
	return fmt.Errorf("ollama response error: %s", resp.Status)
}

func embeddings(embResp []byte) ([]llm.Vector, error) {
	// In case there are no errors, ollama returns
	// a JSON with Embeddings field set.
	var e struct {
		Embeddings []llm.Vector `json:"embeddings"`
	}
	if err := json.Unmarshal(embResp, &e); err != nil {
		return nil, err
	}
	return e.Embeddings, nil
}
