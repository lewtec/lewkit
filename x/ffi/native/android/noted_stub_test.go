//go:build !android

package android

import "testing"

func TestNotedVMIsFalseOffAndroid(t *testing.T) {
	if NotedVM() {
		t.Fatal("host build noted a Java VM")
	}
}
