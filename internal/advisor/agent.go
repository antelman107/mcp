package advisor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/gemini"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"

	"github.com/antelman107/mcp/internal/antalyakart"
)

const (
	appName            = "antalyakart-advisor"
	maxGenerateRounds  = 6
	maxPromptMessages  = 40
	defaultGeminiModel = "gemini-flash-latest"
)

const agentInstruction = `You are the AntalyaKart advisor, a Telegram assistant for public buses in Antalya.
Reply in the same language the user writes.
Use the transit tools for routes, stops, arrivals, and trip plans. Do not invent stop ids, ETAs, or route numbers.
If a tool fails, say what failed and ask for a clearer stop or route.
Keep answers short enough for a chat message.`

// Agent runs the Google ADK webhook agent: LlmAgent plus Runner, capped at
// six GenerateContent rounds, matching the volleyball bot's chat path.
type Agent struct {
	runner   *runner.Runner
	sessions session.Service
	rounds   *roundCap
}

func NewAgent(ctx context.Context, apiKey, modelName string, transit *antalyakart.Client) (*Agent, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY is required")
	}
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		modelName = defaultGeminiModel
	}

	llm, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		return nil, err
	}
	tools, err := transitTools(transit)
	if err != nil {
		return nil, err
	}
	rounds := newRoundCap(maxGenerateRounds)
	adkAgent, err := llmagent.New(llmagent.Config{
		Name:                 "antalyakart_advisor",
		Model:                llm,
		Description:          "Answers questions about Antalya public buses.",
		Instruction:          agentInstruction,
		Tools:                tools,
		BeforeModelCallbacks: []llmagent.BeforeModelCallback{rounds.before},
		AfterAgentCallbacks:  []agent.AfterAgentCallback{rounds.after},
	})
	if err != nil {
		return nil, err
	}
	sessions := session.InMemoryService()
	adkRunner, err := runner.New(runner.Config{
		AppName:           appName,
		Agent:             adkAgent,
		SessionService:    sessions,
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, err
	}
	return &Agent{runner: adkRunner, sessions: sessions, rounds: rounds}, nil
}

// Reply answers one user message. prior is that chat's history without the
// current message; durable history itself lives in the JSON store.
func (a *Agent) Reply(ctx context.Context, chatID string, prior []Message, userText string) (string, error) {
	sessionID := chatID + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	defer func() {
		_ = a.sessions.Delete(context.Background(), &session.DeleteRequest{
			AppName:   appName,
			UserID:    chatID,
			SessionID: sessionID,
		})
	}()

	prompt := transcript(prior, userText)
	var final string
	for event, err := range a.runner.Run(ctx, chatID, sessionID, genai.NewContentFromText(prompt, genai.RoleUser), agent.RunConfig{}) {
		if err != nil {
			return "", err
		}
		if event == nil || event.Partial || !event.IsFinalResponse() {
			continue
		}
		if text := eventText(event.Content); text != "" {
			final = text
		}
	}
	final = strings.TrimSpace(final)
	if final == "" {
		return "", fmt.Errorf("agent returned no text")
	}
	return final, nil
}

func transcript(prior []Message, current string) string {
	if len(prior) == 0 {
		return current
	}
	if len(prior) > maxPromptMessages {
		prior = prior[len(prior)-maxPromptMessages:]
	}
	var b strings.Builder
	b.WriteString("Previous messages:\n")
	for _, message := range prior {
		role := message.Role
		if role == "" {
			role = "user"
		}
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(message.Text)
		b.WriteString("\n")
	}
	b.WriteString("\nUser message:\n")
	b.WriteString(current)
	return b.String()
}

func eventText(content *genai.Content) string {
	if content == nil {
		return ""
	}
	var b strings.Builder
	for _, part := range content.Parts {
		if part == nil || part.Thought || part.Text == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(part.Text)
	}
	return b.String()
}

type roundCap struct {
	max    int
	mu     sync.Mutex
	counts map[string]int
}

func newRoundCap(max int) *roundCap {
	return &roundCap{max: max, counts: map[string]int{}}
}

func (c *roundCap) before(ctx agent.Context, _ *model.LLMRequest) (*model.LLMResponse, error) {
	id := ctx.InvocationID()
	c.mu.Lock()
	c.counts[id]++
	n := c.counts[id]
	c.mu.Unlock()
	if n <= c.max {
		return nil, nil
	}
	return &model.LLMResponse{
		Content: &genai.Content{
			Role:  string(genai.RoleModel),
			Parts: []*genai.Part{{Text: "Я остановился на лимите запросов к инструментам. Спросите более узко: одну остановку или один маршрут."}},
		},
		TurnComplete: true,
	}, nil
}

func (c *roundCap) after(ctx agent.Context) (*genai.Content, error) {
	c.mu.Lock()
	delete(c.counts, ctx.InvocationID())
	c.mu.Unlock()
	return nil, nil
}
