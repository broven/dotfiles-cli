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

type parsedMappingsYAML struct {
	link        rawMappings
	partialLink rawPartialMappings
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

	if linkMappings, ok := m["link"]; ok {
		switch section := linkMappings.(type) {
		case map[string]interface{}:
			raw, err := parseRawMappings(section)
			if err != nil {
				return nil, err
			}
			ret.link = raw
		default:
			return nil, fmt.Errorf("'link' section in mappings must be an object")
		}
	}

	if partialMappings, ok := m["partial_link"]; ok {
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

	if ret.link == nil && ret.partialLink == nil {
		return nil, fmt.Errorf("'link' or 'partial_link' section in mappings is required")
	}

	return ret, nil
}

func parseRawMappings(m map[string]interface{}) (rawMappings, error) {
	maps := make(rawMappings, len(m))
	for k, v := range m {
		switch v := v.(type) {
		case string:
			maps[k] = []string{v}
		case []interface{}:
			vs := make([]string, 0, len(v))
			for _, iface := range v {
				s, ok := iface.(string)
				if !ok {
					return nil, fmt.Errorf("value of mappings object must be string or string[]: %v", v)
				}
				vs = append(vs, s)
			}
			maps[k] = vs
		}
	}

	return maps, nil
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

func mergeMappingsFromFile(dist Mappings, file abspath.AbsPath) error {
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

func mergeMappingsFromPreferredFile(dist Mappings, parent abspath.AbsPath, name string) error {
	root := parent.Join(name)
	if _, err := os.Stat(root.String()); err == nil {
		return mergeMappingsFromFile(dist, root)
	} else if !os.IsNotExist(err) {
		return err
	}

	dotfiles := parent.Join(".dotfiles").Join(name)
	if _, err := os.Stat(dotfiles.String()); err == nil {
		return mergeMappingsFromFile(dist, dotfiles)
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

func isUnixLikePlatform(platform string) bool {
	return platform == "linux" || platform == "darwin"
}

func GetMappingsForPlatform(platform string, parent abspath.AbsPath) (Mappings, error) {
	m := Mappings{}
	partial := rawPartialMappings{}

	if isUnixLikePlatform(platform) {
		if err := mergeMappingsFromDefault(m, unixLikePlatformName); err != nil {
			return nil, err
		}
	}
	if err := mergeMappingsFromDefault(m, platform); err != nil {
		return nil, err
	}

	if err := mergeMappingsFromPreferredFile(m, parent, "mappings.yaml"); err != nil {
		return nil, err
	}
	if err := mergePartialMappingsFromPreferredFile(partial, parent, "mappings.yaml"); err != nil {
		return nil, err
	}

	if isUnixLikePlatform(platform) {
		if err := mergeMappingsFromPreferredFile(m, parent, fmt.Sprintf("mappings_%s.yaml", unixLikePlatformName)); err != nil {
			return nil, err
		}
		if err := mergePartialMappingsFromPreferredFile(partial, parent, fmt.Sprintf("mappings_%s.yaml", unixLikePlatformName)); err != nil {
			return nil, err
		}
	}
	if err := mergeMappingsFromPreferredFile(m, parent, fmt.Sprintf("mappings_%s.yaml", platform)); err != nil {
		return nil, err
	}
	if err := mergePartialMappingsFromPreferredFile(partial, parent, fmt.Sprintf("mappings_%s.yaml", platform)); err != nil {
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

	return m, nil
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

func link(from, to abspath.AbsPath, dry bool) (bool, error) {
	if _, err := os.Stat(from.String()); err != nil {
		return false, nil
	}

	if _, err := os.Lstat(to.String()); err == nil {
		// Target already exists. Skipped.
		fmt.Printf("Exist: '%s' -> '%s'\n", from, to.String())
		return true, nil
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
	created := false
	for f, tos := range maps {
		from := dir.Join(filepath.FromSlash(f))
		for _, to := range tos {
			linked, err := link(from, to, dry)
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
	created := false
	for _, f := range specified {
		if tos, ok := maps[f]; ok {
			from := dir.Join(filepath.FromSlash(f))
			for _, to := range tos {
				linked, err := link(from, to, dry)
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
