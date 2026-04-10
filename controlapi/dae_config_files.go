package controlapi

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/daeuniverse/dae/common"
	"github.com/daeuniverse/dae/config"
	"github.com/daeuniverse/dae/pkg/config_parser"
)

type daeConfigFileNode struct {
	DaeConfigFile
	order int
}

func loadDaeConfigDocumentBundle(entryPath string) (DaeConfigDocument, error) {
	documents, err := collectDaeConfigFileNodes(entryPath)
	if err != nil {
		return DaeConfigDocument{}, err
	}
	files := make([]DaeConfigFile, 0, len(documents))
	for _, document := range documents {
		files = append(files, document.DaeConfigFile)
	}
	entry := files[0]
	return DaeConfigDocument{
		Path:      entry.Path,
		Content:   entry.Content,
		Documents: files,
	}, nil
}

func buildDaeConfigUpdateBundle(entryPath string, update DaeConfigDocument) ([]daeConfigFileNode, error) {
	current, err := collectDaeConfigFileNodes(entryPath)
	if err != nil {
		return nil, err
	}

	byPath := make(map[string]daeConfigFileNode, len(current))
	for _, document := range current {
		byPath[document.Path] = document
	}

	if len(update.Documents) == 0 {
		entry := byPath[entryPath]
		entry.Content = normalizeConfigDocumentText(update.Content)
		entry.Missing = false
		byPath[entryPath] = entry
		return daeConfigNodesInOrder(byPath), nil
	}

	maxOrder := len(current)
	for index, document := range update.Documents {
		targetPath := strings.TrimSpace(document.Path)
		if targetPath == "" {
			if document.Entry || index == 0 {
				targetPath = entryPath
			} else {
				return nil, fmt.Errorf("config document path is required")
			}
		}
		cleanPath := filepath.Clean(targetPath)
		if err := ensureConfigDocumentPath(cleanPath, entryPath); err != nil {
			return nil, err
		}

		node, exists := byPath[cleanPath]
		if !exists {
			relativePath, err := configDocumentRelativePath(cleanPath, filepath.Dir(entryPath))
			if err != nil {
				return nil, err
			}
			node = daeConfigFileNode{
				DaeConfigFile: DaeConfigFile{
					Path:         cleanPath,
					RelativePath: relativePath,
				},
				order: maxOrder,
			}
			maxOrder += 1
		}

		node.Path = cleanPath
		node.RelativePath = document.RelativePath
		if node.RelativePath == "" {
			relativePath, err := configDocumentRelativePath(cleanPath, filepath.Dir(entryPath))
			if err != nil {
				return nil, err
			}
			node.RelativePath = relativePath
		}
		node.Content = normalizeConfigDocumentText(document.Content)
		node.Entry = cleanPath == entryPath || document.Entry
		node.Missing = false
		byPath[cleanPath] = node
	}

	nodes := daeConfigNodesInOrder(byPath)
	for index := range nodes {
		nodes[index].Entry = nodes[index].Path == entryPath
	}
	return nodes, nil
}

