package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"
)

const (
	qwenChatBaseURL         = "https://chat.qwen.ai"
	defaultQwenAndroidUA    = "Dalvik/2.1.0 (Linux; U; Android 15; 25028RN03A Build/AP3A.240905.015.A2) AliApp(QWENCHAT/2.5.1) AppType/Release AplusBridgeLite,Dalvik/2.1.0 (Linux; U; Android 15; 25028RN03A Build/AP3A.240905.015.A2)"
	defaultAndroidAppWaf    = "Z9Tr56YmQpXcO2K_d_3nAbJvRqMLFW8HTNjvRguWHEowM1xY"
)

func alibabaQwenDeviceID() string {
	b := make([]byte, 16)
	_, _ = cryptorand.Read(b)
	h := md5.Sum(b)
	return "ai" + hex.EncodeToString(h[:])
}

type QwenModelSpec struct {
	BaseModel       string
	ResolvedModel   string
	ChatType        string
	ThinkingEnabled bool
	AutoSearch      bool
	Mode            string
}

func parseQwenModelMode(model string) QwenModelSpec {
	trimmed := strings.TrimSpace(model)
	lowered := strings.ToLower(trimmed)

	suffixes := []struct {
		suffix        string
		chatType      string
		forceThinking bool
		autoSearch    bool
		mode          string
	}{
		{"-deep-research", "deep_research", false, true, "deep_research"},
		{"-deep_research", "deep_research", false, true, "deep_research"},
		{"-web-dev", "web_dev", false, false, "webdev"},
		{"-webdev", "web_dev", false, false, "webdev"},
		{"-thinking", "t2t", true, false, "thinking"},
		{"-search", "t2t", false, true, "search"},
		{"-image", "t2i", false, false, "image"},
		{"-video", "t2v", false, false, "video"},
		{"-slides", "slides", false, false, "slides"},
		{"-t2i", "t2i", false, false, "image"},
		{"-t2v", "t2v", false, false, "video"},
	}

	base := trimmed
	chatType := "t2t"
	forceThinking := false
	autoSearch := false
	mode := "chat"

	for _, s := range suffixes {
		if strings.HasSuffix(lowered, s.suffix) && len(trimmed) > len(s.suffix) {
			base = strings.TrimSpace(trimmed[:len(trimmed)-len(s.suffix)])
			chatType = s.chatType
			forceThinking = s.forceThinking
			autoSearch = s.autoSearch
			mode = s.mode
			break
		}
	}

	resolvedBase := resolveQwenBaseModel(base)
	return QwenModelSpec{
		BaseModel:       base,
		ResolvedModel:   resolvedBase,
		ChatType:        chatType,
		ThinkingEnabled: forceThinking,
		AutoSearch:      autoSearch,
		Mode:            mode,
	}
}

func resolveQwenBaseModel(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	switch m {
	case "qwen-max", "qwen-max-latest", "qwen3.8", "qwen3.8-max":
		return "qwen3.8-max"
	case "qwen3.7", "qwen3.7-max":
		return "qwen3.7-max"
	case "qwen-plus", "qwen3.7-plus":
		return "qwen3.7-plus"
	case "qwen2.5-72b-instruct", "qwen", "qwen3.6", "qwen3.6-plus":
		return "qwen3.6-plus"
	case "qwen-turbo", "qwen-flash", "qwen3.5", "qwen3.5-plus", "qwen3.5-turbo":
		return "qwen3.5-plus"
	case "qwen3.5-omni", "qwen3.5-omni-plus", "qwen-omni":
		return "qwen3.5-omni-plus"
	case "wanx2.1-t2i", "wanx-image", "qwen-image", "wanx2.1-i2v", "wanx-video", "qwen-video", "wanx2.1", "wanx", "wanx-v2.1":
		return "qwen3.7-plus"
	default:
		if strings.HasPrefix(m, "qwen3.") {
			return m
		}
		if strings.Contains(m, "wanx") {
			return "qwen3.7-plus"
		}
		return "qwen3.8-max"
	}
}

