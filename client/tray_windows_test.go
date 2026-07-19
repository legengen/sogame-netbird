//go:build windows && amd64

package main

import "testing"

func TestTraySystemProceduresResolve(t *testing.T) {
	for name, procedure := range map[string]interface{ Find() error }{
		"GetModuleHandleW": getModuleHandle,
		"PostMessageW":     postMessage,
	} {
		if err := procedure.Find(); err != nil {
			t.Fatalf("resolve %s: %v", name, err)
		}
	}
}