func collectDaeConfigFileNodes(entryPath string) ([]daeConfigFileNode, error) {
	rootDir := filepath.Dir(entryPath)
	visited := map[string]struct{}{}
	stack := map[string]struct{}{}
	documents := make([]daeConfigFileNode, 0, 4)

	var dfs func(path string) error
	dfs = func(path string) error {
		cleanPath := filepath.Clean(path)
		if _, seen := visited[cleanPath]; seen {
			return nil
		}
		if _, seen := stack[cleanPath]; seen {
			return fmt.Errorf("%w: %s", config.ErrCircularInclude, cleanPath)
		}
		if err := ensureConfigDocumentPath(cleanPath, entryPath); err != nil {
			return err
		}

		contentBytes, err := os.ReadFile(cleanPath)
		if err != nil {
			return fmt.Errorf("read config file %s: %w", cleanPath, err)
		}
		content := normalizeConfigDocumentText(string(contentBytes))
		relativePath, err := configDocumentRelativePath(cleanPath, rootDir)
		if err != nil {
			return err
		}

		visited[cleanPath] = struct{}{}
		stack[cleanPath] = struct{}{}
		documents = append(documents, daeConfigFileNode{
			DaeConfigFile: DaeConfigFile{
				Path:         cleanPath,
				RelativePath: relativePath,
				Content:      content,
				Entry:        cleanPath == entryPath,
			},
			order: len(documents),
		})

		includeTargets, err := parseIncludedConfigTargets(content, rootDir)
		delete(stack, cleanPath)
		if err != nil {
			return fmt.Errorf("read include targets from %s: %w", cleanPath, err)
		}
		for _, target := range includeTargets {
			if target.Missing {
				if _, seen := visited[target.Path]; seen {
					continue
				}
				visited[target.Path] = struct{}{}
				documents = append(documents, daeConfigFileNode{
					DaeConfigFile: DaeConfigFile{
						Path:         target.Path,
						RelativePath: target.RelativePath,
						Content:      "",
						Missing:      true,
					},
					order: len(documents),
				})
				continue
			}
			if err := dfs(target.Path); err != nil {
				return err
			}
		}
		return nil
	}

	if err := dfs(entryPath); err != nil {
		return nil, err
	}
	return documents, nil
}

func daeConfigNodesInOrder(nodes map[string]daeConfigFileNode) []daeConfigFileNode {
	ordered := make([]daeConfigFileNode, 0, len(nodes))
	for _, node := range nodes {
		ordered = append(ordered, node)
	}
	slices.SortFunc(ordered, func(left, right daeConfigFileNode) int {
		switch {
		case left.Path == right.Path:
			return 0
		case left.Entry && !right.Entry:
			return -1
		case !left.Entry && right.Entry:
			return 1
		case left.order != right.order:
			if left.order < right.order {
				return -1
			}
			return 1
		case left.Path < right.Path:
			return -1
		default:
			return 1
		}
	})
	return ordered
}

