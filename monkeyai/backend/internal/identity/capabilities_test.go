package identity

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chaitin/MonkeyCode/monkeyai/backend/license"
)

type capabilitySource struct {
	value license.Capabilities
	err   error
}

func (s *capabilitySource) Capabilities(context.Context) (license.Capabilities, error) {
	return s.value, s.err
}

type licensedLogo struct{ opens int }

func (*licensedLogo) BrandingLogoURL(context.Context) (string, error) { return "/custom.png", nil }
func (s *licensedLogo) OpenBrandingLogo(context.Context) (io.ReadCloser, string, error) {
	s.opens++
	return io.NopCloser(strings.NewReader("image")), "version", nil
}

func TestPublicCapabilitiesAndEffectiveBranding(t *testing.T) {
	source := &capabilitySource{}
	logo := new(licensedLogo)
	settings := authenticationStub{json.RawMessage(`{"workspace_name":"自定义团队","product_name":"自定义产品","allow_custom_branding":true}`)}
	service := NewService(nil, settings, "").WithLicense(source).WithBrandingLogo(logo)
	for _, enabled := range []bool{false, true, false, true} {
		source.value = license.Capabilities{AllowCloudComputer: !enabled, AllowCustomBranding: enabled}
		for _, path := range []string{"/branding", "/capabilities", "/branding/logo"} {
			response := httptest.NewRecorder()
			request := httptest.NewRequest("GET", path, nil)
			request.Header.Set("If-None-Match", `"version"`)
			service.AuthRouter().ServeHTTP(response, request)
			if path == "/branding/logo" {
				want := 404
				if enabled {
					want = 304
				}
				if response.Code != want {
					t.Fatalf("Logo 状态=%d，期望=%d", response.Code, want)
				}
				continue
			}
			var got map[string]any
			if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &got) != nil {
				t.Fatalf("响应=%s", response.Body.String())
			}
			if path == "/capabilities" {
				if len(got) != 2 || got["allow_cloud_computer"] != !enabled || got["allow_custom_branding"] != enabled {
					t.Fatalf("能力=%+v", got)
				}
			} else if enabled {
				if got["workspace_name"] != "自定义团队" || got["product_name"] != "自定义产品" || got["logo_url"] != "/custom.png" {
					t.Fatalf("品牌=%+v", got)
				}
			} else if got["workspace_name"] != defaultWorkspaceName || got["product_name"] != defaultProductName || got["logo_url"] != "" {
				t.Fatalf("未授权仍返回自定义品牌=%+v", got)
			}
		}
	}
	if logo.opens != 2 {
		t.Fatalf("未授权仍读取了 Logo：%d", logo.opens)
	}
	source.err = errors.New("private failure")
	response := httptest.NewRecorder()
	service.AuthRouter().ServeHTTP(response, httptest.NewRequest("GET", "/capabilities", nil))
	var got license.Capabilities
	if json.Unmarshal(response.Body.Bytes(), &got) != nil || got.AllowCloudComputer || got.AllowCustomBranding || strings.Contains(response.Body.String(), "private failure") {
		t.Fatal("授权读取错误未安全关闭")
	}
}
