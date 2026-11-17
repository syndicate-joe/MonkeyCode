package environment

import (
	"context"
	"errors"
	"testing"

	"github.com/chaitin/MonkeyCode/monkeyai/backend/internal/resource"
	"github.com/chaitin/MonkeyCode/monkeyai/backend/license"
)

type capabilitySource struct {
	value license.Capabilities
	err   error
}

func (s *capabilitySource) Capabilities(context.Context) (license.Capabilities, error) {
	return s.value, s.err
}

func TestCloudLicenseBlocksAdmissionBeforeSideEffects(t *testing.T) {
	source := &capabilitySource{value: license.Capabilities{AllowCustomBranding: true}}
	service := NewService(nil).WithLicense(source)
	check := func(err error) {
		t.Helper()
		var f *failure
		if !errors.As(err, &f) || f.status != 403 || f.code != "cloud_computer_not_licensed" {
			t.Fatalf("错误=%v", err)
		}
	}
	_, err := service.Create(t.Context(), "", Creation{})
	check(err)
	_, err = service.Change(t.Context(), nil, "", "running", Change{})
	check(err)
	check(service.approveRuntime(t.Context(), "", RuntimeRequest{}))
	_, err = service.UpdateRuntime(t.Context(), "", "", RuntimeUpdate{})
	check(err)
	source.value.AllowCloudComputer = true
	if service.requireCloudComputer(t.Context()) != nil {
		t.Fatal("品牌开关错误地限制了云电脑")
	}
	source.err = errors.New("unavailable")
	check(service.requireCloudComputer(t.Context()))
}

func TestCloudLicenseRevocationRetainsCleanupAndPersonalDevices(t *testing.T) {
	s, created, host, _ := managedEnvironment(t)
	e := created.Environment
	in := runtimeRequest(e)
	if err := s.approveRuntime(t.Context(), host, in); err != nil {
		t.Fatal(err)
	}
	source := &capabilitySource{}
	s.WithLicense(source)
	if allowed, err := s.RuntimeAllowed(t.Context(), nil, e.UserID, in.MachineID); err != nil || allowed {
		t.Fatal("撤销能力后云电脑仍可使用", err)
	}
	if allowed, err := s.RuntimeAllowed(t.Context(), nil, e.UserID, resource.ID()); err != nil || !allowed {
		t.Fatal("个人电脑被云电脑授权误伤", err)
	}
	source.value.AllowCloudComputer = true
	if allowed, err := s.RuntimeAllowed(t.Context(), nil, e.UserID, in.MachineID); err != nil || !allowed {
		t.Fatal("重新授权后无法恢复", err)
	}
	source.value.AllowCloudComputer = false
	syncState(t, s, host, observe(e, "running"))
	stopped, err := s.Change(t.Context(), &e.UserID, e.ID, "stopped", Change{e.Generation, "license-stop"})
	if err != nil {
		t.Fatal("未授权无法停止环境", err)
	}
	syncState(t, s, host, observe(stopped.Environment, "stopped"))
	if _, err = s.Change(t.Context(), &e.UserID, e.ID, "deleted", Change{stopped.Environment.Generation, "license-delete"}); err != nil {
		t.Fatal("未授权无法删除环境", err)
	}
}
