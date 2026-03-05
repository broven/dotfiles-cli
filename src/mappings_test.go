package dotfiles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rhysd/abspath"
)

func getcwd() abspath.AbsPath {
	cwd, err := abspath.Getwd()
	if err != nil {
		panic(err)
	}
	return cwd
}

func createTestDir() string {
	dir := "_test_config"
	if err := os.MkdirAll(dir, os.ModeDir|os.ModePerm); err != nil {
		panic(err)
	}
	return dir
}

func createTestMappingFile(fname, contents string) string {
	dir := createTestDir()
	contents = strings.ReplaceAll(contents, "\t", "  ")

	f, err := os.OpenFile(getcwd().Join(dir, fname).String(), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		os.RemoveAll(dir)
		panic(err)
	}
	defer f.Close()

	_, err = f.WriteString(contents)
	if err != nil {
		os.RemoveAll(dir)
		panic(err)
	}

	return dir
}

func hasOnlyDestination(m Mappings, src string, dest string) bool {
	if len(m[src]) != 1 {
		return false
	}
	return m[src][0].String() == dest
}

func mapping(k string, v string) Mappings {
	m := make(Mappings, 1)
	m[k] = []abspath.AbsPath{getcwd().Join(v)}
	return m
}

func openFile(n string) *os.File {
	f, err := os.OpenFile(getcwd().Join(n).String(), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	_, err = f.WriteString("this file is for test")
	if err != nil {
		panic(err)
	}
	return f
}

func isSymlinkTo(n, d string) bool {
	cwd := getcwd()
	source := cwd.Join(n).String()
	s, err := os.Lstat(source)
	if err != nil {
		return false
	}
	if s.Mode()&os.ModeSymlink != os.ModeSymlink {
		return false
	}
	dist, err := os.Readlink(source)
	if err != nil {
		panic(err)
	}
	return dist == cwd.Join(d).String()
}

func createSymlink(from, to string) {
	cwd := getcwd()
	if err := os.Symlink(cwd.Join(from).String(), cwd.Join(to).String()); err != nil {
		panic(err)
	}
}

func TestGetMappingsConfigDirNotExist(t *testing.T) {
	p, err := abspath.ExpandFrom("unknown_directory")
	if err != nil {
		panic(err)
	}
	m, err := GetMappings(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) == 0 {
		t.Errorf("Mappings should not be empty. Default value is not set.")
	}
	if len(m[".vimrc"]) == 0 {
		t.Errorf("Any platform default value must have '.vimrc' mapping. %v", m)
	}
}

func TestGetMappingsConfigFileNotExist(t *testing.T) {
	testDir := createTestDir()
	defer os.Remove(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}
	m, err := GetMappings(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) == 0 {
		t.Errorf("Mappings should not be empty. Default value is not set.")
	}
	if len(m[".vimrc"]) == 0 {
		t.Errorf("Any platform default value must have '.vimrc' mapping. %v", m)
	}
}

func TestGetMappingsUnknownPlatform(t *testing.T) {
	p, err := abspath.ExpandFrom("unknown_directory")
	if err != nil {
		panic(err)
	}

	m, err := GetMappingsForPlatform("unknown", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 0 {
		t.Fatalf("Unknown mappings for unknown platform %v", m)
	}
}

func TestGetMappingsMappingsYAML(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
link:
  some_file: /path/to/some_file
  .vimrc: /override/path/vimrc
  .conf: ~/path/in/home
  multi_dest:
    - /dest1
    - /dest2
	`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	_, err = GetMappingsForPlatform("unknown", p)
	if err != nil {
		t.Fatal(err)
	}

	m, err := GetMappingsForPlatform("darwin", p)
	if err != nil {
		t.Fatal(err)
	}
	if !hasOnlyDestination(m, "some_file", "/path/to/some_file") {
		t.Errorf("Mapping value set in mappings.yaml is wrong: '%s' in Darwin", m["some_file"])
	}
	if !hasOnlyDestination(m, ".vimrc", "/override/path/vimrc") {
		t.Errorf("Mapping should be overridden but actually '%s' for Darwin platform", m[".vimrc"])
	}
	if p := m["multi_dest"]; len(p) != 2 || p[0].String() != "/dest1" || p[1].String() != "/dest2" {
		t.Errorf("Expected two mappings but got '%s' in Darwin", p)
	}
}

func TestGetConfigRelinkDefaultFalse(t *testing.T) {
	testDir := createTestDir()
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	cfg, err := GetConfigForPlatform("darwin", p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Relink {
		t.Fatalf("relink should default to false")
	}
}

func TestGetConfigRelinkFromMappingsYAML(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
relink: true
link:
  some_file: /path/to/some_file
`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	cfg, err := GetConfigForPlatform("unknown", p)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Relink {
		t.Fatalf("relink should be true")
	}
}

func TestGetConfigRelinkPlatformOverride(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
relink: true
link:
  some_file: /path/to/some_file
`)
	createTestMappingFile("mappings_darwin.yaml", `
relink: false
link:
  some_file: /path/to/some_file
`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	cfg, err := GetConfigForPlatform("darwin", p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Relink {
		t.Fatalf("relink should be overridden to false by mappings_darwin.yaml")
	}
}

func TestGetMappingsPartialLink(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
partial_link:
  zsh: /tmp/partial_target
	`)
	defer os.RemoveAll(testDir)

	zshDir := getcwd().Join(testDir).Join("zsh").String()
	if err := os.MkdirAll(zshDir, os.ModeDir|os.ModePerm); err != nil {
		panic(err)
	}
	zshrc, err := os.OpenFile(filepath.Join(zshDir, ".zshrc"), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	zshrc.Close()
	zprofile, err := os.OpenFile(filepath.Join(zshDir, ".zprofile"), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	zprofile.Close()

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	m, err := GetMappingsForPlatform("unknown", p)
	if err != nil {
		t.Fatal(err)
	}
	if !hasOnlyDestination(m, "zsh/.zshrc", "/tmp/partial_target/.zshrc") {
		t.Errorf("partial_link should expand zsh/.zshrc but got '%s'", m["zsh/.zshrc"])
	}
	if !hasOnlyDestination(m, "zsh/.zprofile", "/tmp/partial_target/.zprofile") {
		t.Errorf("partial_link should expand zsh/.zprofile but got '%s'", m["zsh/.zprofile"])
	}
}

func TestGetMappingsPartialLinkMissingSourceDir(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
partial_link:
  not_found: /tmp/partial_target
	`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	m, err := GetMappingsForPlatform("unknown", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 0 {
		t.Fatalf("Missing partial_link source directory should be ignored but got %v", m)
	}
}

func TestGetMappingsPreferRepoRootMappings(t *testing.T) {
	testDir := createTestDir()
	defer os.RemoveAll(testDir)

	rootFile := getcwd().Join(testDir).Join("mappings.yaml").String()
	root, err := os.OpenFile(rootFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	_, err = root.WriteString(`
link:
  some_file: /path/from/root
`)
	if err != nil {
		panic(err)
	}
	root.Close()

	dotfilesDir := getcwd().Join(testDir).Join(".dotfiles").String()
	if err := os.MkdirAll(dotfilesDir, os.ModeDir|os.ModePerm); err != nil {
		panic(err)
	}
	dotfilesFile := getcwd().Join(testDir).Join(".dotfiles").Join("mappings.yaml").String()
	dot, err := os.OpenFile(dotfilesFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	_, err = dot.WriteString(`
link:
  some_file: /path/from/dotfiles
`)
	if err != nil {
		panic(err)
	}
	dot.Close()

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	m, err := GetMappingsForPlatform("darwin", p)
	if err != nil {
		t.Fatal(err)
	}
	if !hasOnlyDestination(m, "some_file", "/path/from/root") {
		t.Errorf("mappings.yaml at repository root should be preferred but got '%s'", m["some_file"])
	}
}

func TestGetMappingsFallbackToDotfilesMappings(t *testing.T) {
	testDir := createTestDir()
	defer os.RemoveAll(testDir)

	dotfilesDir := getcwd().Join(testDir).Join(".dotfiles").String()
	if err := os.MkdirAll(dotfilesDir, os.ModeDir|os.ModePerm); err != nil {
		panic(err)
	}
	dotfilesFile := getcwd().Join(testDir).Join(".dotfiles").Join("mappings.yaml").String()
	dot, err := os.OpenFile(dotfilesFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	_, err = dot.WriteString(`
link:
  some_file: /path/from/dotfiles
`)
	if err != nil {
		panic(err)
	}
	dot.Close()

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	m, err := GetMappingsForPlatform("darwin", p)
	if err != nil {
		t.Fatal(err)
	}
	if !hasOnlyDestination(m, "some_file", "/path/from/dotfiles") {
		t.Errorf("mappings.yaml in .dotfiles should be loaded as fallback but got '%s'", m["some_file"])
	}
}

func TestGetMappingsPlatformSpecificMappingsYAML(t *testing.T) {
	testDir := createTestMappingFile("mappings_darwin.yaml", `
link:
  some_file: /path/to/some_file
  .vimrc: /override/path/vimrc
	`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	m, err := GetMappingsForPlatform("darwin", p)
	if err != nil {
		t.Fatal(err)
	}
	if !hasOnlyDestination(m, "some_file", "/path/to/some_file") {
		t.Errorf("Mapping value set in mappings_darwin.yaml is wrong: '%s' in Darwin", m["some_file"])
	}
	if !hasOnlyDestination(m, ".vimrc", "/override/path/vimrc") {
		t.Errorf("Mapping should be overridden by mappings_darwin.yaml but actually '%s'", m[".vimrc"])
	}

	m, err = GetMappingsForPlatform("windows", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(m["some_file"]) != 0 {
		t.Errorf("Different configuration must not be loaded but actually some_file was linked to '%s'", m["some_file"])
	}

	// Note: Consider '~' prefix in YAML path value
	if !strings.HasSuffix(m[".vimrc"][0].String(), defaultMappings["windows"][".vimrc"][0][1:]) {
		t.Errorf("Mapping should not be overridden by mappings_darwin.yaml on different platform (Windows) but actually '%s'", m[".vimrc"][0])
	}
}

func TestGetMappingsPlatformSpecificMappingsYAMLUnix(t *testing.T) {
	testDir := createTestMappingFile("mappings_unixlike.yaml", `
link:
  some_file: /path/to/some_file
  .vimrc: /hidden/path/vimrc
	`)
	createTestMappingFile("mappings_darwin.yaml", `
link:
  .vimrc: /override/path/vimrc
	`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	m, err := GetMappingsForPlatform("darwin", p)
	if err != nil {
		t.Fatal(err)
	}
	if !hasOnlyDestination(m, "some_file", "/path/to/some_file") {
		t.Errorf("Mapping value set in mappings_unixlike.yaml is wrong: '%s' in Darwin", m["some_file"])
	}
	if !hasOnlyDestination(m, ".vimrc", "/override/path/vimrc") {
		t.Errorf("Mapping should be overridden by mappings_darwin.yaml but actually '%s'", m[".vimrc"])
	}

	m, err = GetMappingsForPlatform("windows", p)
	if err != nil {
		t.Fatal(err)
	}
	if len(m["some_file"]) != 0 {
		t.Errorf("Different configuration must not be loaded but actually some_file was linked to '%s'", m["some_file"])
	}

	// Note: Consider '~' prefix in YAML path value
	if !strings.HasSuffix(m[".vimrc"][0].String(), defaultMappings["windows"][".vimrc"][0][1:]) {
		t.Errorf("Mapping should not be overridden by mappings_unixlike.yaml or mappings_darwin.yaml on different platform (Windows) but actually '%s'", m[".vimrc"][0])
	}
}

func TestGetMappingsInvalidYAML(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
link:
  some_file: [oops
	`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	if _, err := GetMappings(p); err == nil {
		t.Fatalf("Invalid YAML configuration must raise a parse error")
	}
}

func TestGetMappingsEmptyKey(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
link:
  "": /path/to/somewhere
	`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	if _, err := GetMappings(p); err == nil {
		t.Fatalf("Empty key must raise an error")
	}
}

func TestGetMappingsInvalidPathValue(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
link:
  some_file: relative-path
`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	if _, err := GetMappings(p); err == nil {
		t.Fatalf("Relative path must be checked")
	}
}

func TestGetMappingsWithoutLinkNamespace(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
some_file: /path/to/some_file
`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	if _, err := GetMappings(p); err == nil {
		t.Fatalf("Missing link and partial_link namespace must raise an error")
	}
}

func TestGetConfigWithPackageManagers(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
npm:
  - typescript
  - pnpm
homebrew:
  tap:
    - hashicorp/tap
  formula:
    - wget
  cask:
    - iterm2
`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	cfg, err := GetConfigForPlatform("unknown", p)
	if err != nil {
		t.Fatalf("Package manager only namespaces should be allowed but got: %s", err.Error())
	}

	if len(cfg.PackageManagers.NPM) != 2 || cfg.PackageManagers.NPM[0] != "typescript" || cfg.PackageManagers.NPM[1] != "pnpm" {
		t.Fatalf("Unexpected npm packages: %v", cfg.PackageManagers.NPM)
	}
	if len(cfg.PackageManagers.Homebrew.Tap) != 1 || cfg.PackageManagers.Homebrew.Tap[0] != "hashicorp/tap" {
		t.Fatalf("Unexpected homebrew taps: %v", cfg.PackageManagers.Homebrew.Tap)
	}
	if len(cfg.PackageManagers.Homebrew.Formula) != 1 || cfg.PackageManagers.Homebrew.Formula[0] != "wget" {
		t.Fatalf("Unexpected homebrew formula packages: %v", cfg.PackageManagers.Homebrew.Formula)
	}
	if len(cfg.PackageManagers.Homebrew.Cask) != 1 || cfg.PackageManagers.Homebrew.Cask[0] != "iterm2" {
		t.Fatalf("Unexpected homebrew cask packages: %v", cfg.PackageManagers.Homebrew.Cask)
	}
}

func TestGetConfigWithHomebrewFormulaShorthand(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
homebrew:
  - ripgrep
`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	cfg, err := GetConfigForPlatform("unknown", p)
	if err != nil {
		t.Fatalf("homebrew formula shorthand should be allowed but got: %s", err.Error())
	}
	if len(cfg.PackageManagers.Homebrew.Formula) != 1 || cfg.PackageManagers.Homebrew.Formula[0] != "ripgrep" {
		t.Fatalf("Unexpected homebrew formula packages: %v", cfg.PackageManagers.Homebrew.Formula)
	}
}

func TestGetMappingsPartialLinkOnly(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
partial_link:
  some_dir: /path/to/somewhere
`)
	defer os.RemoveAll(testDir)

	someDir := getcwd().Join(testDir).Join("some_dir").String()
	if err := os.MkdirAll(someDir, os.ModeDir|os.ModePerm); err != nil {
		panic(err)
	}
	someFile, err := os.OpenFile(filepath.Join(someDir, "config"), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	someFile.Close()

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	if _, err := GetMappings(p); err != nil {
		t.Fatalf("partial_link without link should be allowed but got error: %s", err.Error())
	}
}

func TestLinkNormalFile(t *testing.T) {
	cwd := getcwd()
	m := mapping("._test_source.conf", "_test.conf")
	f := openFile("._test_source.conf")
	defer func() {
		f.Close()
		defer os.Remove("._test_source.conf")
	}()

	err := m.CreateAllLinks(cwd, false)
	if err != nil {
		t.Fatal(err)
	}

	if !isSymlinkTo("_test.conf", "._test_source.conf") {
		t.Fatalf("Symbolic link not found")
	}
	defer os.Remove("_test.conf")

	// Skipping already existing link
	err = m.CreateAllLinks(cwd, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLinkSkipsDanglingSymlinkTarget(t *testing.T) {
	cwd := getcwd()
	m := mapping("._test_source.conf", "_dangling_dest.conf")
	f := openFile("._test_source.conf")
	defer func() {
		f.Close()
		os.Remove("._test_source.conf")
	}()

	createSymlink("._missing_source.conf", "_dangling_dest.conf")
	defer os.Remove("_dangling_dest.conf")

	if err := m.CreateAllLinks(cwd, false); err != nil {
		t.Fatal(err)
	}

	if !isSymlinkTo("_dangling_dest.conf", "._missing_source.conf") {
		t.Fatalf("Dangling symlink destination should be kept as-is")
	}
}

func TestLinkRelinkRecreatesExistingSymlinkTarget(t *testing.T) {
	cwd := getcwd()
	m := mapping("._test_source.conf", "_dist.conf")
	f := openFile("._test_source.conf")
	defer func() {
		f.Close()
		os.Remove("._test_source.conf")
	}()

	openFile("._old_source.conf").Close()
	defer os.Remove("._old_source.conf")

	createSymlink("._old_source.conf", "_dist.conf")
	defer os.Remove("_dist.conf")

	if err := m.CreateAllLinksWithRelink(cwd, false, true, nil); err != nil {
		t.Fatal(err)
	}

	if !isSymlinkTo("_dist.conf", "._test_source.conf") {
		t.Fatalf("Destination should be recreated to the expected source")
	}
}

func TestLinkToNonExistingDir(t *testing.T) {
	cwd := getcwd()
	m := mapping("._source.conf", "_dist_dir/_dist.conf")
	f := openFile("._source.conf")
	defer func() {
		f.Close()
		defer os.Remove("._source.conf")
	}()

	err := m.CreateAllLinks(cwd, false)
	if err != nil {
		t.Fatal(err)
	}

	if !isSymlinkTo("_dist_dir/_dist.conf", "._source.conf") {
		t.Fatalf("Symbolic link not found. Directory was not generated to put symlink into?")
	}
	defer os.RemoveAll("_dist_dir")
}

func TestLinkDirSymlink(t *testing.T) {
	cwd := getcwd()
	m := mapping("._source_dir", "_dist_dir")
	if err := os.MkdirAll("._source_dir", os.ModeDir|os.ModePerm); err != nil {
		panic(err)
	}
	defer os.Remove("._source_dir")

	err := m.CreateAllLinks(cwd, false)
	if err != nil {
		t.Fatal(err)
	}

	if !isSymlinkTo("_dist_dir", "._source_dir") {
		t.Fatalf("Symbolic link to directory not found.")
	}
	defer os.Remove("_dist_dir")
}

func TestLinkSpecifiedMappingOnly(t *testing.T) {
	cwd := getcwd()
	m := mapping("._source.conf", "_dist.conf")
	m["LICENSE.txt"] = []abspath.AbsPath{
		getcwd().Join("_never_created.txt"),
	}
	f := openFile("._source.conf")
	defer func() {
		f.Close()
		os.Remove("._source.conf")
	}()

	err := m.CreateSomeLinks([]string{"._source.conf"}, cwd, false)
	if err != nil {
		t.Fatal(err)
	}

	if !isSymlinkTo("_dist.conf", "._source.conf") {
		t.Fatalf("Symbolic link not found.")
	}
	defer os.Remove("_dist.conf")

	if isSymlinkTo("_never_created.txt", "LICENSE.txt") {
		t.Fatalf("Symbolic link not found.")
	}
}

func TestLinkSpecifyingNonExistingFile(t *testing.T) {
	cwd := getcwd()
	m := mapping("LICENSE.txt", "never_created.conf")

	err := m.CreateSomeLinks([]string{}, cwd, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat("never_created.conf"); err == nil {
		t.Errorf("never_created.conf was created")
		os.Remove("never_created.conf")
	}

	err = m.CreateSomeLinks([]string{"unknown_config.conf"}, cwd, false)
	if _, ok := err.(*NothingLinkedError); !ok {
		t.Fatal(err)
	}
	if _, err = os.Lstat("never_created.conf"); err == nil {
		t.Errorf("never_created.conf was created")
		os.Remove("never_created.conf")
	}
}

func TestLinkSourceNotExist(t *testing.T) {
	cwd := getcwd()
	m := mapping(".unknown.conf", "never_created.conf")
	err := m.CreateAllLinks(cwd, false)
	if _, ok := err.(*NothingLinkedError); !ok {
		t.Errorf("Not existing file must be ignored but actually error occurred: %s", err.Error())
	}
	m2 := mapping("unknown.conf", "never_created.conf")
	err = m2.CreateSomeLinks([]string{"unknown.conf"}, cwd, false)
	if _, ok := err.(*NothingLinkedError); !ok {
		t.Errorf("Not existing file must be ignored but actually error occurred: %s", err.Error())
	}
}

func TestLinkNullDest(t *testing.T) {
	cwd := getcwd()
	m := Mappings{
		"empty":     []abspath.AbsPath{},
		"null_only": []abspath.AbsPath{abspath.AbsPath{}},
	}
	err := m.CreateAllLinks(cwd, false)
	if err == nil {
		t.Errorf("Nothing was linked but error did not occur")
	}
}

func TestLinkDryRun(t *testing.T) {
	cwd := getcwd()
	m := mapping("._test_source.conf", "_test.conf")
	f := openFile("._test_source.conf")
	defer func() {
		f.Close()
		defer os.Remove("._test_source.conf")
	}()

	err := m.CreateAllLinks(cwd, true)
	if err != nil {
		t.Fatal(err)
	}

	if isSymlinkTo("_test.conf", "._test_source.conf") {
		t.Fatalf("Symbolic link should not be found")
	}
}

func TestUnlinkNoFile(t *testing.T) {
	m := mapping("._source.fonf", "._dist.conf")
	if err := m.UnlinkAll(getcwd()); err != nil {
		t.Error(err)
	}
}

func TestUnlinkFiles(t *testing.T) {
	f := openFile("._source.conf")
	defer func() {
		f.Close()
		os.Remove("._source.conf")
	}()
	createSymlink("._source.conf", "._dist.conf")
	m := mapping("._source.fonf", "._dist.conf")
	if err := m.UnlinkAll(getcwd()); err != nil {
		t.Error(err)
	}

	if _, err := os.Lstat("._dist.conf"); err == nil {
		os.Remove("._dist.conf")
		t.Errorf("Unlinked symlink must be removed")
	}
}

func TestUnlinkAnotherFileAlreadyExist(t *testing.T) {
	openFile("._dummy.conf").Close()
	defer os.Remove("._dummy.conf")
	m := mapping("._source.fonf", "._dummy.conf")
	if err := m.UnlinkAll(getcwd()); err != nil {
		t.Error(err)
	}
}

// e.g.
//
//	expected: dotfiles/vimrc -> ~/.vimrc
//	actual: another_dir/vimrc -> ~/.vimrc
func TestUnlinkDetectLinkToOutsideRepo(t *testing.T) {
	dir := getcwd().Join("_test_dir")

	if err := os.Mkdir(dir.String(), os.ModePerm|os.ModeDir); err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir.String())

	openFile("_outside.conf").Close()
	defer os.Remove("_outside.conf")

	createSymlink("_outside.conf", "_test.conf")
	defer os.Remove("_test.conf")

	m := mapping("_another_test.conf", "_test.conf")
	if err := m.UnlinkAll(dir); err != nil {
		t.Error(err)
	}

	if _, err := os.Lstat(getcwd().Join("_test.conf").String()); err != nil {
		t.Fatalf("When target is already linked to outside dotfiles, error should not occur: %s", err.Error())
	}
}

func TestActualLinksEmpty(t *testing.T) {
	m := mapping("._source.conf", "._dest.conf")
	l, err := m.ActualLinks(getcwd())
	if err != nil {
		t.Fatal(err)
	}
	if len(l) > 0 {
		t.Errorf("Link does not exist but actually '%v' was reported", l)
	}
}

func TestActualLinksLinkExists(t *testing.T) {
	openFile("._source.conf").Close()
	defer os.Remove("._source.conf")
	createSymlink("._source.conf", "._dist.conf")
	defer os.Remove("._dist.conf")
	cwd := getcwd()
	m := mapping("._source.fonf", "._dist.conf")

	l, err := m.ActualLinks(cwd)
	if err != nil {
		t.Fatal(err)
	}

	if len(l) != 1 {
		t.Fatalf("Only one mapping is intended to be added but actually %d mappings exist", len(l))
	}

	if l[0].src != cwd.Join("._source.conf").String() {
		t.Fatalf("._source.conf in current directory must be a source of symlink but actually not: '%v'", l)
	}

	expected := cwd.Join("._dist.conf").String()
	if l[0].dst != expected {
		t.Fatalf("'%s' is expected as a dist of symlink, but actually '%s'", expected, l[0].dst)
	}
}

func TestActualLinksNotDotfile(t *testing.T) {
	openFile("._source.conf").Close()
	defer os.Remove("._source.conf")
	openFile("._dist.conf").Close()
	defer os.Remove("._dist.conf")
	cwd := getcwd()
	m := mapping("._source.fonf", "._dist.conf")

	l, err := m.ActualLinks(cwd)
	if err != nil {
		t.Fatal(err)
	}

	if len(l) > 0 {
		t.Fatalf("When a mapping is a hard link, it's not a dotfile and should not considered.  But actually links '%v' are detected", l)
	}
}

func TestActualLinksTwoDestsFromOneSource(t *testing.T) {
	openFile("._source.conf").Close()
	defer os.Remove("._source.conf")
	createSymlink("._source.conf", "._dest1.conf")
	defer os.Remove("._dest1.conf")
	createSymlink("._source.conf", "._dest2.conf")
	defer os.Remove("._dest2.conf")
	cwd := getcwd()
	m := Mappings{
		"._source.conf": []abspath.AbsPath{getcwd().Join("._dest1.conf"), getcwd().Join("._dest2.conf")},
	}

	links, err := m.ActualLinks(cwd)
	if err != nil {
		t.Fatal(err)
	}

	if len(links) != 2 {
		t.Fatalf("Two mappings are intended to be added but actually %d mappings exist", len(links))
	}

	src := cwd.Join("._source.conf").String()
	expected := []string{"._dest1.conf", "._dest2.conf"}

	// `links` is generated from map. Order of elements in map is randomized. Adjust order of
	// `expected` here
	if strings.HasSuffix(links[0].dst, "._dest2.conf") {
		// Swap order of `expected`
		tmp := expected[0]
		expected[0] = expected[1]
		expected[1] = tmp
	}

	for i, c := range expected {
		l := links[i]
		if l.src != src {
			t.Fatalf("Wanted %+v but got %+v for source (index=%d)", src, l.src, i)
		}
		dst := cwd.Join(c).String()
		if l.dst != dst {
			t.Fatalf("Wanted %+v but got %+v for source (index=%d)", dst, l.dst, i)
		}
	}
}

func TestConvertRawMappingsToMappings(t *testing.T) {
	raw := rawMappings{
		"empty":     []string{},
		"null_only": []string{""},
	}
	m, err := convertRawMappingsToMappings(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(m["empty"]) != 0 {
		t.Fatalf("Converted mapping value for `empty` is wrong: '%v'", m["empty"])
	}
	// Expected value for `null_only` is also an empty slice,
	// because the empty string is ignored when converting.
	if len(m["null_only"]) != 0 {
		t.Fatalf("Converted mapping value for `null_only` is wrong: '%v'", m["null_only"])
	}
}

func TestLinkOutsideDir(t *testing.T) {
	testDir := createTestDir()
	defer os.Remove(testDir)

	cwd := getcwd()
	f := "._test_source.conf"
	m := mapping(f, "_test.conf")
	p := filepath.Join(testDir, f)
	openFile(p).Close()

	d := cwd.Join(testDir)
	err := m.CreateAllLinks(d, false)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("_test.conf")

	if !isSymlinkTo("_test.conf", p) {
		t.Fatalf("Symbolic link not found")
	}

	// Skipping already existing link
	err = m.CreateAllLinks(d, false)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseObjectMappingWithRequireTarget(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
link:
  simple_file: /path/to/simple
  obj_file:
    path: /path/to/target
    require_target: true
  obj_multi:
    path:
      - /path/to/target1
      - /path/to/target2
    require_target: true
`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	cfg, err := GetConfigForPlatform("unknown", p)
	if err != nil {
		t.Fatal(err)
	}

	if !hasOnlyDestination(cfg.Mappings, "simple_file", "/path/to/simple") {
		t.Errorf("simple_file mapping incorrect: %v", cfg.Mappings["simple_file"])
	}

	if !hasOnlyDestination(cfg.Mappings, "obj_file", "/path/to/target") {
		t.Errorf("obj_file mapping incorrect: %v", cfg.Mappings["obj_file"])
	}

	if len(cfg.Mappings["obj_multi"]) != 2 {
		t.Fatalf("obj_multi should have 2 paths, got %d", len(cfg.Mappings["obj_multi"]))
	}

	if !cfg.RequireTarget["obj_file"] {
		t.Errorf("obj_file should have require_target=true")
	}
	if !cfg.RequireTarget["obj_multi"] {
		t.Errorf("obj_multi should have require_target=true")
	}
	if cfg.RequireTarget["simple_file"] {
		t.Errorf("simple_file should not have require_target")
	}
}

func TestLinkRequireTargetSkipsWhenParentMissing(t *testing.T) {
	cwd := getcwd()
	m := mapping("._source.conf", "_nonexistent_dir/_dist.conf")
	f := openFile("._source.conf")
	defer func() {
		f.Close()
		os.Remove("._source.conf")
	}()

	reqTarget := map[string]bool{"._source.conf": true}
	err := m.CreateAllLinksWithRelink(cwd, false, false, reqTarget)
	// Should get NothingLinkedError because the link was skipped
	if _, ok := err.(*NothingLinkedError); !ok {
		t.Fatalf("Expected NothingLinkedError when parent dir missing with require_target, got: %v", err)
	}

	if _, err := os.Lstat(cwd.Join("_nonexistent_dir").String()); !os.IsNotExist(err) {
		t.Fatalf("Parent directory should not have been created")
		os.RemoveAll("_nonexistent_dir")
	}
}

func TestLinkRequireTargetLinksWhenParentExists(t *testing.T) {
	cwd := getcwd()
	if err := os.MkdirAll("_existing_dir", os.ModeDir|os.ModePerm); err != nil {
		panic(err)
	}
	defer os.RemoveAll("_existing_dir")

	m := mapping("._source.conf", "_existing_dir/_dist.conf")
	f := openFile("._source.conf")
	defer func() {
		f.Close()
		os.Remove("._source.conf")
	}()

	reqTarget := map[string]bool{"._source.conf": true}
	err := m.CreateAllLinksWithRelink(cwd, false, false, reqTarget)
	if err != nil {
		t.Fatal(err)
	}

	if !isSymlinkTo("_existing_dir/_dist.conf", "._source.conf") {
		t.Fatalf("Symbolic link not found when parent dir exists with require_target")
	}
}

func TestRequireTargetPlatformOverrideClearsFalse(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
link:
  foo:
    path: /path/to/foo
    require_target: true
  bar:
    path: /path/to/bar
    require_target: true
`)
	createTestMappingFile("mappings_darwin.yaml", `
link:
  foo:
    path: /path/to/foo
    require_target: false
`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	cfg, err := GetConfigForPlatform("darwin", p)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.RequireTarget["foo"] {
		t.Errorf("foo require_target should be overridden to false by mappings_darwin.yaml")
	}
	if !cfg.RequireTarget["bar"] {
		t.Errorf("bar require_target should remain true (not mentioned in platform file)")
	}
}

func TestParseObjectMappingWithoutRequireTarget(t *testing.T) {
	testDir := createTestMappingFile("mappings.yaml", `
link:
  obj_file:
    path: /path/to/target
`)
	defer os.RemoveAll(testDir)

	p, err := abspath.ExpandFrom(testDir)
	if err != nil {
		panic(err)
	}

	cfg, err := GetConfigForPlatform("unknown", p)
	if err != nil {
		t.Fatal(err)
	}

	if !hasOnlyDestination(cfg.Mappings, "obj_file", "/path/to/target") {
		t.Errorf("obj_file mapping incorrect: %v", cfg.Mappings["obj_file"])
	}
	if cfg.RequireTarget["obj_file"] {
		t.Errorf("obj_file should not have require_target when not specified")
	}
}
