package dotfiles

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/rhysd/abspath"
	"gopkg.in/yaml.v3"
)

type NothingLinkedError struct {
	RepoPath string
}

func (err NothingLinkedError) Error() string {
	if err.RepoPath == "" {
		return "Nothing was linked."
	}
	return fmt.Sprintf("Nothing was linked. '%s' was specified as dotfiles repository. Please check it", err.RepoPath)
}

// unixLikePlatformName is a special platform name used commonly for Unix-like platform (Linux and macOS)
const unixLikePlatformName = "unixlike"

type Mappings map[string][]abspath.AbsPath
type rawMappings map[string][]string
type rawPartialMappings map[string]string

type HomebrewPackages struct {
	Tap     []string
	Formula []string
	Cask    []string
}

type PackageManagers struct {
	NPM      []string
	Homebrew HomebrewPackages
}

type Config struct {
	Mappings        Mappings
	RequireTarget   map[string]bool
	PackageManagers PackageManagers
	Relink          bool
}

type parsedMappingsYAML struct {
	link          rawMappings
	requireTarget map[string]bool
	partialLink   rawPartialMappings
	npm           []string
	homebrew      HomebrewPackages
	relink        *bool
}

var defaultMappings = map[string]rawMappings{
	"windows": rawMappings{
		".gvimrc": []string{"~/vimfiles/gvimrc"},
		".vim":    []string{"~/vimfiles"},
		".vimrc":  []string{"~/vimfiles/vimrc"},
	},
	unixLikePlatformName: rawMappings{
		".agignore":      []string{"~/.agignore"},
		".bash_login":    []string{"~/.bash_login"},
		".bash_profile":  []string{"~/.bash_profile"},
		".bashrc":        []string{"~/.bashrc"},
		".emacs.d":       []string{"~/.emacs.d"},
		".emacs.el":      []string{"~/.emacs.d/init.el"},
		".eslintrc":      []string{"~/.eslintrc"},
		".eslintrc.json": []string{"~/.eslintrc.json"},
		".eslintrc.yml":  []string{"~/.eslintrc.yml"},
		".gvimrc":        []string{"~/.gvimrc"},
		".npmrc":         []string{"~/.npmrc"},
		".profile":       []string{"~/.profile"},
		".pryrc":         []string{"~/.pryrc"},
		".pylintrc":      []string{"~/.pylintrc"},
		".tmux.conf":     []string{"~/.tmux.conf"},
		".vim":           []string{"~/.vim"},
		".vimrc":         []string{"~/.vimrc"},
		".zlogin":        []string{"~/.zlogin"},
		".zprofile":      []string{"~/.zprofile"},
		".zshenv":        []string{"~/.zshenv"},
		".zshrc":         []string{"~/.zshrc"},
		"agignore":       []string{"~/.agignore"},
		"bash_login":     []string{"~/.bash_login"},
		"bash_profile":   []string{"~/.bash_profile"},
		"bashrc":         []string{"~/.bashrc"},
		"emacs.d":        []string{"~/.emacs.d"},
		"emacs.el":       []string{"~/.emacs.d/init.el"},
		"eslintrc":       []string{"~/.eslintrc"},
		"eslintrc.json":  []string{"~/.eslintrc.json"},
		"eslintrc.yml":   []string{"~/.eslintrc.yml"},
		"gvimrc":         []string{"~/.gvimrc"},
		"npmrc":          []string{"~/.npmrc"},
		"profile":        []string{"~/.profile"},
		"pryrc":          []string{"~/.pryrc"},
		"pylintrc":       []string{"~/.pylintrc"},
		"tmux.conf":      []string{"~/.tmux.conf"},
		"vim":            []string{"~/.vim"},
		"vimrc":          []string{"~/.vimrc"},
		"zlogin":         []string{"~/.zlogin"},
		"zprofile":       []string{"~/.zprofile"},
		"zshenv":         []string{"~/.zshenv"},
		"zshrc":          []string{"~/.zshrc"},
		"init.el":        []string{"~/.emacs.d/init.el"},
		"peco":           []string{"~/.config/peco"},
	},
	"linux": rawMappings{
		".Xmodmap":    []string{"~/.Xmodmap"},
		".Xresources": []string{"~/.Xresources"},
		"Xmodmap":     []string{"~/.Xmodmap"},
		"Xresources":  []string{"~/.Xresources"},
		"rc.lua":      []string{"~/.config/rc.lua"},
	},
	"darwin": rawMappings{
		".htoprc": []string{"~/.htoprc"},
		"htoprc":  []string{"~/.htoprc"},
	},
}

