package dotfiles

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var updateCommandRunner = func(exe string, args ...string) *exec.Cmd {
	cmd := exec.Command(exe, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

var updateCommandOutput = func(exe string, args ...string) ([]byte, error) {
	return exec.Command(exe, args...).Output()
}

var updateGetConfig = GetConfig
var updatePackageSync = syncPackageManagers

func Update(repoInput string) error {
	repo, err := absolutePathToRepo(repoInput)
	if err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if repo.String() != cwd {
		if err := os.Chdir(repo.String()); err != nil {
			return err
		}
		defer os.Chdir(cwd)
	}

	gitExe := os.Getenv("DOTFILES_GIT_COMMAND")
	if gitExe == "" {
		gitExe = "git"
	}

	errs := []error{}

	fmt.Println("Update: checking repository status...")

	dirty, err := hasLocalChanges(gitExe)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: git update failed: %s\n", err.Error())
		errs = append(errs, fmt.Errorf("git update failed: %w", err))
	} else if dirty {
		fmt.Println("Skip: local changes detected in dotfiles repo. Commit/stash/discard changes, then run update again.")
	} else {
		fmt.Println("Update: pulling latest changes...")
		if err := updateCommandRunner(gitExe, "pull").Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: git update failed: %s\n", err.Error())
			errs = append(errs, fmt.Errorf("git update failed: %w", err))
		}
	}

	fmt.Println("Update: syncing package managers...")

	cfg, err := updateGetConfig(repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: package sync failed: %s\n", err.Error())
		errs = append(errs, fmt.Errorf("package sync failed: %w", err))
	} else {
		if _, err := updatePackageSync(cfg.PackageManagers, false); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: package sync failed: %s\n", err.Error())
			errs = append(errs, fmt.Errorf("package sync failed: %w", err))
		}
	}

	return joinErrors(errs)
}

func hasLocalChanges(gitExe string) (bool, error) {
	out, err := updateCommandOutput(gitExe, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func joinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}

	msgs := make([]string, 0, len(errs))
	for _, err := range errs {
		msgs = append(msgs, err.Error())
	}
	return fmt.Errorf(strings.Join(msgs, "; "))
}
