// SPDX-License-Identifier: Apache-2.0
// Package archdoc verifies that docs/ARCHITECTURE.md stays factually
// consistent with the module's actual source tree. It is generic
// infrastructure and contains no consumer-project-specific knowledge.
package archdoc

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ModuleRoot walks upward from start until it finds a directory with go.mod.
func ModuleRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("archdoc: go.mod not found at or above %s", start)
		}
		dir = parent
	}
}

var (
	pkgPathRe  = regexp.MustCompile(`pkg/[a-zA-Z0-9_]+`)
	goFileRe   = regexp.MustCompile(`[a-zA-Z0-9_]+\.go`)
	typeDeclRe = regexp.MustCompile(`type ([A-Z][A-Za-z0-9]*) (?:struct|interface)`)
	goBlockRe  = regexp.MustCompile("(?s)```go\n(.*?)```")
	methodRe   = regexp.MustCompile(`(?m)^\s+([A-Z][A-Za-z0-9]*)\(`)
	codeTypeRe = regexp.MustCompile(`type ([A-Z][A-Za-z0-9]*) `)
	identRe    = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
)

// Verify checks docs/ARCHITECTURE.md under root against the source tree.
// requiredMentions are substrings that MUST appear in the doc (a curated
// guard for significant exported symbols package-completeness cannot catch).
// Returns a sorted list of human-readable problems; empty means accurate.
func Verify(root string, requiredMentions []string) ([]string, error) {
	docBytes, err := os.ReadFile(filepath.Join(root, "docs", "ARCHITECTURE.md"))
	if err != nil {
		return nil, err
	}
	doc := string(docBytes)
	var problems []string

	for _, p := range uniqueStrings(pkgPathRe.FindAllString(doc, -1)) {
		if !isDir(filepath.Join(root, p)) {
			problems = append(problems, "references missing package directory: "+p)
		}
	}

	goFiles := goFileBasenames(root)
	for _, f := range uniqueStrings(goFileRe.FindAllString(doc, -1)) {
		if !goFiles[f] {
			problems = append(problems, "references non-existent Go file: "+f)
		}
	}

	codeTypes, codeIdents := codeSymbols(root)
	for _, block := range goBlocks(doc) {
		for _, m := range typeDeclRe.FindAllStringSubmatch(block, -1) {
			if !codeTypes[m[1]] {
				problems = append(problems, "doc declares type absent from code: "+m[1])
			}
		}
		for _, m := range methodRe.FindAllStringSubmatch(block, -1) {
			if !codeIdents[m[1]] {
				problems = append(problems, "doc declares method absent from code: "+m[1])
			}
		}
	}

	for _, d := range pkgDirs(root) {
		if !strings.Contains(doc, "pkg/"+d) {
			problems = append(problems, "code package undocumented: pkg/"+d)
		}
	}

	for _, want := range requiredMentions {
		if !strings.Contains(doc, want) {
			problems = append(problems, "required mention missing: "+want)
		}
	}

	sort.Strings(problems)
	return problems, nil
}

func goBlocks(doc string) []string {
	var out []string
	for _, m := range goBlockRe.FindAllStringSubmatch(doc, -1) {
		out = append(out, m[1])
	}
	return out
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func pkgDirs(root string) []string {
	entries, err := os.ReadDir(filepath.Join(root, "pkg"))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

func goFileBasenames(root string) map[string]bool {
	out := map[string]bool{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "vendor") {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".go") {
			out[d.Name()] = true
		}
		return nil
	})
	return out
}

func codeSymbols(root string) (types, idents map[string]bool) {
	types = map[string]bool{}
	idents = map[string]bool{}
	_ = filepath.WalkDir(filepath.Join(root, "pkg"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			b, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			src := string(b)
			for _, m := range codeTypeRe.FindAllStringSubmatch(src, -1) {
				types[m[1]] = true
			}
			for _, id := range identRe.FindAllString(src, -1) {
				idents[id] = true
			}
		}
		return nil
	})
	return types, idents
}
