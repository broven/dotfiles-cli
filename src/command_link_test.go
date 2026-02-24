package dotfiles

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"
)

func TestLinkAll(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	distConf := path.Join(cwd, "_dist.conf")
	dir := path.Join(cwd, ".dotfiles")
	if err := os.MkdirAll(dir, os.ModePerm|os.ModeDir); err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	f, err := os.OpenFile(path.Join(dir, "mappings.yaml"), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}

	_, err = f.WriteString(`
link:
  _source.conf: "` + distConf + `"
`)
	if err != nil {
		panic(err)
	}
	f.Close()

	source := path.Join(cwd, "_source.conf")
	g, err := os.OpenFile(source, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	defer func() {
		g.Close()
		os.Remove(source)
	}()
	_, err = g.WriteString("this file is for test")
	if err != nil {
		panic(err)
	}

	if err := Link("", nil, false); err != nil {
		t.Error(err)
	}
	defer os.Remove("_dist.conf")
}

func TestLinkSome(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	distConf := path.Join(cwd, "_dist.conf")
	dir := path.Join(cwd, ".dotfiles")
	if err := os.MkdirAll(dir, os.ModePerm|os.ModeDir); err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	f, err := os.OpenFile(path.Join(dir, "mappings.yaml"), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		os.RemoveAll(dir)
		panic(err)
	}
	defer f.Close()

	_, err = f.WriteString(`
link:
  _source.conf: "` + distConf + `"
  _tmp.conf: /path/to/somewhere
`)
	if err != nil {
		panic(err)
	}

	source := path.Join(cwd, "_source.conf")
	g, err := os.OpenFile(source, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	defer func() {
		g.Close()
		os.Remove(source)
	}()
	_, err = g.WriteString("this file is for test")
	if err != nil {
		panic(err)
	}

	if err := Link("", []string{"_source.conf"}, false); err != nil {
		t.Error(err)
	}
	defer os.Remove("_dist.conf")
}

func TestLinkConfigDirDoesNotExist(t *testing.T) {
	if err := Link("", nil, false); err != nil {
		if _, ok := err.(*NothingLinkedError); !ok {
			t.Errorf("Non-existtence of .dotfiles directory does not cause an error: %s", err.Error())
		}
	}
}

func TestLinkAllWithPartialLink(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	dir := path.Join(cwd, ".dotfiles")
	if err := os.MkdirAll(dir, os.ModePerm|os.ModeDir); err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	f, err := os.OpenFile(path.Join(dir, "mappings.yaml"), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	_, err = f.WriteString(`
partial_link:
  _zsh: "` + path.Join(cwd, "_home") + `"
`)
	if err != nil {
		panic(err)
	}
	f.Close()

	if err := os.MkdirAll(path.Join(cwd, "_zsh"), os.ModePerm|os.ModeDir); err != nil {
		panic(err)
	}
	defer os.RemoveAll(path.Join(cwd, "_zsh"))

	files := []string{".zshrc", ".zprofile"}
	for _, name := range files {
		p := filepath.Join(cwd, "_zsh", name)
		file, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			panic(err)
		}
		_, err = file.WriteString("test")
		if err != nil {
			file.Close()
			panic(err)
		}
		file.Close()
	}
	defer os.RemoveAll(path.Join(cwd, "_home"))

	if err := Link("", nil, false); err != nil {
		t.Fatal(err)
	}

	for _, name := range files {
		dst := filepath.Join(cwd, "_home", name)
		info, err := os.Lstat(dst)
		if err != nil {
			t.Fatalf("Expected %s to be linked: %s", dst, err.Error())
		}
		if info.Mode()&os.ModeSymlink != os.ModeSymlink {
			t.Fatalf("Expected %s to be a symlink", dst)
		}
		actual, err := os.Readlink(dst)
		if err != nil {
			t.Fatal(err)
		}
		expected := filepath.Join(cwd, "_zsh", name)
		if actual != expected {
			t.Fatalf("Expected symlink %s -> %s but got %s", dst, expected, actual)
		}
	}
}

func TestLinkSpecifiedRepoDoesNotExist(t *testing.T) {
	if err := Link("unknown_directory", nil, false); err == nil {
		t.Errorf("Should make an error for unknown dotfiles repository")
	}

	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	p := path.Join(cwd, "_dummy_file")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	defer func() {
		f.Close()
		os.Remove(p)
	}()

	_, err = f.WriteString("dummy file")
	if err != nil {
		panic(err)
	}

	if err := Link("_dummy_file", nil, false); err == nil {
		t.Errorf("Should make an error when repository is actually a file")
	}
}

func TestLinkIgnoresPackageManagers(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	restoreLookPath := lookPath
	restoreCommandRunner := commandRunner
	defer func() {
		lookPath = restoreLookPath
		commandRunner = restoreCommandRunner
	}()

	lookPath = func(file string) (string, error) {
		panic("lookPath must not be called from Link")
	}
	commandRunner = func(name string, args ...string) *exec.Cmd {
		panic("commandRunner must not be called from Link")
	}

	distConf := path.Join(cwd, "_dist.conf")
	dir := path.Join(cwd, ".dotfiles")
	if err := os.MkdirAll(dir, os.ModePerm|os.ModeDir); err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	f, err := os.OpenFile(path.Join(dir, "mappings.yaml"), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	_, err = f.WriteString(`
link:
  _source.conf: "` + distConf + `"
npm:
  - typescript
homebrew:
  formula:
    - wget
`)
	if err != nil {
		panic(err)
	}
	f.Close()

	source := path.Join(cwd, "_source.conf")
	g, err := os.OpenFile(source, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	defer func() {
		g.Close()
		os.Remove(source)
	}()
	_, err = g.WriteString("this file is for test")
	if err != nil {
		panic(err)
	}

	if err := Link("", nil, false); err != nil {
		t.Fatal(err)
	}
	defer os.Remove("_dist.conf")
}

func TestLinkPackageManagersOnly(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	dir := path.Join(cwd, ".dotfiles")
	if err := os.MkdirAll(dir, os.ModePerm|os.ModeDir); err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	f, err := os.OpenFile(path.Join(dir, "mappings.yaml"), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	_, err = f.WriteString(`
npm:
  - typescript
homebrew:
  - ripgrep
`)
	if err != nil {
		panic(err)
	}
	f.Close()

	if err := Link("", nil, false); err == nil {
		t.Fatalf("link with package manager only configuration must fail due to nothing linked")
	} else if _, ok := err.(*NothingLinkedError); !ok {
		t.Fatalf("expected NothingLinkedError but got: %s", err.Error())
	}
}

func TestLinkRelinkEnabledRecreatesExistingTarget(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	distConf := path.Join(cwd, "_dist.conf")
	dir := path.Join(cwd, ".dotfiles")
	if err := os.MkdirAll(dir, os.ModePerm|os.ModeDir); err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	f, err := os.OpenFile(path.Join(dir, "mappings.yaml"), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	_, err = f.WriteString(`
relink: true
link:
  _source.conf: "` + distConf + `"
`)
	if err != nil {
		panic(err)
	}
	f.Close()

	source := path.Join(cwd, "_source.conf")
	g, err := os.OpenFile(source, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	defer func() {
		g.Close()
		os.Remove(source)
	}()
	_, err = g.WriteString("this file is for test")
	if err != nil {
		panic(err)
	}

	oldSource := path.Join(cwd, "_old_source.conf")
	old, err := os.OpenFile(oldSource, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	old.Close()
	defer os.Remove(oldSource)

	if err := os.Symlink(oldSource, distConf); err != nil {
		panic(err)
	}
	defer os.Remove(distConf)

	if err := Link("", nil, false); err != nil {
		t.Fatal(err)
	}

	actual, err := os.Readlink(distConf)
	if err != nil {
		t.Fatal(err)
	}
	if actual != source {
		t.Fatalf("Expected relinked destination to point to %s but got %s", source, actual)
	}
}
