package environment

import (
	"context"

	"github.com/chaitin/MonkeyCode/monkeyai/backend/license"
)

func (s *Service) WithLicense(reader license.Reader) *Service {
	s.license = reader
	return s
}

func (s *Service) cloudComputerAllowed(ctx context.Context) bool {
	return license.Read(ctx, s.license).AllowCloudComputer
}

func (s *Service) requireCloudComputer(ctx context.Context) error {
	if !s.cloudComputerAllowed(ctx) {
		return &failure{403, "cloud_computer_not_licensed", "当前授权不允许使用云电脑"}
	}
	return nil
}
