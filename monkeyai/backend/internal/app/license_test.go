package app

import (
	"context"
	"testing"

	"github.com/chaitin/MonkeyCode/monkeyai/backend/license"
	"github.com/go-chi/chi/v5"
)

type capabilityHost struct{ value license.Capabilities }

func (h *capabilityHost) RegisterAdmin(chi.Router) {}
func (h *capabilityHost) Capabilities(context.Context) (license.Capabilities, error) {
	return h.value, nil
}

func TestLicenseReaderUsesHostExtension(t *testing.T) {
	if licenseReader(nil) != nil {
		t.Fatal("开源入口不应自动创建私有授权源")
	}
	host := &capabilityHost{license.Capabilities{AllowCustomBranding: true}}
	if licenseReader([]AdminRegistrar{host}) != nil {
		t.Fatal("未装配商业模块不应公开启用能力")
	}
	reader := licenseReader([]AdminRegistrar{nil, host, &moduleStub{}})
	if reader != host {
		t.Fatal("没有接入宿主授权源")
	}
	flags := license.Read(t.Context(), reader)
	if flags.AllowCloudComputer || !flags.AllowCustomBranding {
		t.Fatal("宿主能力读取不一致")
	}
	host.value = license.Capabilities{AllowCloudComputer: true}
	flags = license.Read(t.Context(), reader)
	if !flags.AllowCloudComputer || flags.AllowCustomBranding {
		t.Fatal("运行时能力变更没有生效")
	}
}
