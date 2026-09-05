package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"qwen2api-go/toolcall"
)

type ProviderType string

const (
	ProviderDeepSeek ProviderType = "deepseek"
	ProviderMistral  ProviderType = "mistral"
	ProviderChatGPT  ProviderType = "chatgpt"
	ProviderQwen     ProviderType = "qwen"
)

type TargetSpec struct {
	Provider ProviderType
	Model    string
}

func ParseTarget(target string) TargetSpec {
	parts := strings.SplitN(target, ":", 2)
	if len(parts) == 2 {
		return TargetSpec{
			Provider: ProviderType(strings.ToLower(strings.TrimSpace(parts[0]))),
			Model:    strings.TrimSpace(parts[1]),
		}
	}

	// Auto-detect provider from model name if no prefix
	m := strings.ToLower(strings.TrimSpace(target))
	switch {
	case strings.HasPrefix(m, "deepseek"):
		return TargetSpec{Provider: ProviderDeepSeek, Model: m}
	case strings.HasPrefix(m, "mistral") || strings.HasPrefix(m, "vibe") || strings.HasPrefix(m, "le-chat"):
		return TargetSpec{Provider: ProviderMistral, Model: m}
	case strings.HasPrefix(m, "gpt-") || strings.HasPrefix(m, "chatgpt") || strings.HasPrefix(m, "o1") || strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "o4"):
		return TargetSpec{Provider: ProviderChatGPT, Model: m}
	case strings.HasPrefix(m, "qwen") || strings.HasPrefix(m, "wanx"):
		return TargetSpec{Provider: ProviderQwen, Model: m}
	default:
		return TargetSpec{Provider: ProviderDeepSeek, Model: m}
	}
}

// HelperClient communicates with the Node helper daemon on 127.0.0.1:18835
type HelperClient struct {
	baseURL string
	http    *http.Client
}

func NewHelperClient(baseURL string) *HelperClient {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:18835"
	}
	return &HelperClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout: 180 * time.Second,
		},
	}
}

func (hc *HelperClient) StreamChat(ctx context.Context, endpoint string, prompt string, webSearch bool, onEvent func(UpstreamEvent) error) error {
	reqBody := map[string]any{
		"prompt":     prompt,
		"stream":     true,
		"web_search": webSearch,
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hc.baseURL+endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := hc.http.Do(req)
	if err != nil {
		return fmt.Errorf("gagal menghubungi helper %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("helper %s HTTP %d: %s", endpoint, resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	receivedAnyContent := false

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" || line == "data: [DONE]" {
			if line == "data: [DONE]" {
				break
			}
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			jsonStr := strings.TrimPrefix(line, "data: ")
			var evt map[string]any
			if jsonErr := json.Unmarshal([]byte(jsonStr), &evt); jsonErr != nil {
				continue
			}

			evtType, _ := evt["type"].(string)
			switch evtType {
			case "content":
				text, _ := evt["text"].(string)
				if text != "" {
					receivedAnyContent = true
					if err := onEvent(UpstreamEvent{
						Type:    "delta",
						Phase:   "answer",
						Content: text,
					}); err != nil {
						return err
					}
				}
			case "reasoning":
				text, _ := evt["text"].(string)
				if text != "" {
					receivedAnyContent = true
					if err := onEvent(UpstreamEvent{
						Type:    "delta",
						Phase:   "thinking",
						Content: text,
					}); err != nil {
						return err
					}
				}
			case "error":
				errMsg, _ := evt["error"].(string)
				return fmt.Errorf("upstream stream error: %s", errMsg)
			case "done":
				// stream finished
			}
		}
	}

	if !receivedAnyContent {
		return fmt.Errorf("tidak menerima respon teks dari upstream %s", endpoint)
	}
	return nil
}

func (app *App) resolveTargetChain(modelName string) []string {
	if app.comboStore != nil {
		if targets, ok := app.comboStore.ResolveTargets(modelName); ok && len(targets) > 0 {
			return targets
		}
	}
	m := strings.ToLower(strings.TrimSpace(modelName))
	if strings.HasPrefix(m, "gpt-") || strings.HasPrefix(m, "chatgpt") || strings.HasPrefix(m, "o1") || strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "o4") {
		return []string{modelName, "qwen:qwen-max", "deepseek:deepseek-chat"}
	}
	return []string{modelName}
}

