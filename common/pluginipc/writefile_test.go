package pluginipc

import "os"

func writeFileOS(path string, b []byte) error {
	return os.WriteFile(path, b, 0o600)
}
