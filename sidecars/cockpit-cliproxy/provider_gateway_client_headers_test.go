package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v7/sdk/translator"
)

// providerClientHeaderTransport 在实际网关发送边界检查请求，避免连接外部服务。
type providerClientHeaderTransport func(*http.Request) (*http.Response, error)

// RoundTrip 执行测试提供的请求检查并返回模拟上游响应。
func (f providerClientHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// TestProviderGatewayClientHeaders 验证两种上游协议的客户端标识透传和凭据隔离。
func TestProviderGatewayClientHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalClient := http.DefaultClient
	t.Cleanup(func() { http.DefaultClient = originalClient })
	for _, wireAPI := range []string{"responses", "chat_completions"} {
		for _, tc := range []struct {
			name       string   // 标识当前域名及客户端头组合。
			baseURL    string   // 配置的供应商地址，不会建立真实连接。
			clientKeys []string // 实际客户端提供的标识头。
			forward    bool     // 是否允许向该域名透传。
		}{
			{"agentrouter", "https://agentrouter.org/v1", []string{"User-Agent", "Originator"}, true},
			{"case_and_port", "https://AGENTROUTER.ORG:443/v1", []string{"User-Agent", "Originator"}, true},
			{"missing", "https://agentrouter.org/v1", nil, true},
			{"user_agent_only", "https://agentrouter.org/v1", []string{"User-Agent"}, true},
			{"originator_only", "https://agentrouter.org/v1", []string{"Originator"}, true},
			{"other_provider", "https://api.openai.com/v1", []string{"User-Agent", "Originator"}, false},
			{"suffix", "https://agentrouter.org.example.com/v1", []string{"User-Agent", "Originator"}, false},
			{"subdomain", "https://other.agentrouter.org/v1", []string{"User-Agent", "Originator"}, false},
			{"userinfo", "https://agentrouter.org@example.com/v1", []string{"User-Agent", "Originator"}, false},
		} {
			t.Run(wireAPI+"/"+tc.name, func(t *testing.T) {
				body := `{"model":"deepseek-v4-flash","input":"hello","stream":false}`
				incoming := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
				incoming.Header.Set("Authorization", "Bearer downstream-only-secret")
				incoming.Header.Set("Chatgpt-Account-Id", "downstream-only-account")
				incoming.Header.Set("Cookie", "session=downstream-only-cookie")
				incoming.Header.Set("X-Api-Key", "downstream-only-api-key")
				incoming.Header.Set("X-Client-Request-Id", "client-request-id")
				incoming.Header.Set("X-Agtools-Trace", "diagnostic-trace")
				for _, key := range tc.clientKeys {
					incoming.Header.Set(key, "actual-client-"+key)
				}
				calls := 0
				http.DefaultClient = &http.Client{Transport: providerClientHeaderTransport(func(req *http.Request) (*http.Response, error) {
					calls++
					for _, key := range []string{"User-Agent", "Originator"} {
						want := ""
						if tc.forward {
							want = incoming.Header.Get(key)
						}
						if got := req.Header.Get(key); got != want {
							t.Errorf("%s = %q, want %q", key, got, want)
						}
					}
					if got := req.Header.Get("Authorization"); got != "Bearer provider-only-key" {
						t.Errorf("provider authorization replaced: %q", got)
					}
					for _, key := range []string{"Chatgpt-Account-Id", "Cookie", "X-Api-Key"} {
						if req.Header.Get(key) != "" {
							t.Errorf("downstream credential leaked through %s", key)
						}
					}
					if req.Header.Get("X-Client-Request-Id") != "client-request-id" || req.Header.Get("X-Agtools-Trace") != "diagnostic-trace" {
						t.Error("existing diagnostic headers were lost")
					}
					payload := `{"id":"resp-test","object":"response","status":"completed","output":[]}`
					wantPath := "/v1/responses"
					if wireAPI == "chat_completions" {
						wantPath = "/v1/chat/completions"
						payload = `{"id":"chat-test","object":"chat.completion","model":"deepseek-v4-flash","choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}]}`
					}
					if req.URL.Path != wantPath {
						t.Errorf("upstream path = %q, want %q", req.URL.Path, wantPath)
					}
					return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(payload)), Request: req}, nil
				})}
				gateway := &providerGatewaySpec{
					BaseURL: tc.baseURL, APIKey: "provider-only-key", WireAPI: wireAPI,
					UpstreamModel: "deepseek-v4-flash", UpstreamModels: []string{"deepseek-v4-flash"},
				}
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = incoming
				(&relayServer{}).handleProviderGatewayRequest(c, gateway, []byte(body), "deepseek-v4-flash", sdktranslator.FormatOpenAIResponse, "")
				if calls != 1 || w.Code != http.StatusOK {
					t.Fatalf("upstream calls = %d, status = %d, body = %s", calls, w.Code, w.Body.String())
				}
			})
		}
	}
}

// TestProviderGatewayClientHeadersInvalidInput 确保无头或无效地址不会新增标识或触发崩溃。
func TestProviderGatewayClientHeadersInvalidInput(t *testing.T) {
	src := http.Header{"User-Agent": {"actual-client"}, "Originator": {"actual-originator"}}
	for _, baseURL := range []string{"", "://invalid", "https://[invalid"} {
		dst := make(http.Header)
		copyProviderGatewayCodexClientHeaders(dst, src, baseURL)
		if len(dst) != 0 {
			t.Errorf("invalid URL %q copied headers: %v", baseURL, dst)
		}
	}
	copyProviderGatewayCodexClientHeaders(nil, src, "https://agentrouter.org")
	copyProviderGatewayCodexClientHeaders(make(http.Header), nil, "https://agentrouter.org")
}