func (app *App) allSupportedModels() []map[string]any {
	var list []map[string]any

	// 1. Combo Models
	if app.comboStore != nil {
		for _, c := range app.comboStore.List() {
			if !c.Enabled {
				continue
			}
			list = append(list, map[string]any{
				"id":           c.ID,
				"object":       "model",
				"created":      1700000000,
				"owned_by":     "convertsw",
				"display_name": c.Name,
				"family":       "combo",
				"description":  c.Description,
				"capabilities": map[string]bool{
					"thinking": true,
					"search":   true,
				},
			})
		}
	}

	// 2. Individual Provider Models
	builtIns := []struct {
		ID       string
		Family   string
		OwnedBy  string
		Name     string
		Thinking bool
		Search   bool
	}{
		// DeepSeek
		{ID: "deepseek-chat", Family: "deepseek", OwnedBy: "deepseek", Name: "DeepSeek-V3", Thinking: false, Search: false},
		{ID: "deepseek-reasoner", Family: "deepseek", OwnedBy: "deepseek", Name: "DeepSeek-R1 (Reasoning)", Thinking: true, Search: false},
		{ID: "deepseek-v3", Family: "deepseek", OwnedBy: "deepseek", Name: "DeepSeek-V3", Thinking: false, Search: false},
		{ID: "deepseek-r1", Family: "deepseek", OwnedBy: "deepseek", Name: "DeepSeek-R1", Thinking: true, Search: false},
		{ID: "deepseek-chat-search", Family: "deepseek", OwnedBy: "deepseek", Name: "DeepSeek-V3 (Search)", Thinking: false, Search: true},
		{ID: "deepseek-reasoner-search", Family: "deepseek", OwnedBy: "deepseek", Name: "DeepSeek-R1 (Search)", Thinking: true, Search: true},

		// ChatGPT
		{ID: "gpt-4o", Family: "chatgpt", OwnedBy: "openai", Name: "ChatGPT (GPT-4o)", Thinking: false, Search: false},
		{ID: "gpt-4o-mini", Family: "chatgpt", OwnedBy: "openai", Name: "ChatGPT (GPT-4o Mini)", Thinking: false, Search: false},
		{ID: "gpt-5-6", Family: "chatgpt", OwnedBy: "openai", Name: "ChatGPT (GPT-5/4o Latest)", Thinking: false, Search: false},
		{ID: "chatgpt-auto", Family: "chatgpt", OwnedBy: "openai", Name: "ChatGPT Auto", Thinking: false, Search: false},
		{ID: "chatgpt-search", Family: "chatgpt", OwnedBy: "openai", Name: "ChatGPT (Search)", Thinking: false, Search: true},

		// Mistral AI
		{ID: "mistral-chat", Family: "mistral", OwnedBy: "mistral", Name: "Mistral Le Chat", Thinking: false, Search: false},
		{ID: "mistral-large", Family: "mistral", OwnedBy: "mistral", Name: "Mistral Large", Thinking: false, Search: false},
		{ID: "mistral-small", Family: "mistral", OwnedBy: "mistral", Name: "Mistral Small", Thinking: false, Search: false},
		{ID: "mistral-vibe", Family: "mistral", OwnedBy: "mistral", Name: "Mistral Vibe Assistant", Thinking: false, Search: false},
		{ID: "mistral-search", Family: "mistral", OwnedBy: "mistral", Name: "Mistral (Search)", Thinking: false, Search: true},

		// Qwen Official Models (Sesuai Scrape Terkini)
		{ID: "qwen3.8-max", Family: "qwen", OwnedBy: "qwen", Name: "Qwen 3.8 Max (Flagship)", Thinking: true, Search: true},
		{ID: "qwen3.8-max-thinking", Family: "qwen", OwnedBy: "qwen", Name: "Qwen 3.8 Max (Thinking)", Thinking: true, Search: false},
		{ID: "qwen3.8-max-search", Family: "qwen", OwnedBy: "qwen", Name: "Qwen 3.8 Max (Web Search)", Thinking: false, Search: true},
		{ID: "qwen3.8-max-deep-research", Family: "qwen", OwnedBy: "qwen", Name: "Qwen 3.8 Max (Deep Research)", Thinking: true, Search: true},
		{ID: "qwen3.7-max", Family: "qwen", OwnedBy: "qwen", Name: "Qwen 3.7 Max", Thinking: true, Search: true},
		{ID: "qwen3.7-plus", Family: "qwen", OwnedBy: "qwen", Name: "Qwen 3.7 Plus", Thinking: true, Search: true},
		{ID: "qwen3.7-plus-thinking", Family: "qwen", OwnedBy: "qwen", Name: "Qwen 3.7 Plus (Thinking)", Thinking: true, Search: false},
		{ID: "qwen3.7-plus-search", Family: "qwen", OwnedBy: "qwen", Name: "Qwen 3.7 Plus (Web Search)", Thinking: false, Search: true},
		{ID: "qwen3.6-plus", Family: "qwen", OwnedBy: "qwen", Name: "Qwen 3.6 Plus", Thinking: true, Search: true},
		{ID: "qwen3.5-plus", Family: "qwen", OwnedBy: "qwen", Name: "Qwen 3.5 Plus", Thinking: true, Search: true},
		{ID: "qwen3.5-omni-plus", Family: "qwen", OwnedBy: "qwen", Name: "Qwen 3.5 Omni Plus (Multimodal)", Thinking: false, Search: false},
		{ID: "wanx2.1-t2i", Family: "qwen", OwnedBy: "qwen", Name: "WanX 2.1 Text-to-Image", Thinking: false, Search: false},
		{ID: "wanx2.1-i2v", Family: "qwen", OwnedBy: "qwen", Name: "WanX 2.1 Image-to-Video", Thinking: false, Search: false},
		// Alias Kompatibel
		{ID: "qwen-max", Family: "qwen", OwnedBy: "qwen", Name: "Qwen Max (Alias -> 3.8 Max)", Thinking: true, Search: true},
		{ID: "qwen-plus", Family: "qwen", OwnedBy: "qwen", Name: "Qwen Plus (Alias -> 3.7 Plus)", Thinking: true, Search: true},
		{ID: "qwen-turbo", Family: "qwen", OwnedBy: "qwen", Name: "Qwen Turbo (Alias -> 3.5 Plus)", Thinking: false, Search: false},
		{ID: "qwen2.5-72b-instruct", Family: "qwen", OwnedBy: "qwen", Name: "Qwen 2.5 72B Instruct", Thinking: false, Search: false},
	}

	for _, m := range builtIns {
		list = append(list, map[string]any{
			"id":           m.ID,
			"object":       "model",
			"created":      1700000000,
			"owned_by":     m.OwnedBy,
			"display_name": m.Name,
			"family":       m.Family,
			"capabilities": map[string]bool{
				"thinking": m.Thinking,
				"search":   m.Search,
			},
		})
	}

	return list
}

