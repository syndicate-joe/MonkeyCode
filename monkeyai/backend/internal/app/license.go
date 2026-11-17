package app

import "github.com/chaitin/MonkeyCode/monkeyai/backend/license"

func licenseReader(registrars []AdminRegistrar) license.Reader {
	for _, registrar := range registrars {
		if reader, ok := registrar.(license.Reader); ok {
			return reader
		}
	}
	return nil
}