func validateDaeConfigDocuments(documents []daeConfigFileNode) error {
	if len(documents) == 0 {
		return fmt.Errorf("no config documents to validate")
	}

	entryPath := documents[0].Path
	rootDir := filepath.Dir(entryPath)
	tempRoot, err := os.MkdirTemp(rootDir, ".dae-controller-config-*")
	if err != nil {
		return fmt.Errorf("create temp config dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempRoot) }()

	for _, document := range documents {
		relativePath := document.RelativePath
		if relativePath == "" {
			relativePath, err = configDocumentRelativePath(document.Path, rootDir)
			if err != nil {
				return err
			}
		}
		tempPath := filepath.Join(tempRoot, relativePath)
		if err := os.MkdirAll(filepath.Dir(tempPath), 0755); err != nil {
			return fmt.Errorf("create temp config directory: %w", err)
		}
		mode := configDocumentMode(document.Path)
		if err := os.WriteFile(tempPath, []byte(document.Content), mode); err != nil {
			return fmt.Errorf("write temp config file %s: %w", tempPath, err)
		}
	}

	entryRelativePath, err := configDocumentRelativePath(entryPath, rootDir)
	if err != nil {
		return err
	}
	return validateDaeConfigFile(filepath.Join(tempRoot, entryRelativePath))
}

func writeDaeConfigDocuments(documents []daeConfigFileNode) error {
	tempPaths := make([]string, 0, len(documents))
	defer func() {
		for _, tempPath := range tempPaths {
			_ = os.Remove(tempPath)
		}
	}()

	for _, document := range documents {
		targetDir := filepath.Dir(document.Path)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("create config directory %s: %w", targetDir, err)
		}
		tmp, err := os.CreateTemp(targetDir, ".dae-controller-*.dae")
		if err != nil {
			return fmt.Errorf("create temp config for %s: %w", document.Path, err)
		}
		tmpPath := tmp.Name()
		tempPaths = append(tempPaths, tmpPath)

		mode := configDocumentMode(document.Path)
		if err := tmp.Chmod(mode); err != nil {
			_ = tmp.Close()
			return fmt.Errorf("set temp config mode for %s: %w", document.Path, err)
		}
		if _, err := tmp.WriteString(document.Content); err != nil {
			_ = tmp.Close()
			return fmt.Errorf("write temp config for %s: %w", document.Path, err)
		}
		if err := tmp.Close(); err != nil {
			return fmt.Errorf("close temp config for %s: %w", document.Path, err)
		}
	}

	for index := len(documents) - 1; index >= 0; index -= 1 {
		if err := os.Rename(tempPaths[index], documents[index].Path); err != nil {
			return fmt.Errorf("replace config file %s: %w", documents[index].Path, err)
		}
	}
	return nil
}

type daeIncludeTarget struct {
	Path         string
	RelativePath string
	Missing      bool
}

func parseIncludedConfigTargets(content string, rootDir string) ([]daeIncludeTarget, error) {
	sections, err := config_parser.Parse(content)
	if err != nil {
		return nil, err
	}

	targets := make([]daeIncludeTarget, 0)
	seen := map[string]struct{}{}
	for _, section := range sections {
		if section.Name != "include" {
			continue
		}
		for _, item := range section.Items {
			param, ok := item.Value.(*config_parser.Param)
			if !ok {
				return nil, fmt.Errorf("unsupported include grammar: %s", item.String(false, false))
			}
			targetPath := param.String(true, false)
			resolvedPath := targetPath
			if !filepath.IsAbs(resolvedPath) {
				resolvedPath = filepath.Join(rootDir, resolvedPath)
			}
			resolvedPath = filepath.Clean(resolvedPath)

			if hasConfigGlob(targetPath) {
				matches, err := filepath.Glob(resolvedPath)
				if err != nil {
					return nil, err
				}
				for _, match := range matches {
					if _, exists := seen[match]; exists {
						continue
					}
					if !strings.HasSuffix(match, ".dae") {
						continue
					}
					info, err := os.Stat(match)
					if err != nil || info.IsDir() {
						continue
					}
					relativePath, err := configDocumentRelativePath(match, rootDir)
					if err != nil {
						return nil, err
					}
					seen[match] = struct{}{}
					targets = append(targets, daeIncludeTarget{
						Path:         match,
						RelativePath: relativePath,
					})
				}
				continue
			}

			relativePath, err := configDocumentRelativePath(resolvedPath, rootDir)
			if err != nil {
				return nil, err
			}
			if _, exists := seen[resolvedPath]; exists {
				continue
			}
			seen[resolvedPath] = struct{}{}

			if _, err := os.Stat(resolvedPath); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					targets = append(targets, daeIncludeTarget{
						Path:         resolvedPath,
						RelativePath: relativePath,
						Missing:      true,
					})
					continue
				}
				return nil, err
			}
			targets = append(targets, daeIncludeTarget{
				Path:         resolvedPath,
				RelativePath: relativePath,
			})
		}
	}
	return targets, nil
}

func ensureConfigDocumentPath(path string, entryPath string) error {
	rootDir := filepath.Dir(entryPath)
	if !strings.HasSuffix(path, ".dae") {
		return fmt.Errorf("invalid config filename %s: must have suffix .dae", path)
	}
	if err := common.EnsureFileInSubDir(path, rootDir); err != nil {
		return fmt.Errorf("failed to validate config path %s: %w", path, err)
	}
	return nil
}

func configDocumentRelativePath(path string, rootDir string) (string, error) {
	relativePath, err := filepath.Rel(rootDir, path)
	if err != nil {
		return "", fmt.Errorf("resolve config relative path for %s: %w", path, err)
	}
	relativePath = filepath.Clean(relativePath)
	if relativePath == "." {
		return filepath.Base(path), nil
	}
	return relativePath, nil
}

func configDocumentMode(path string) os.FileMode {
	if stat, err := os.Stat(path); err == nil {
		return stat.Mode().Perm()
	}
	return 0600
}

func normalizeConfigDocumentText(content string) string {
	return strings.ReplaceAll(content, "\r\n", "\n")
}

func hasConfigGlob(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

func validateDaeConfigFile(path string) error {
	merger := config.NewMerger(path)
	sections, _, err := merger.Merge()
	if err != nil {
		return err
	}
	if _, err := config.New(sections); err != nil {
		return err
	}
	return nil
}