func (app *App) executeSingleTargetStream(ctx context.Context, target string, req StandardRequest, preferredEmail string, onEvent func(UpstreamEvent) error) error {
	spec := ParseTarget(target)
	switch spec.Provider {
	case ProviderMistral:
		app.logInfo(ctx, "[Router] Executing Mistral AI", "model", spec.Model)
		return app.helperClient.StreamChat(ctx, "/mistral/chat", req.Prompt, req.EnableSearch, onEvent)

	case ProviderChatGPT:
		app.logInfo(ctx, "[Router] Executing ChatGPT", "model", spec.Model)
		return app.helperClient.StreamChat(ctx, "/chatgpt/chat", req.Prompt, req.EnableSearch, onEvent)

	case ProviderQwen:
		app.logInfo(ctx, "[Router] Executing Qwen AI", "model", spec.Model)
		acc, err := app.accounts.AcquireForProvider(ctx, "qwen", preferredEmail, "chat")
		if err != nil {
			return fmt.Errorf("tidak ada akun Qwen aktif di pool: %w", err)
		}
		defer app.accounts.Release(acc)

		chatID, err := app.alibabaQwen.CreateChat(ctx, acc.Token, acc.Cookies, spec.Model, "t2t")
		if err != nil {
			return fmt.Errorf("gagal membuat sesi Qwen chat: %w", err)
		}
		defer app.alibabaQwen.DeleteChat(context.Background(), acc.Token, acc.Cookies, chatID)

		payload, resolvedModel := BuildQwenChatPayload(chatID, spec.Model, req.Prompt, req.ThinkingEnabled, req.EnableSearch)
		app.logInfo(ctx, "[Qwen] Stream request prepared", "requested_model", spec.Model, "resolved_model", resolvedModel)
		return app.alibabaQwen.StreamChat(ctx, acc.Token, acc.Cookies, chatID, payload, onEvent)

	case ProviderDeepSeek:
		fallthrough
	default:
		app.logInfo(ctx, "[Router] Executing DeepSeek", "model", spec.Model)
		acc, chatID, reused, err := app.acquireCompletionChat(ctx, req, preferredEmail)
		if err != nil {
			return err
		}
		defer app.accounts.Release(acc)
		defer asyncDeleteChat(app.client, acc.Token, chatID)
		setRequestLogFields(ctx, "chat_id", chatID)
		app.logInfo(ctx, "创建上游会话", "chat_type", req.ChatType, "prewarmed", reused)

		payload := buildChatPayload(chatID, spec.Model, req.Prompt, req.ToolEnabled, req.UpstreamFiles, req.ChatType, nil, req.ThinkingEnabled, req.EnableSearch)
		return app.client.StreamChat(ctx, acc.Token, chatID, payload, onEvent)
	}
}

