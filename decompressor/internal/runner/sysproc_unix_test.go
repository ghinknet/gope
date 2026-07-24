//go:build !windows

package runner

import (
	"os/exec"
	"testing"
)

func TestApplySysProcAttrKeepsForegroundProcessGroup(t *testing.T) {
	cmd := exec.Command("/bin/sh", "-c", "true")
	applySysProcAttr(cmd)
	if cmd.SysProcAttr != nil && cmd.SysProcAttr.Setpgid {
		t.Fatal("child must remain in the foreground process group")
	}
}
