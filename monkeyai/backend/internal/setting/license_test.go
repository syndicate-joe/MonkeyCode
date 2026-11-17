package setting

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chaitin/MonkeyCode/monkeyai/backend/license"
	"github.com/go-chi/chi/v5"
)

type capabilitySource struct {
	value license.Capabilities
	err   error
}

func (s *capabilitySource) Capabilities(context.Context) (license.Capabilities, error) {
	return s.value, s.err
}

func TestBrandingLicenseBlocksEveryWriteAndPreservesConfiguration(t *testing.T) {
	original := json.RawMessage(`{"workspace_name":"原团队","product_name":"原产品","logo_s3_key":"branding/logos/original.png"}`)
	store := &logoStore{memoryStore: memoryStore{records: map[string]Record{"branding": {Key: "branding", Value: original, SchemaVersion: 1}}}}
	storage := &logoStorage{files: map[string][]byte{"branding/logos/original.png": []byte("image")}}
	source := &capabilitySource{value: license.Capabilities{AllowCloudComputer: true}}
	service := NewService(store).WithLogoStorage(storage, "https://example.com").WithLicense(source)
	router := chi.NewRouter()
	service.RegisterAdmin(router)
	for _, tc := range []struct{ method, path, contentType string }{
		{"PUT", "/settings/branding", "application/json"},
		{"PUT", "/settings/branding", "multipart/form-data; boundary=test"},
		{"PUT", "/settings/branding/logo", "multipart/form-data; boundary=test"},
		{"DELETE", "/settings/branding/logo", ""},
	} {
		request := httptest.NewRequest(tc.method, tc.path, strings.NewReader("{}"))
		request.Header.Set("Content-Type", tc.contentType)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != 403 || !strings.Contains(response.Body.String(), "custom_branding_not_licensed") {
			t.Fatalf("%s %s: %d %s", tc.method, tc.path, response.Code, response.Body.String())
		}
	}
	if _, err := service.Put(t.Context(), "branding", original, 1, "admin"); !errors.Is(err, ErrBrandingNotLicensed) {
		t.Fatalf("业务方法绕过授权：%v", err)
	}
	value, err := service.GetValue(t.Context(), "branding")
	if err != nil || strings.Contains(string(value), "原团队") {
		t.Fatal("读取仍使用原品牌")
	}
	config, err := service.AgentConfig(t.Context())
	if err != nil || strings.Contains(string(config.Settings["branding"]), "原团队") {
		t.Fatal("客户端仍收到原品牌")
	}
	if _, _, err := service.OpenBrandingLogo(t.Context()); !errors.Is(err, ErrNotFound) {
		t.Fatal("未授权仍可读取 Logo")
	}
	if string(store.records["branding"].Value) != string(original) || len(storage.files) != 1 {
		t.Fatal("未授权检查破坏了已有配置")
	}
	source.value.AllowCustomBranding = true
	value, err = service.GetValue(t.Context(), "branding")
	if err != nil || string(value) != string(original) {
		t.Fatal("重新授权后配置未恢复")
	}
	body, _, err := service.OpenBrandingLogo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	body.Close()
	source.err = errors.New("unavailable")
	if service.customBranding(t.Context()) {
		t.Fatal("授权查询失败仍放行")
	}
}
