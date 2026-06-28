package acp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// translateInput مدخلات الترجمة
type translateInput struct {
	Text   string `json:"text"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// executeInput مدخلات التنفيذ الآمن
type executeInput struct {
	Action string          `json:"action"`
	Params json.RawMessage `json:"params,omitempty"`
}

// registerBuiltinTasks يسجّل المهام المدمجة
func registerBuiltinTasks(r *Router) {
	r.Register(TaskPing, handlePing)
	r.Register(TaskEcho, handleEcho)
	r.Register(TaskTranslate, handleTranslate)
	r.Register(TaskExecute, handleExecute)
}

func handlePing(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]string{"status": "pong"})
}

func handleEcho(_ context.Context, input json.RawMessage) (json.RawMessage, error) {
	if len(input) == 0 {
		return json.Marshal(map[string]string{"echo": ""})
	}
	var data map[string]interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return json.Marshal(map[string]string{"echo": string(input)})
	}
	text, _ := data["text"].(string)
	return json.Marshal(map[string]string{"echo": text})
}

func handleTranslate(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	var in translateInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("مدخلات translate غير صالحة")
	}
	if in.Text == "" {
		return nil, fmt.Errorf("text مطلوب")
	}
	if in.Source == "" {
		in.Source = "en"
	}
	if in.Target == "" {
		in.Target = "ar"
	}

	apiURL := fmt.Sprintf("https://api.mymemory.translated.net/get?q=%s&langpair=%s|%s",
		url.QueryEscape(in.Text), url.QueryEscape(in.Source), url.QueryEscape(in.Target))

	var translatedText string
	var engine = "mymemory.translated.net"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err == nil {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var result struct {
					ResponseData struct {
						TranslatedText string `json:"translatedText"`
					} `json:"responseData"`
				}
				if json.NewDecoder(resp.Body).Decode(&result) == nil && result.ResponseData.TranslatedText != "" {
					translatedText = result.ResponseData.TranslatedText
				}
			}
		}
	}

	if translatedText == "" {
		translatedText = fmt.Sprintf("[%s→%s] %s", in.Source, in.Target, in.Text)
		engine = "stub/v1-fallback"
	}

	result := map[string]interface{}{
		"translated": translatedText,
		"source":     in.Source,
		"target":     in.Target,
		"engine":     engine,
	}
	return json.Marshal(result)
}

// إجراءات task.execute المسموحة فقط
var allowedExecuteActions = map[string]bool{
	"get_time": true,
	"hash":     true,
	"upper":    true,
	"lower":    true,
}

func handleExecute(_ context.Context, input json.RawMessage) (json.RawMessage, error) {
	var in executeInput
	if err := json.Unmarshal(input, &in); err != nil {
		return nil, fmt.Errorf("مدخلات execute غير صالحة")
	}
	if !allowedExecuteActions[in.Action] {
		return nil, fmt.Errorf("إجراء غير مسموح: %s", in.Action)
	}

	switch in.Action {
	case "get_time":
		return json.Marshal(map[string]interface{}{
			"unix": time.Now().Unix(),
			"iso":  time.Now().UTC().Format(time.RFC3339),
		})
	case "hash":
		var p struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(in.Params, &p); err != nil {
			return nil, fmt.Errorf("failed to parse params: %w", err)
		}
		h := sha256.Sum256([]byte(p.Text))
		return json.Marshal(map[string]string{
			"hash": hex.EncodeToString(h[:]),
		})
	case "upper":
		var p struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(in.Params, &p); err != nil {
			return nil, fmt.Errorf("failed to parse params: %w", err)
		}
		return json.Marshal(map[string]string{"result": strings.ToUpper(p.Text)})
	case "lower":
		var p struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(in.Params, &p); err != nil {
			return nil, fmt.Errorf("failed to parse params: %w", err)
		}
		return json.Marshal(map[string]string{"result": strings.ToLower(p.Text)})
	default:
		return nil, fmt.Errorf("إجراء غير معروف")
	}
}
