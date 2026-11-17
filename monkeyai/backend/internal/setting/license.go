package setting

import (
	"context"
	"errors"
	"net/http"

	"github.com/chaitin/MonkeyCode/monkeyai/backend/internal/web"
	"github.com/chaitin/MonkeyCode/monkeyai/backend/license"
)

var ErrBrandingNotLicensed = errors.New("当前授权不允许自定义品牌信息")

var defaultBranding = []byte(`{"workspace_name":"Monkey AI","product_name":"MonkeyAI"}`)

func (s *Service) WithLicense(reader license.Reader) *Service {
	s.license = reader
	return s
}

func (s *Service) customBranding(ctx context.Context) bool {
	return license.Read(ctx, s.license).AllowCustomBranding
}

func (s *Service) requireBranding(w http.ResponseWriter, r *http.Request) bool {
	if s.customBranding(r.Context()) {
		return true
	}
	settingJSON(r.Context(), w, http.StatusForbidden, map[string]any{
		"error": web.Body(r.Context(), "custom_branding_not_licensed", ErrBrandingNotLicensed.Error()),
	})
	return false
}