// Router Completion with Multi-Provider Fallback
func (app *App) runCompletionWithHooksMultiProvider(ctx context.Context, req StandardRequest, preferredEmail string, hooks *completionStreamHooks) (CompletionResult, error) {
	if req.BoundAccount != nil {
		preferredEmail = req.BoundAccount.Email
	}
	if preferredEmail == "" {
		preferredEmail = req.PreferredEmail
	}

	targets := app.resolveTargetChain(req.ResolvedModel)
	if len(targets) == 0 {
		targets = []string{req.ResolvedModel}
	}

	app.logInfo(ctx, "[Router] Starting completion with target chain", "model", req.ResolvedModel, "targets", strings.Join(targets, " -> "))

	var lastErr error
	for targetIdx, target := range targets {
		start := time.Now()
		result := CompletionResult{FinishReason: "stop"}
		var sieve *toolcall.ToolSieve
		if req.ToolEnabled {
			sieve = toolcall.NewToolSieve(req.Tools)
			app.logInfo(ctx, "[Collect] tool filter enabled", "tools", strings.Join(req.ToolNames, ","))
		}

		streamContent := func(reasoning bool, text string) error {
			if text == "" {
				return nil
			}
			if reasoning {
				result.ReasoningText += text
				if hooks != nil && hooks.OnReasoningDelta != nil && !shouldBufferStreamTextDeltas(req) {
					return hooks.OnReasoningDelta(text)
				}
				return nil
			}
			result.AnswerText += text
			if hooks != nil && hooks.OnAnswerDelta != nil && !shouldBufferStreamTextDeltas(req) {
				return hooks.OnAnswerDelta(text)
			}
			return nil
		}

		err := app.executeSingleTargetStream(ctx, target, req, preferredEmail, func(evt UpstreamEvent) error {
			result.Events = append(result.Events, evt)
			if evt.Type != "delta" || evt.Content == "" {
				return nil
			}
			reasoning := strings.Contains(evt.Phase, "thinking") || strings.Contains(evt.Phase, "reasoning")
			if sieve == nil {
				return streamContent(reasoning, evt.Content)
			}
			for _, sieveEvt := range sieve.ProcessChunk(evt.Content) {
				switch sieveEvt.Type {
				case "content":
					if err := streamContent(reasoning, sieveEvt.Text); err != nil {
						return err
					}
				case "tool_calls":
					if len(sieveEvt.Calls) == 0 {
						continue
					}
					stage := "tool_sieve_stream"
					if reasoning {
						stage = "tool_sieve_reasoning_stream"
					}
					captured, err := app.captureSieveToolCalls(ctx, req, &result, sieveEvt.Calls, stage, func(calls []ParsedToolCall) error {
						if hooks != nil && hooks.OnToolCalls != nil {
							return hooks.OnToolCalls(calls)
						}
						return nil
					})
					if err != nil {
						return err
					}
					if !captured {
						continue
					}
					return errToolSieveDetected
				}
			}
			if reasoning {
				if blocked := extractBlockedToolNames(result.ReasoningText, req.ToolNames); len(blocked) > 0 {
					result.FinishReason = "blocked_tool_name:" + blocked[0]
					suppressBlockedToolNameOutput(&result, req.ToolNames)
					return errToolSieveDetected
				}
			} else {
				if blocked := extractBlockedToolNames(result.AnswerText, req.ToolNames); len(blocked) > 0 {
					result.FinishReason = "blocked_tool_name:" + blocked[0]
					suppressBlockedToolNameOutput(&result, req.ToolNames)
					return errToolSieveDetected
				}
			}
			return nil
		})

		if err == nil || errors.Is(err, errToolSieveDetected) {
			if req.ToolEnabled && len(result.ToolCalls) == 0 && sieve != nil {
				for _, sieveEvt := range sieve.Flush() {
					switch sieveEvt.Type {
					case "content":
						if strings.TrimSpace(sieveEvt.Text) != "" {
							_ = streamContent(false, sieveEvt.Text)
						}
					case "tool_calls":
						if len(sieveEvt.Calls) > 0 {
							_, _ = app.captureSieveToolCalls(ctx, req, &result, sieveEvt.Calls, "tool_sieve_flush", func(calls []ParsedToolCall) error {
								if hooks != nil && hooks.OnToolCalls != nil {
									return hooks.OnToolCalls(calls)
								}
								return nil
							})
						}
					}
				}
			}
			app.logInfo(ctx, "[Router] Target completed successfully", "target", target, "events", len(result.Events), "duration_ms", time.Since(start).Milliseconds())
			return result, nil
		}

		lastErr = err
		app.logWarn(ctx, "[Router] Target failed, attempting fallback to next provider", "failed_target", target, "attempt", targetIdx+1, "total", len(targets), "error", err)
	}

	return CompletionResult{}, fmt.Errorf("all providers in target chain failed: %w", lastErr)
}

