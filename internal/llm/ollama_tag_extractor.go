package llm

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type LLMService struct {
	OllamaURL string
	logger    *slog.Logger
}

func NewLLMService(OllamaURL string, logger *slog.Logger) *LLMService {
	return &LLMService{
		OllamaURL: OllamaURL,
		logger:    logger,
	}
}

type TagResponse struct {
	Tags []string `json:"tags"`
}

type OllamaResponse struct {
	Response string `json:"response"`
}

func (s *LLMService) ExtractTags(text string, allowedTags []string) ([]string, error) {

	escapedText := strings.ReplaceAll(text, `"`, `\"`)

	prompt := fmt.Sprintf(
		`### Role
		You are a precise Text-to-Tag Classifier.
		
        ### Constraints
        1. Choose tags ONLY from this allowed list: [%s]
        2. If no tags are relevant, return {\"tags\": []}.
        3. Map synonyms or related concepts (e.g., \"mojito\" -> \"cocktails\", \"stars\" -> \"astronomy\").
        4. Output MUST be valid JSON. No conversation, no explanations.
		
        ### Examples
        Input: \"A nice place for a drink with a view\"
        Output: {\"tags\": [\"cocktails\", \"popular\"]}
		
        Input: \"somewhere to see the planets\"
        Output: {\"tags\": [\"astronomy\", \"universe\"]}
		
        Input: \"Looking for kittens\"
        Output: {\"tags\": [\"cats\", \"animals\"]}

		### Task
        User Input: \"%s\"
        Output:`,
		strings.Join(allowedTags, ", "),
		escapedText,
	)

	s.logger.Debug("Generated Ollama prompt", "prompt", prompt)

	payload := map[string]interface{}{
		"model":  "llama3.2:1b",
		"prompt": prompt,
		"stream": false,
		"format": "json",
		"options": map[string]interface{}{
			"temperature": 0,
		},
	}
	agent := fiber.Post(s.OllamaURL + "/api/generate").
		JSON(payload).
		Timeout(30 * time.Second)

	var outer OllamaResponse

	statusCode, _, errs := agent.Struct(&outer)

	if len(errs) > 0 {
		return nil, errs[0]
	}

	if statusCode != fiber.StatusOK {
		return nil, fmt.Errorf("ollama status: %d", statusCode)
	}

	var inner TagResponse

	cleanJSON := strings.TrimSpace(outer.Response)

	if err := json.Unmarshal([]byte(cleanJSON), &inner); err != nil {
		return nil, fmt.Errorf("LLM JSON error: %w", err)
	}
	return inner.Tags, nil
}

func (s *LLMService) ExtractMultiTags(segments, allowedTags []string) ([][]string, error) {
	results := make([][]string, len(segments))
	errs := make([]error, len(segments))

	var wg sync.WaitGroup
	maxConcurrentRequests := 10
	sem := make(chan struct{}, maxConcurrentRequests)

	for i, seg := range segments {
		wg.Add(1)
		sem <- struct{}{}

		go func(index int, text string) {
			defer wg.Done()
			defer func() { <-sem }()

			tags, err := s.ExtractTags(text, allowedTags)
			if err != nil {
				errs[index] = err
				return
			}
			results[index] = tags
		}(i, seg)
	}

	var errMsgs []string
	for i, err := range errs {
		if err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("segment %d: %v", i, err))
		}
	}

	if len(errMsgs) > 0 {
		return results, fmt.Errorf("extraction errors: %s", strings.Join(errMsgs, "; "))
	}

	wg.Wait()
	return results, nil
}

func (s *LLMService) Warmup() {
	s.logger.Info("Starting LLM warmup request...")
	start := time.Now()

	_, err := s.ExtractTags("System startup warmup sequence", []string{"warmup"})

	if err != nil {
		s.logger.Warn("LLM warmup request completed with error (fetching might have timed out, but model should be loading)",
			"error", err,
			"duration", time.Since(start))
		return
	}

	s.logger.Info("LLM warmup completed successfully", "duration", time.Since(start))
}
