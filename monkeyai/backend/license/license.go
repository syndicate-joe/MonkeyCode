package license

import "context"

type Capabilities struct {
	AllowCloudComputer  bool `json:"allow_cloud_computer"`
	AllowCustomBranding bool `json:"allow_custom_branding"`
}

type Reader interface {
	Capabilities(context.Context) (Capabilities, error)
}

func Read(ctx context.Context, reader Reader) Capabilities {
	// 未接入私有授权的开源入口保留原有行为；私有入口始终注入 Reader。
	if reader == nil {
		return Capabilities{AllowCloudComputer: true, AllowCustomBranding: true}
	}
	capabilities, err := reader.Capabilities(ctx)
	if err != nil {
		return Capabilities{}
	}
	return capabilities
}