func BuildQwenChatPayload(chatID, model, content string, thinkingEnabled *bool, enableSearch bool) (map[string]any, string) {
	spec := parseQwenModelMode(model)
	ts := time.Now().Unix()

	thinking := spec.ThinkingEnabled
	autoThinking := spec.ThinkingEnabled
	thinkingMode := "Disabled"
	if spec.ThinkingEnabled {
		thinkingMode = "Auto"
	} else if thinkingEnabled != nil {
		thinking = *thinkingEnabled
		autoThinking = *thinkingEnabled
		if thinking {
			thinkingMode = "Auto"
		} else {
			thinkingMode = "Disabled"
		}
	}

	search := spec.AutoSearch || enableSearch

	isImage := spec.ChatType == "t2i" || spec.Mode == "image"
	isVideo := spec.ChatType == "t2v" || spec.Mode == "video"

	featureConfig := map[string]any{}
	messageChatType := spec.ChatType
	subChatType := spec.ChatType
	messageMeta := map[string]any{"subChatType": spec.ChatType}

	if isImage {
		ratio := "1:1"
		featureConfig = map[string]any{
			"thinking_enabled": false, "output_schema": "phase", "auto_thinking": false,
			"thinking_mode": "off", "auto_search": false, "code_interpreter": false,
			"function_calling": false, "plugins_enabled": true, "image_generation": true,
			"default_aspect_ratio": ratio,
		}
		messageChatType = "t2t"
		subChatType = "t2i"
		messageMeta = map[string]any{"subChatType": "t2i", "mode": "image_generation", "aspectRatio": ratio, "size": ratio}
	} else if isVideo {
		ratio := "16:9"
		featureConfig = map[string]any{
			"thinking_enabled": false, "output_schema": "phase", "auto_thinking": false,
			"thinking_mode": "off", "auto_search": false, "code_interpreter": false,
			"function_calling": false, "plugins_enabled": true, "video_generation": true,
			"default_aspect_ratio": ratio,
		}
		messageChatType = "t2v"
		subChatType = "t2v"
		messageMeta = map[string]any{"subChatType": "t2v", "mode": "video_generation", "aspectRatio": ratio, "size": ratio}
	} else {
		featureConfig = map[string]any{
			"thinking_enabled":     thinking,
			"output_schema":        "phase",
			"research_mode":        "normal",
			"auto_thinking":        autoThinking,
			"thinking_mode":        thinkingMode,
			"thinking_format":      "summary",
			"auto_search":          search,
			"code_interpreter":     false,
			"plugins_enabled":      false,
			"function_calling":     false,
			"enable_tools":         false,
			"enable_function_call": false,
			"tool_choice":          "none",
		}
	}

	fid := fmt.Sprintf("fid-%x", ts)
	cid := fmt.Sprintf("cid-%x", ts)

	payload := map[string]any{
		"stream":             true,
		"version":            "2.1",
		"incremental_output": true,
		"chat_id":            chatID,
		"chat_mode":          "normal",
		"model":              spec.ResolvedModel,
		"parent_id":          nil,
		"messages": []map[string]any{{
			"fid":            fid,
			"parentId":       nil,
			"childrenIds":    []string{cid},
			"role":           "user",
			"content":        content,
			"user_action":    "chat",
			"files":          []map[string]any{},
			"timestamp":      ts,
			"models":         []string{spec.ResolvedModel},
			"chat_type":      messageChatType,
			"feature_config": featureConfig,
			"extra":          map[string]any{"meta": messageMeta},
			"sub_chat_type":  subChatType,
			"parent_id":      nil,
		}},
		"timestamp": ts,
	}
	if isImage || isVideo {
		payload["size"] = "1:1"
	}
	return payload, spec.ResolvedModel
}

type AlibabaQwenClient struct {
	http    *http.Client
	mu      sync.Mutex
	deleted map[string]bool
}

func NewAlibabaQwenClient() *AlibabaQwenClient {
	jar, _ := cookiejar.New(nil)
	return &AlibabaQwenClient{
		http: &http.Client{
			Jar:     jar,
			Timeout: 120 * time.Second,
		},
		deleted: make(map[string]bool),
	}
}