// Admin API Handlers for Combos and Providers
func (app *App) adminListCombos(w http.ResponseWriter, r *http.Request) {
	if _, ok := app.verifyAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "combos": app.comboStore.List()})
}

func (app *App) adminSaveCombo(w http.ResponseWriter, r *http.Request) {
	if _, ok := app.verifyAdmin(w, r); !ok {
		return
	}
	var body ComboModel
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := app.comboStore.Set(body); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "combo": body})
}

func (app *App) adminDeleteCombo(w http.ResponseWriter, r *http.Request) {
	if _, ok := app.verifyAdmin(w, r); !ok {
		return
	}
	id := r.PathValue("id")
	if err := app.comboStore.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": id})
}

func (app *App) adminListProviders(w http.ResponseWriter, r *http.Request) {
	if _, ok := app.verifyAdmin(w, r); !ok {
		return
	}
	providers := []map[string]any{
		{
			"id":          "deepseek",
			"name":        "DeepSeek AI",
			"status":      "active",
			"type":        "official_reverse",
			"description": "DeepSeek-V3 & DeepSeek-R1 (Deep Reasoning) with Wasm SHA-3 PoW bypass",
			"models":      []string{"deepseek-chat", "deepseek-reasoner", "deepseek-v3", "deepseek-r1", "deepseek-vision"},
		},
		{
			"id":          "chatgpt",
			"name":        "ChatGPT (OpenAI)",
			"status":      "active",
			"type":        "official_reverse",
			"description": "OpenAI ChatGPT with Sentinel FNV-1a PoW & Turnstile bypass (GPT-5/4o)",
			"models":      []string{"gpt-4o", "gpt-4o-mini", "gpt-5-6", "chatgpt-auto", "chatgpt-search"},
		},
		{
			"id":          "mistral",
			"name":        "Mistral AI",
			"status":      "active",
			"type":        "official_reverse",
			"description": "Mistral Le Chat mobile reverse with native web search",
			"models":      []string{"mistral-chat", "mistral-large", "mistral-small", "mistral-vibe", "mistral-search"},
		},
		{
			"id":          "qwen",
			"name":        "Qwen AI (Alibaba)",
			"status":      "active",
			"type":        "official_reverse",
			"description": "Alibaba Cloud Qwen AI with Android APK protocol, Anti-WAF bypass, and multimodal",
			"models": []string{
				"qwen3.8-max", "qwen3.8-max-thinking", "qwen3.8-max-search", "qwen3.8-max-deep-research",
				"qwen3.7-max", "qwen3.7-plus", "qwen3.7-plus-thinking", "qwen3.7-plus-search",
				"qwen3.6-plus", "qwen3.5-plus", "qwen3.5-omni-plus",
				"wanx2.1-t2i", "wanx2.1-i2v",
				"qwen-max", "qwen-plus", "qwen-turbo", "qwen2.5-72b-instruct",
			},
		},
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "providers": providers})
}