type PathLink struct {
	src, dst string
}

func parseMappingsYAML(file abspath.AbsPath) (*parsedMappingsYAML, error) {
	var m map[string]interface{}

	bytes, err := ioutil.ReadFile(file.String())
	if err != nil {
		// Note:
		// It's not an error that the file is not found
		return nil, nil
	}

	if err := yaml.Unmarshal(bytes, &m); err != nil {
		return nil, err
	}

	ret := &parsedMappingsYAML{}
	hasNamespace := false

	if linkMappings, ok := m["link"]; ok {
		hasNamespace = true
		switch section := linkMappings.(type) {
		case map[string]interface{}:
			raw, reqTarget, err := parseRawMappings(section)
			if err != nil {
				return nil, err
			}
			ret.link = raw
			ret.requireTarget = reqTarget
		default:
			return nil, fmt.Errorf("'link' section in mappings must be an object")
		}
	}

	if partialMappings, ok := m["partial_link"]; ok {
		hasNamespace = true
		switch section := partialMappings.(type) {
		case map[string]interface{}:
			raw, err := parseRawPartialMappings(section)
			if err != nil {
				return nil, err
			}
			ret.partialLink = raw
		default:
			return nil, fmt.Errorf("'partial_link' section in mappings must be an object")
		}
	}

	if npmMappings, ok := m["npm"]; ok {
		hasNamespace = true
		raw, err := parseRawStringList(npmMappings, "npm")
		if err != nil {
			return nil, err
		}
		ret.npm = raw
	}

	if homebrewMappings, ok := m["homebrew"]; ok {
		hasNamespace = true
		switch section := homebrewMappings.(type) {
		case map[string]interface{}:
			raw, err := parseRawHomebrew(section)
			if err != nil {
				return nil, err
			}
			ret.homebrew = raw
		case []interface{}:
			raw, err := parseRawStringList(section, "homebrew")
			if err != nil {
				return nil, err
			}
			ret.homebrew.Formula = raw
		default:
			return nil, fmt.Errorf("'homebrew' section in mappings must be an object or string[]")
		}
	}

	if relinkRaw, ok := m["relink"]; ok {
		hasNamespace = true
		switch relink := relinkRaw.(type) {
		case bool:
			ret.relink = &relink
		default:
			return nil, fmt.Errorf("'relink' section in mappings must be a boolean")
		}
	}

	if !hasNamespace {
		return nil, fmt.Errorf("at least one of 'link', 'partial_link', 'npm', 'homebrew', or 'relink' sections in mappings is required")
	}

	return ret, nil
}

func parseRawMappings(m map[string]interface{}) (rawMappings, map[string]bool, error) {
	maps := make(rawMappings, len(m))
	reqTarget := map[string]bool{}
	for k, v := range m {
		switch v := v.(type) {
		case string:
			maps[k] = []string{v}
		case []interface{}:
			vs := make([]string, 0, len(v))
			for _, iface := range v {
				s, ok := iface.(string)
				if !ok {
					return nil, nil, fmt.Errorf("value of mappings object must be string or string[]: %v", v)
				}
				vs = append(vs, s)
			}
			maps[k] = vs
		case map[string]interface{}:
			paths, rt, err := parseObjectMapping(v)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid object mapping for '%s': %w", k, err)
			}
			maps[k] = paths
			reqTarget[k] = rt
		}
	}

	return maps, reqTarget, nil
}

func parseObjectMapping(m map[string]interface{}) ([]string, bool, error) {
	pathRaw, ok := m["path"]
	if !ok {
		return nil, false, fmt.Errorf("'path' is required in object mapping")
	}

	var paths []string
	switch p := pathRaw.(type) {
	case string:
		paths = []string{p}
	case []interface{}:
		paths = make([]string, 0, len(p))
		for _, iface := range p {
			s, ok := iface.(string)
			if !ok {
				return nil, false, fmt.Errorf("'path' values must be strings: %v", p)
			}
			paths = append(paths, s)
		}
	default:
		return nil, false, fmt.Errorf("'path' must be a string or string[]: %v", pathRaw)
	}

	requireTarget := false
	if rt, ok := m["require_target"]; ok {
		b, ok := rt.(bool)
		if !ok {
			return nil, false, fmt.Errorf("'require_target' must be a boolean: %v", rt)
		}
		requireTarget = b
	}

	return paths, requireTarget, nil
}