func (c *AlibabaQwenClient) headersForToken(token, cookies string) http.Header {
	h := http.Header{}
	if token != "" {
		if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
			h.Set("Authorization", "Bearer "+token)
		} else {
			h.Set("Authorization", token)
		}
	}
	h.Set("x-request-id", fmt.Sprintf("req-%d", time.Now().UnixNano()))
	h.Set("x-device-id", alibabaQwenDeviceID())
	h.Set("User-Agent", defaultQwenAndroidUA)
	h.Set("Connection", "Keep-Alive")
	h.Set("Accept", "application/json")
	h.Set("X-Platform", "android")
	h.Set("source", "app")
	h.Set("Accept-Language", "en-US")
	h.Set("Accept-Charset", "UTF-8")
	h.Set("Cache-Control", "no-store")
	h.Set("app_waf", defaultAndroidAppWaf)
	h.Set("Content-Type", "application/json; charset=UTF-8")
	if cookies != "" {
		h.Set("Cookie", cookies)
	}
	return h
}

func (c *AlibabaQwenClient) SignIn(ctx context.Context, email, password string) (*SignInResult, error) {
	if strings.TrimSpace(email) == "" || strings.TrimSpace(password) == "" {
		return nil, errors.New("Email dan kata sandi tidak boleh kosong")
	}
	hash := sha256.Sum256([]byte(password))
	hashedPassword := hex.EncodeToString(hash[:])

	reqBody := map[string]any{
		"email":    strings.TrimSpace(email),
		"password": hashedPassword,
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qwenChatBaseURL+"/api/v2/auths/signin", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header = c.headersForToken("", "")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Gagal menghubungi server Qwen: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res map[string]any
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return nil, fmt.Errorf("Gagal mengurai respons signin: %s", string(respBytes))
	}
	if success, ok := res["success"].(bool); !ok || !success {
		data, _ := res["data"].(map[string]any)
		details := stringValue(data, "details", stringValue(res, "message", "Login gagal"))
		return nil, fmt.Errorf("Login Qwen gagal: %s", details)
	}
	data, _ := res["data"].(map[string]any)
	token := stringValue(data, "token", "")
	if token == "" {
		return nil, errors.New("Token tidak ditemukan dalam respons login Qwen")
	}
	var cookieParts []string
	for _, cookie := range resp.Cookies() {
		cookieParts = append(cookieParts, cookie.Name+"="+cookie.Value)
	}
	cookieStr := strings.Join(cookieParts, "; ")
	return &SignInResult{Token: token, Cookies: cookieStr}, nil
}

func (c *AlibabaQwenClient) VerifyTokenDetail(ctx context.Context, token, cookies string) TokenVerifyResult {
	if token == "" {
		return TokenVerifyResult{Valid: false, StatusCode: "invalid", Error: "Token kosong"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, qwenChatBaseURL+"/api/v2/chats?limit=1", nil)
	if err != nil {
		return TokenVerifyResult{Valid: false, StatusCode: "invalid", Error: err.Error()}
	}
	req.Header = c.headersForToken(token, cookies)
	resp, err := c.http.Do(req)
	if err != nil {
		return TokenVerifyResult{Valid: false, StatusCode: "invalid", Error: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		var res map[string]any
		if err := json.Unmarshal(respBytes, &res); err == nil {
			if success, ok := res["success"].(bool); ok && !success {
				return TokenVerifyResult{Valid: false, StatusCode: "invalid", Error: "Token Qwen kedaluwarsa atau tidak valid"}
			}
		}
		return TokenVerifyResult{Valid: true, StatusCode: "valid"}
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return TokenVerifyResult{Valid: false, StatusCode: "banned", Error: fmt.Sprintf("HTTP %d Unauthorized/Forbidden", resp.StatusCode)}
	}
	return TokenVerifyResult{Valid: true, StatusCode: "valid"}
}

func (c *AlibabaQwenClient) CreateChat(ctx context.Context, token, cookies, model, chatType string) (string, error) {
	spec := parseQwenModelMode(model)
	if chatType == "" {
		chatType = spec.ChatType
	}
	ts := time.Now().Unix()
	body := map[string]any{
		"title":     fmt.Sprintf("api_%d", ts),
		"models":    []string{spec.ResolvedModel},
		"chat_mode": "normal",
		"chat_type": chatType,
		"timestamp": ts,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qwenChatBaseURL+"/api/v2/chats/new", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header = c.headersForToken(token, cookies)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create_chat Qwen HTTP %d: %s", resp.StatusCode, string(respBytes))
	}
	var res map[string]any
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return "", err
	}
	if success, ok := res["success"].(bool); ok && !success {
		return "", fmt.Errorf("Qwen CreateChat rejected: %s", string(respBytes))
	}
	data, _ := res["data"].(map[string]any)
	id, _ := data["id"].(string)
	if id == "" {
		if strings.Contains(string(respBytes), "rgv587_flag") || strings.Contains(string(respBytes), "punish") {
			return "", errors.New("Alibaba Cloud WAF Captcha / Anti-Bot: Sesi atau IP terhalang verifikasi keamanan upstream.")
		}
		return "", fmt.Errorf("Qwen returned no chat ID: %s", string(respBytes))
	}
	return id, nil
}

func (c *AlibabaQwenClient) DeleteChat(ctx context.Context, token, cookies, chatID string) bool {
	if token == "" || chatID == "" {
		return true
	}
	c.mu.Lock()
	if c.deleted[chatID] {
		c.mu.Unlock()
		return true
	}
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, qwenChatBaseURL+"/api/v2/chats/"+chatID, nil)
	if err != nil {
		return false
	}
	req.Header = c.headersForToken(token, cookies)
	resp, err := c.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	c.mu.Lock()
	c.deleted[chatID] = true
	c.mu.Unlock()
	return resp.StatusCode == http.StatusOK || resp.StatusCode == 204
}

