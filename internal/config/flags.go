package config

import (
	"fmt"
	"strings"
)

// importMapFlag implements flag.Value for repeatable -m pkg=import/path flags
type importMapFlag map[string]string

func (m *importMapFlag) String() string {
	return fmt.Sprintf("%v", map[string]string(*m))
}

func (m *importMapFlag) Set(val string) error {
	parts := strings.SplitN(val, "=", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("expected pkg=import/path, got %q", val)
	}
	if *m == nil {
		*m = make(importMapFlag)
	}
	(*m)[parts[0]] = parts[1]
	return nil
}