func parseRawPartialMappings(m map[string]interface{}) (rawPartialMappings, error) {
	maps := make(rawPartialMappings, len(m))
	for k, v := range m {
		if k == "" {
			return nil, fmt.Errorf("empty key cannot be included.  Note: Corresponding value is '%v'", v)
		}
		switch v := v.(type) {
		case string:
			if v == "" {
				continue
			}
			if v[0] != '~' && v[0] != '/' {
				return nil, fmt.Errorf("value of partial_link mappings must be an absolute path like '/foo/.bar' or '~/.foo': %s", v)
			}
			maps[k] = v
		default:
			return nil, fmt.Errorf("value of partial_link mappings object must be string: %v", v)
		}
	}
	return maps, nil
}

func parseRawStringList(raw interface{}, namespace string) ([]string, error) {
	values, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("'%s' section in mappings must be string[]", namespace)
	}

	ret := make([]string, 0, len(values))
	for _, v := range values {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("value of '%s' section must be string[]: %v", namespace, values)
		}
		if s == "" {
			continue
		}
		ret = append(ret, s)
	}
	return ret, nil
}

func parseRawHomebrew(m map[string]interface{}) (HomebrewPackages, error) {
	ret := HomebrewPackages{}
	for key, raw := range m {
		switch key {
		case "tap":
			tap, err := parseRawStringList(raw, "homebrew.tap")
			if err != nil {
				return HomebrewPackages{}, err
			}
			ret.Tap = tap
		case "formula":
			formula, err := parseRawStringList(raw, "homebrew.formula")
			if err != nil {
				return HomebrewPackages{}, err
			}
			ret.Formula = formula
		case "cask":
			cask, err := parseRawStringList(raw, "homebrew.cask")
			if err != nil {
				return HomebrewPackages{}, err
			}
			ret.Cask = cask
		default:
			return HomebrewPackages{}, fmt.Errorf("unknown key in 'homebrew' section: %s", key)
		}
	}
	return ret, nil
}

func convertRawMappingsToMappings(raw rawMappings) (Mappings, error) {
	if raw == nil {
		return nil, nil
	}
	m := make(Mappings, len(raw))
	for k, vs := range raw {
		if k == "" {
			return nil, fmt.Errorf("empty key cannot be included.  Note: Corresponding value is '%s'", vs)
		}
		ps := make([]abspath.AbsPath, 0, len(vs))
		for _, v := range vs {
			if v == "" {
				continue
			}
			if v[0] != '~' && v[0] != '/' {
				return nil, fmt.Errorf("value of mappings must be an absolute path like '/foo/.bar' or '~/.foo': %s", v)
			}
			p, err := abspath.ExpandFromSlash(v)
			if err != nil {
				return nil, err
			}
			ps = append(ps, p)
		}
		m[k] = ps
	}
	return m, nil
}

func mergeMappingsFromDefault(dist Mappings, platform string) error {
	m, err := convertRawMappingsToMappings(defaultMappings[platform])
	if err != nil {
		return err
	}

	for k, v := range m {
		dist[k] = v
	}

	return nil
}

func mergeMappingsFromFile(dist Mappings, reqTarget map[string]bool, file abspath.AbsPath) error {
	parsed, err := parseMappingsYAML(file)
	if err != nil {
		return err
	}
	if parsed == nil {
		return nil
	}

	m, err := convertRawMappingsToMappings(parsed.link)
	if err != nil {
		return err
	}

	for k, v := range m {
		dist[k] = v
	}
	for k, v := range parsed.requireTarget {
		reqTarget[k] = v
	}

	return nil
}

func mergePartialMappingsFromFile(dist rawPartialMappings, file abspath.AbsPath) error {
	parsed, err := parseMappingsYAML(file)
	if err != nil {
		return err
	}
	if parsed == nil || parsed.partialLink == nil {
		return nil
	}

	for k, v := range parsed.partialLink {
		dist[k] = v
	}

	return nil
}