func (c *AlibabaQwenClient) StreamChat(ctx context.Context, token, cookies, chatID string, payload map[string]any, onEvent func(UpstreamEvent) error) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qwenChatBaseURL+"/api/v2/chat/completions?chat_id="+chatID, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header = c.headersForToken(token, cookies)
	req.Header.Set("Accept", "text/event-stream")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Qwen StreamChat HTTP %d: %s", resp.StatusCode, string(body))
	}

	reader := bufio.NewReader(resp.Body)
	receivedAny := false
	for {
		line, readErr := reader.ReadString('\n')
		if strings.HasPrefix(line, "{") {
			var jsonEvt map[string]any
			if err := json.Unmarshal([]byte(line), &jsonEvt); err == nil {
				if success, ok := jsonEvt["success"].(bool); ok && !success {
					dataMap, _ := jsonEvt["data"].(map[string]any)
					code := stringValue(dataMap, "code", "error")
					details := stringValue(dataMap, "details", stringValue(jsonEvt, "message", "Upstream error"))
					return fmt.Errorf("Qwen upstream error (%s): %s", code, details)
				}
			}
		}
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				break
			}
			var jsonEvt map[string]any
			if err := json.Unmarshal([]byte(data), &jsonEvt); err == nil {
				// Check for error payload
				if success, ok := jsonEvt["success"].(bool); ok && !success {
					dataMap, _ := jsonEvt["data"].(map[string]any)
					code := stringValue(dataMap, "code", "error")
					details := stringValue(dataMap, "details", stringValue(jsonEvt, "message", "Upstream error"))
					return fmt.Errorf("Qwen upstream error (%s): %s", code, details)
				}

				content := ""
				phase := "answer"
				if choices, ok := jsonEvt["choices"].([]any); ok && len(choices) > 0 {
					if choice, ok := choices[0].(map[string]any); ok {
						if delta, ok := choice["delta"].(map[string]any); ok {
							content = stringValue(delta, "content", "")
							p := stringValue(delta, "phase", "")
							if p != "" {
								phase = p
							}
							reasoning := stringValue(delta, "reasoning_content", "")
							if reasoning != "" {
								content = reasoning
								phase = "thinking"
							}
						}
					}
				}
				if content != "" {
					receivedAny = true
					if err := onEvent(UpstreamEvent{Type: "delta", Phase: phase, Content: content}); err != nil {
						return err
					}
				}
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return readErr
		}
	}
	if !receivedAny {
		return errors.New("Upstream Qwen mengembalikan respons kosong (biasanya akun terhalang verifikasi Captcha/WAF atau token kedaluwarsa). Silakan periksa status akun di menu Manajemen Akun.")
	}
	return nil
}

