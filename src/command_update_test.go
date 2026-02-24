package dotfiles

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/rhysd/abspath"
)

func withUpdateStubs(t *testing.T) {
	t.Helper()

	restoreOutput := updateCommandOutput
	restoreRunner := updateCommandRunner
	restoreGetConfig := updateGetConfig
	restorePackageSync := updatePackageSync

	t.Cleanup(func() {
		updateCommandOutput = restoreOutput
		updateCommandRunner = restoreRunner
		updateGetConfig = restoreGetConfig
		updatePackageSync = restorePackageSync
	})
}

func TestUpdateErrorCase(t *testing.T) {
	if err := Update("unknown_repo"); err == nil {
		t.Fatalf("it should raise an error when unknown repository specified")
	}
}

func TestUpdateSkipsGitPullWhenLocalChangesAndSyncsPackages(t *testing.T) {
	withUpdateStubs(t)

	repo := t.TempDir()
	packageSynced := false

	updateCommandOutput = func(_ string, args ...string) ([]byte, error) {
		if len(args) >= 1 && args[0] == "status" {
			return []byte(" M mappings.yaml\n"), nil
		}
		return nil, nil
	}
	updateCommandRunner = func(_ string, args ...string) *exec.Cmd {
		if len(args) >= 1 && args[0] == "pull" {
			t.Fatalf("git pull must not run when local changes exist")
		}
		return exec.Command("true")
	}
	updateGetConfig = func(_ abspath.AbsPath) (*Config, error) {
		return &Config{PackageManagers: PackageManagers{NPM: []string{"typescript"}}}, nil
	}
	updatePackageSync = func(pm PackageManagers, _ bool) (bool, error) {
		packageSynced = true
		if len(pm.NPM) != 1 || pm.NPM[0] != "typescript" {
			t.Fatalf("unexpected package manager configuration: %+v", pm)
		}
		return true, nil
	}

	if err := Update(repo); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !packageSynced {
		t.Fatal("package sync phase must run even when git phase is skipped")
	}
}

func TestUpdateContinuesPackageSyncWhenGitUpdateFails(t *testing.T) {
	withUpdateStubs(t)

	repo := t.TempDir()
	packageSynced := false

	updateCommandOutput = func(_ string, args ...string) ([]byte, error) {
		if len(args) >= 1 && args[0] == "status" {
			return []byte{}, nil
		}
		return nil, nil
	}
	updateCommandRunner = func(_ string, args ...string) *exec.Cmd {
		if len(args) >= 1 && args[0] == "pull" {
			return exec.Command("false")
		}
		return exec.Command("true")
	}
	updateGetConfig = func(_ abspath.AbsPath) (*Config, error) {
		return &Config{}, nil
	}
	updatePackageSync = func(_ PackageManagers, _ bool) (bool, error) {
		packageSynced = true
		return false, nil
	}

	err := Update(repo)
	if err == nil {
		t.Fatal("expected error when git update phase fails")
	}
	if !strings.Contains(err.Error(), "git update failed") {
		t.Fatalf("expected git update error but got: %s", err.Error())
	}
	if !packageSynced {
		t.Fatal("package sync phase must run even when git update fails")
	}
}

func TestUpdateUsesDOTFILESGitCommand(t *testing.T) {
	withUpdateStubs(t)

	repo := t.TempDir()
	const expected = "internal-git"

	saved := os.Getenv("DOTFILES_GIT_COMMAND")
	t.Cleanup(func() {
		os.Setenv("DOTFILES_GIT_COMMAND", saved)
	})
	if err := os.Setenv("DOTFILES_GIT_COMMAND", expected); err != nil {
		t.Fatalf("failed to set DOTFILES_GIT_COMMAND: %s", err)
	}

	var seen []string
	updateCommandOutput = func(exe string, args ...string) ([]byte, error) {
		seen = append(seen, exe+" "+strings.Join(args, " "))
		return []byte{}, nil
	}
	updateCommandRunner = func(exe string, args ...string) *exec.Cmd {
		seen = append(seen, exe+" "+strings.Join(args, " "))
		return exec.Command("true")
	}
	updateGetConfig = func(_ abspath.AbsPath) (*Config, error) {
		return &Config{}, nil
	}
	updatePackageSync = func(_ PackageManagers, _ bool) (bool, error) {
		return false, nil
	}

	if err := Update(repo); err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if len(seen) != 2 {
		t.Fatalf("expected two git invocations but got: %v", seen)
	}
	if !strings.HasPrefix(seen[0], expected+" status") {
		t.Fatalf("status check must use DOTFILES_GIT_COMMAND: %v", seen)
	}
	if !strings.HasPrefix(seen[1], expected+" pull") {
		t.Fatalf("pull must use DOTFILES_GIT_COMMAND: %v", seen)
	}
}

func TestUpdateReturnsCombinedErrorWhenGitAndPackageFail(t *testing.T) {
	withUpdateStubs(t)

	repo := t.TempDir()

	updateCommandOutput = func(_ string, args ...string) ([]byte, error) {
		if len(args) >= 1 && args[0] == "status" {
			return []byte{}, nil
		}
		return nil, nil
	}
	updateCommandRunner = func(_ string, args ...string) *exec.Cmd {
		if len(args) >= 1 && args[0] == "pull" {
			return exec.Command("false")
		}
		return exec.Command("true")
	}
	updateGetConfig = func(_ abspath.AbsPath) (*Config, error) {
		return &Config{}, nil
	}
	updatePackageSync = func(_ PackageManagers, _ bool) (bool, error) {
		return false, errors.New("package sync failed")
	}

	err := Update(repo)
	if err == nil {
		t.Fatal("expected combined error but got nil")
	}
	if !strings.Contains(err.Error(), "git update failed") {
		t.Fatalf("missing git update error: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "package sync failed") {
		t.Fatalf("missing package sync error: %s", err.Error())
	}
}