func appendUniqueStrings(dist []string, src []string) []string {
	if len(src) == 0 {
		return dist
	}
	seen := map[string]struct{}{}
	for _, v := range dist {
		seen[v] = struct{}{}
	}
	for _, v := range src {
		if _, ok := seen[v]; ok {
			continue
		}
		dist = append(dist, v)
		seen[v] = struct{}{}
	}
	return dist
}

func mergePackageManagersFromFile(dist *PackageManagers, file abspath.AbsPath) error {
	parsed, err := parseMappingsYAML(file)
	if err != nil {
		return err
	}
	if parsed == nil {
		return nil
	}

	dist.NPM = appendUniqueStrings(dist.NPM, parsed.npm)
	dist.Homebrew.Tap = appendUniqueStrings(dist.Homebrew.Tap, parsed.homebrew.Tap)
	dist.Homebrew.Formula = appendUniqueStrings(dist.Homebrew.Formula, parsed.homebrew.Formula)
	dist.Homebrew.Cask = appendUniqueStrings(dist.Homebrew.Cask, parsed.homebrew.Cask)
	return nil
}

func mergeRelinkFromFile(current bool, file abspath.AbsPath) (bool, error) {
	parsed, err := parseMappingsYAML(file)
	if err != nil {
		return false, err
	}
	if parsed == nil || parsed.relink == nil {
		return current, nil
	}
	return *parsed.relink, nil
}

func mergeMappingsFromPreferredFile(dist Mappings, reqTarget map[string]bool, parent abspath.AbsPath, name string) error {
	root := parent.Join(name)
	if _, err := os.Stat(root.String()); err == nil {
		return mergeMappingsFromFile(dist, reqTarget, root)
	} else if !os.IsNotExist(err) {
		return err
	}

	dotfiles := parent.Join(".dotfiles").Join(name)
	if _, err := os.Stat(dotfiles.String()); err == nil {
		return mergeMappingsFromFile(dist, reqTarget, dotfiles)
	} else if !os.IsNotExist(err) {
		return err
	}

	return nil
}

func mergePartialMappingsFromPreferredFile(dist rawPartialMappings, parent abspath.AbsPath, name string) error {
	root := parent.Join(name)
	if _, err := os.Stat(root.String()); err == nil {
		return mergePartialMappingsFromFile(dist, root)
	} else if !os.IsNotExist(err) {
		return err
	}

	dotfiles := parent.Join(".dotfiles").Join(name)
	if _, err := os.Stat(dotfiles.String()); err == nil {
		return mergePartialMappingsFromFile(dist, dotfiles)
	} else if !os.IsNotExist(err) {
		return err
	}

	return nil
}

func mergePackageManagersFromPreferredFile(dist *PackageManagers, parent abspath.AbsPath, name string) error {
	root := parent.Join(name)
	if _, err := os.Stat(root.String()); err == nil {
		return mergePackageManagersFromFile(dist, root)
	} else if !os.IsNotExist(err) {
		return err
	}

	dotfiles := parent.Join(".dotfiles").Join(name)
	if _, err := os.Stat(dotfiles.String()); err == nil {
		return mergePackageManagersFromFile(dist, dotfiles)
	} else if !os.IsNotExist(err) {
		return err
	}

	return nil
}

func mergeRelinkFromPreferredFile(current bool, parent abspath.AbsPath, name string) (bool, error) {
	root := parent.Join(name)
	if _, err := os.Stat(root.String()); err == nil {
		return mergeRelinkFromFile(current, root)
	} else if !os.IsNotExist(err) {
		return false, err
	}

	dotfiles := parent.Join(".dotfiles").Join(name)
	if _, err := os.Stat(dotfiles.String()); err == nil {
		return mergeRelinkFromFile(current, dotfiles)
	} else if !os.IsNotExist(err) {
		return false, err
	}

	return current, nil
}

func isUnixLikePlatform(platform string) bool {
	return platform == "linux" || platform == "darwin"
}