func (c *AlibabaQwenClient) GetChatDetail(ctx context.Context, token, cookies, chatID string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, qwenChatBaseURL+"/api/v2/chats/"+chatID, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header = c.headersForToken(token, cookies)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", err
	}
	return resp.StatusCode, string(raw), nil
}

func (c *AlibabaQwenClient) PostChatCompletion(ctx context.Context, token, cookies, chatID string, payload map[string]any) (int, string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qwenChatBaseURL+"/api/v2/chat/completions?chat_id="+chatID, bytes.NewReader(raw))
	if err != nil {
		return 0, "", err
	}
	req.Header = c.headersForToken(token, cookies)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	rawResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", err
	}
	return resp.StatusCode, string(rawResp), nil
}

func (c *AlibabaQwenClient) GetVisionTaskStatus(ctx context.Context, token, cookies, taskID string) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, qwenChatBaseURL+"/api/v1/tasks/status/"+taskID, nil)
	if err != nil {
		return 0, "", err
	}
	req.Header = c.headersForToken(token, cookies)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	rawResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", err
	}
	return resp.StatusCode, string(rawResp), nil
}

func BuildQwenImagePayload(chatID, model, promptText, ratio string) map[string]any {
	if ratio == "" {
		ratio = "1:1"
	}
	ts := time.Now().Unix()
	resolvedModel := resolveQwenBaseModel(model)

	return map[string]any{
		"stream":             true,
		"version":            "2.1",
		"incremental_output": true,
		"chat_id":            chatID,
		"chat_mode":          "normal",
		"model":              resolvedModel,
		"parent_id":          nil,
		"messages": []map[string]any{{
			"fid":         fmt.Sprintf("fid-img-%x", ts),
			"parentId":    nil,
			"childrenIds": []string{fmt.Sprintf("cid-img-%x", ts)},
			"role":        "user",
			"content":     promptText,
			"user_action": "chat",
			"files":       []map[string]any{},
			"timestamp":   ts,
			"models":      []string{resolvedModel},
			"chat_type":   "t2t",
			"feature_config": map[string]any{
				"thinking_enabled":     false,
				"output_schema":        "phase",
				"auto_thinking":        false,
				"thinking_mode":        "off",
				"auto_search":          false,
				"code_interpreter":     false,
				"function_calling":     false,
				"plugins_enabled":      true,
				"image_generation":     true,
				"default_aspect_ratio": ratio,
			},
			"extra":         map[string]any{"meta": map[string]any{"subChatType": "t2i", "mode": "image_generation", "aspectRatio": ratio, "size": ratio}},
			"sub_chat_type": "t2i",
			"parent_id":     nil,
		}},
		"timestamp": ts,
		"size":      ratio,
	}
}

func BuildQwenVideoPayload(chatID, model, promptText, ratio string) map[string]any {
	if ratio == "" {
		ratio = "16:9"
	}
	ts := time.Now().Unix()
	resolvedModel := resolveQwenBaseModel(model)

	return map[string]any{
		"stream":                   false,
		"version":                  "2.1",
		"incremental_output":       false,
		"chat_id":                  chatID,
		"chat_mode":                "normal",
		"model":                    resolvedModel,
		"parent_id":                nil,
		"messages": []map[string]any{{
			"fid":         fmt.Sprintf("fid-vid-%x", ts),
			"parentId":    nil,
			"childrenIds": []string{fmt.Sprintf("cid-vid-%x", ts)},
			"role":        "user",
			"content":     promptText,
			"user_action": "chat",
			"files":       []map[string]any{},
			"timestamp":   ts,
			"models":      []string{resolvedModel},
			"chat_type":   "t2v",
			"feature_config": map[string]any{
				"output_schema":    "phase",
				"thinking_enabled": false,
				"thinking_format":  "summary",
				"auto_thinking":    true,
				"auto_search":      true,
			},
			"extra":         map[string]any{"meta": map[string]any{"subChatType": "t2v"}},
			"sub_chat_type": "t2v",
			"parent_id":     nil,
		}},
		"timestamp":                ts,
		"size":                     ratio,
		"share_id":                 "",
		"origin_branch_message_id": "",
	}
}
