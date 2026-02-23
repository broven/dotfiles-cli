package dotfiles

import (
	"fmt"
	"os"
	"os/exec"
)

var lookPath = exec.LookPath

var commandRunner = func(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func hasPackageManagerConfig(pm PackageManagers) bool {
	return len(pm.NPM) > 0 || len(pm.Homebrew.Tap) > 0 || len(pm.Homebrew.Formula) > 0 || len(pm.Homebrew.Cask) > 0
}

func syncPackageManagers(pm PackageManagers, dry bool) (bool, error) {
	if !hasPackageManagerConfig(pm) {
		return false, nil
	}

	if err := syncNPM(pm.NPM, dry); err != nil {
		return true, err
	}
	if err := syncHomebrew(pm.Homebrew, dry); err != nil {
		return true, err
	}

	return true, nil
}

func syncNPM(packages []string, dry bool) error {
	if len(packages) == 0 {
		return nil
	}

	if _, err := lookPath("npm"); err != nil {
		fmt.Println("Skip: 'npm' command was not found. npm packages were not processed.")
		return nil
	}

	for _, pkg := range packages {
		args := []string{"install", "-g", pkg}
		if dry {
			fmt.Printf("DryRun: npm %s\n", joinArgs(args))
			continue
		}
		fmt.Printf("Run: npm %s\n", joinArgs(args))
		if err := commandRunner("npm", args...).Run(); err != nil {
			return fmt.Errorf("failed to install/update npm package '%s': %w", pkg, err)
		}
	}
	return nil
}

func syncHomebrew(h HomebrewPackages, dry bool) error {
	if len(h.Tap) == 0 && len(h.Formula) == 0 && len(h.Cask) == 0 {
		return nil
	}

	if _, err := lookPath("brew"); err != nil {
		fmt.Println("Skip: 'brew' command was not found. homebrew packages were not processed.")
		return nil
	}

	for _, tap := range h.Tap {
		args := []string{"tap", tap}
		if dry {
			fmt.Printf("DryRun: brew %s\n", joinArgs(args))
			continue
		}
		fmt.Printf("Run: brew %s\n", joinArgs(args))
		if err := commandRunner("brew", args...).Run(); err != nil {
			return fmt.Errorf("failed to tap homebrew repository '%s': %w", tap, err)
		}
	}

	for _, formula := range h.Formula {
		args := []string{"install", formula}
		if dry {
			fmt.Printf("DryRun: brew %s\n", joinArgs(args))
			continue
		}
		fmt.Printf("Run: brew %s\n", joinArgs(args))
		if err := commandRunner("brew", args...).Run(); err != nil {
			return fmt.Errorf("failed to install/update homebrew formula '%s': %w", formula, err)
		}
	}

	for _, cask := range h.Cask {
		args := []string{"install", "--cask", cask}
		if dry {
			fmt.Printf("DryRun: brew %s\n", joinArgs(args))
			continue
		}
		fmt.Printf("Run: brew %s\n", joinArgs(args))
		if err := commandRunner("brew", args...).Run(); err != nil {
			return fmt.Errorf("failed to install/update homebrew cask '%s': %w", cask, err)
		}
	}

	return nil
}

func joinArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	s := args[0]
	for _, arg := range args[1:] {
		s += " " + arg
	}
	return s
}