func GetConfigForPlatform(platform string, parent abspath.AbsPath) (*Config, error) {
	m := Mappings{}
	reqTarget := map[string]bool{}
	partial := rawPartialMappings{}
	pm := PackageManagers{}
	relink := false

	if isUnixLikePlatform(platform) {
		if err := mergeMappingsFromDefault(m, unixLikePlatformName); err != nil {
			return nil, err
		}
	}
	if err := mergeMappingsFromDefault(m, platform); err != nil {
		return nil, err
	}

	if err := mergeMappingsFromPreferredFile(m, reqTarget, parent, "mappings.yaml"); err != nil {
		return nil, err
	}
	if err := mergePartialMappingsFromPreferredFile(partial, parent, "mappings.yaml"); err != nil {
		return nil, err
	}
	if err := mergePackageManagersFromPreferredFile(&pm, parent, "mappings.yaml"); err != nil {
		return nil, err
	}
	relink, err := mergeRelinkFromPreferredFile(relink, parent, "mappings.yaml")
	if err != nil {
		return nil, err
	}

	if isUnixLikePlatform(platform) {
		if err := mergeMappingsFromPreferredFile(m, reqTarget, parent, fmt.Sprintf("mappings_%s.yaml", unixLikePlatformName)); err != nil {
			return nil, err
		}
		if err := mergePartialMappingsFromPreferredFile(partial, parent, fmt.Sprintf("mappings_%s.yaml", unixLikePlatformName)); err != nil {
			return nil, err
		}
		if err := mergePackageManagersFromPreferredFile(&pm, parent, fmt.Sprintf("mappings_%s.yaml", unixLikePlatformName)); err != nil {
			return nil, err
		}
		relink, err = mergeRelinkFromPreferredFile(relink, parent, fmt.Sprintf("mappings_%s.yaml", unixLikePlatformName))
		if err != nil {
			return nil, err
		}
	}
	if err := mergeMappingsFromPreferredFile(m, reqTarget, parent, fmt.Sprintf("mappings_%s.yaml", platform)); err != nil {
		return nil, err
	}
	if err := mergePartialMappingsFromPreferredFile(partial, parent, fmt.Sprintf("mappings_%s.yaml", platform)); err != nil {
		return nil, err
	}
	if err := mergePackageManagersFromPreferredFile(&pm, parent, fmt.Sprintf("mappings_%s.yaml", platform)); err != nil {
		return nil, err
	}
	relink, err = mergeRelinkFromPreferredFile(relink, parent, fmt.Sprintf("mappings_%s.yaml", platform))
	if err != nil {
		return nil, err
	}

	expanded, err := expandPartialMappings(partial, parent)
	if err != nil {
		return nil, err
	}
	for k, v := range expanded {
		// Explicit 'link' mappings are prioritized when conflicts happen.
		if _, exists := m[k]; exists {
			continue
		}
		m[k] = v
	}

	return &Config{Mappings: m, RequireTarget: reqTarget, PackageManagers: pm, Relink: relink}, nil
}

func GetMappingsForPlatform(platform string, parent abspath.AbsPath) (Mappings, error) {
	cfg, err := GetConfigForPlatform(platform, parent)
	if err != nil {
		return nil, err
	}
	return cfg.Mappings, nil
}

func expandPartialMappings(partial rawPartialMappings, repo abspath.AbsPath) (Mappings, error) {
	if partial == nil {
		return nil, nil
	}

	expanded := Mappings{}
	for fromDir, toDir := range partial {
		toBase, err := abspath.ExpandFromSlash(toDir)
		if err != nil {
			return nil, err
		}

		entries, err := os.ReadDir(repo.Join(filepath.FromSlash(fromDir)).String())
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		for _, entry := range entries {
			name := entry.Name()
			expanded[filepath.ToSlash(filepath.Join(fromDir, name))] = []abspath.AbsPath{
				toBase.Join(filepath.FromSlash(name)),
			}
		}
	}

	return expanded, nil
}

func GetMappings(configDir abspath.AbsPath) (Mappings, error) {
	return GetMappingsForPlatform(runtime.GOOS, configDir)
}

func GetConfig(configDir abspath.AbsPath) (*Config, error) {
	return GetConfigForPlatform(runtime.GOOS, configDir)
}

