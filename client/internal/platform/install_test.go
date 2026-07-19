package platform

import (
	"context"
	"errors"
	"reflect"
	"testing"

	releasebuild "github.com/legengen/sogame-netbird/client/build"
)

type fakeArtifactCheck struct {
	err    error
	called bool
}

func (f *fakeArtifactCheck) Verify(context.Context, string, releasebuild.WindowsArtifact) error {
	f.called = true
	return f.err
}

type fakeMSIRunner struct {
	called bool
	action MSIAction
}

func (f *fakeMSIRunner) Run(_ context.Context, action MSIAction, _, _ string) error {
	f.called = true
	f.action = action
	return nil
}

func TestPrivilegedInstallerVerifiesBeforeExecution(t *testing.T) {
	artifact := `C:\release\netbird.msi`
	logPath := `C:\release\install.log`
	check := &fakeArtifactCheck{}
	runner := &fakeMSIRunner{}
	installer := NewPrivilegedInstaller(check, runner)

	if err := installer.Execute(context.Background(), MSIInstall, artifact, logPath, releasebuild.WindowsArtifact{}); err != nil {
		t.Fatal(err)
	}
	if !check.called || !runner.called || runner.action != MSIInstall {
		t.Fatalf("unexpected call order result: check=%v runner=%+v", check.called, runner)
	}
}

func TestPrivilegedInstallerStopsOnVerificationFailure(t *testing.T) {
	check := &fakeArtifactCheck{err: ErrArtifactDigest}
	runner := &fakeMSIRunner{}
	err := NewPrivilegedInstaller(check, runner).Execute(
		context.Background(), MSIRepair, `C:\release\netbird.msi`, `C:\release\repair.log`, releasebuild.WindowsArtifact{},
	)
	if !errors.Is(err, ErrArtifactDigest) || runner.called {
		t.Fatalf("error=%v runner called=%v", err, runner.called)
	}
}

func TestBuildMSIArguments(t *testing.T) {
	install, err := BuildMSIArguments(MSIInstall, `C:\release\netbird.msi`, `C:\logs\install.log`)
	if err != nil {
		t.Fatal(err)
	}
	wantInstall := []string{"/i", `C:\release\netbird.msi`, "/quiet", "/qn", "/norestart", "/l*v", `C:\logs\install.log`, "AUTOSTART=0"}
	if !reflect.DeepEqual(install, wantInstall) {
		t.Fatalf("install args=%q", install)
	}
	repair, err := BuildMSIArguments(MSIRepair, `C:\release\netbird.msi`, `C:\logs\repair.log`)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(repair[len(repair)-2:], []string{"REINSTALL=ALL", "REINSTALLMODE=vomus"}) {
		t.Fatalf("repair args=%q", repair)
	}
	if _, err := BuildMSIArguments("arbitrary", "x", "y"); !errors.Is(err, ErrUnsupportedAction) {
		t.Fatalf("unexpected action error=%v", err)
	}
}