func link(from, to abspath.AbsPath, dry bool, relink bool, requireTarget bool) (bool, error) {
	if _, err := os.Stat(from.String()); err != nil {
		return false, nil
	}

	if requireTarget {
		if _, err := os.Stat(to.Dir().String()); os.IsNotExist(err) {
			color.Yellow("Skip (target not found): '%s' -> '%s'\n", from, to.String())
			return false, nil
		} else if err != nil {
			return false, err
		}
	}

	if _, err := os.Lstat(to.String()); err == nil {
		if !relink {
			// Target already exists. Skipped.
			fmt.Printf("Exist: '%s' -> '%s'\n", from, to.String())
			return true, nil
		}
		if dry {
			color.Yellow("Relink: '%s' -> '%s'\n", from, to.String())
			return true, nil
		}
		if err := os.Remove(to.String()); err != nil {
			return false, err
		}
		color.Yellow("Relink: '%s' -> '%s'\n", from, to.String())
	} else if !os.IsNotExist(err) {
		return false, err
	}

	if err := os.MkdirAll(to.Dir().String(), os.ModeDir|os.ModePerm); err != nil {
		return false, err
	}

	color.Cyan("Link:  '%s' -> '%s'\n", from, to.String())

	if dry {
		return true, nil
	}

	if err := os.Symlink(from.String(), to.String()); err != nil {
		return false, err
	}

	return true, nil
}

func (maps Mappings) CreateAllLinks(dir abspath.AbsPath, dry bool) error {
	return maps.CreateAllLinksWithRelink(dir, dry, false, nil)
}

func (maps Mappings) CreateAllLinksWithRelink(dir abspath.AbsPath, dry bool, relink bool, requireTarget map[string]bool) error {
	created := false
	for f, tos := range maps {
		from := dir.Join(filepath.FromSlash(f))
		rt := requireTarget[f]
		for _, to := range tos {
			linked, err := link(from, to, dry, relink, rt)
			if err != nil {
				return err
			}
			if linked {
				created = true
			}
		}
	}

	if !created {
		return &NothingLinkedError{}
	}

	return nil
}

func (maps Mappings) CreateSomeLinks(specified []string, dir abspath.AbsPath, dry bool) error {
	return maps.CreateSomeLinksWithRelink(specified, dir, dry, false, nil)
}

func (maps Mappings) CreateSomeLinksWithRelink(specified []string, dir abspath.AbsPath, dry bool, relink bool, requireTarget map[string]bool) error {
	created := false
	for _, f := range specified {
		if tos, ok := maps[f]; ok {
			from := dir.Join(filepath.FromSlash(f))
			rt := requireTarget[f]
			for _, to := range tos {
				linked, err := link(from, to, dry, relink, rt)
				if err != nil {
					return err
				}
				if linked {
					created = true
				}
			}
		}
	}

	if !created && len(specified) > 0 {
		return &NothingLinkedError{}
	}

	return nil
}

func getLinkSource(repo, to abspath.AbsPath) (string, error) {
	s, err := os.Lstat(to.String())
	if err != nil {
		// Note: Symlink not found
		return "", nil
	}

	if s.Mode()&os.ModeSymlink != os.ModeSymlink {
		return "", nil
	}

	source, err := os.Readlink(to.String())
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(source, repo.String()) {
		// Note: When the symlink is not linked from dotfiles repository.
		return "", nil
	}

	return source, nil
}

func (maps Mappings) unlink(repo, to abspath.AbsPath) (bool, error) {
	source, err := getLinkSource(repo, to)
	if source == "" || err != nil {
		return false, err
	}

	if err := os.Remove(to.String()); err != nil {
		return false, err
	}

	fmt.Printf("Unlink: '%s' -> '%s'\n", source, to.String())

	return true, nil
}

func (maps Mappings) UnlinkAll(repo abspath.AbsPath) error {
	removed := false
	for _, tos := range maps {
		for _, to := range tos {
			unlinked, err := maps.unlink(repo, to)
			if err != nil {
				return err
			}
			if unlinked {
				removed = true
			}
		}
	}

	if !removed {
		fmt.Printf("No symlink was removed (dotfiles: '%s').\n", repo.String())
	}

	return nil
}

func (maps Mappings) ActualLinks(repo abspath.AbsPath) ([]PathLink, error) {
	// Avoid duplicate of destination by using map. For example, when following mappings exist:
	//   my_vimrc -> ~/.vimrc (from user config)
	//   .vimrc -> ~/.vimrc (from default config)
	// It might lists up duplicate links. (#9)
	m := map[PathLink]struct{}{}
	for _, tos := range maps {
		for _, to := range tos {
			s, err := getLinkSource(repo, to)
			if err != nil {
				return nil, err
			}
			if s != "" {
				m[PathLink{s, to.String()}] = struct{}{}
			}
		}
	}

	ret := make([]PathLink, 0, len(m))
	for l := range m {
		ret = append(ret, l)
	}

	return ret, nil
}
