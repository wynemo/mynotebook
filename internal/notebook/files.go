// Package notebook manages plain Markdown files. It has no UI dependencies.
package notebook

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const MaxDocument = 8 << 20

var ErrConflict = errors.New("文件已被外部修改，请重新载入或另存为，避免覆盖")

type Node struct {
	Path, Name string
	Dir        bool
	Open       bool
	Loaded     bool
	Children   []*Node
}
type Document struct{ Path, Text, Version string }
type Match struct {
	Path             string
	Line, Start, End int
	Text             string
}
type SearchResult struct {
	Matches   []Match
	Skipped   int
	Truncated bool
}
type Session struct {
	Roots        []string `json:"roots"`
	LastFile     string   `json:"lastFile"`
	SidebarWidth float32  `json:"sidebarWidth"`
}

func IsMarkdown(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".mdown":
		return true
	}
	return false
}
func Canonical(path string) (string, error) {
	p, e := filepath.Abs(path)
	if e != nil {
		return "", e
	}
	return filepath.EvalSymlinks(p)
}
func Root(path string) (*Node, error) {
	p, e := Canonical(path)
	if e != nil {
		return nil, e
	}
	st, e := os.Stat(p)
	if e != nil {
		return nil, e
	}
	if !st.IsDir() {
		return nil, errors.New("请选择文件夹")
	}
	n := &Node{Path: p, Name: filepath.Base(p), Dir: true, Open: true}
	return n, Load(n)
}
func Load(n *Node) error {
	entries, e := os.ReadDir(n.Path)
	if e != nil {
		return e
	}
	children := []*Node{}
	for _, d := range entries {
		if strings.HasPrefix(d.Name(), ".") || d.Type()&os.ModeSymlink != 0 || d.Name() == "node_modules" {
			continue
		}
		if d.IsDir() || IsMarkdown(d.Name()) {
			children = append(children, &Node{Path: filepath.Join(n.Path, d.Name()), Name: d.Name(), Dir: d.IsDir()})
		}
	}
	sort.Slice(children, func(i, j int) bool {
		a, b := children[i], children[j]
		if a.Dir != b.Dir {
			return a.Dir
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	n.Children = children
	n.Loaded = true
	return nil
}
func Digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func Read(path string) (Document, error) {
	st, e := os.Stat(path)
	if e != nil {
		return Document{}, e
	}
	if !st.Mode().IsRegular() || st.Size() > MaxDocument {
		return Document{}, errors.New("只能打开 8 MiB 以内的文本文件")
	}
	b, e := os.ReadFile(path)
	if e != nil {
		return Document{}, e
	}
	if !utf8.Valid(b) || bytes.IndexByte(b, 0) >= 0 {
		return Document{}, errors.New("文件不是 UTF-8 文本")
	}
	return Document{path, string(b), Digest(b)}, nil
}

// Save checks the last observed content before replacing the file. A blank
// version means a new file and must never overwrite an existing file.
func Save(path, text, version string) (string, error) {
	b := []byte(text)
	if len(b) > MaxDocument {
		return "", errors.New("文档超过 8 MiB")
	}
	mode := fs.FileMode(0644)
	if st, e := os.Lstat(path); e == nil {
		if !st.Mode().IsRegular() {
			return "", errors.New("目标不是普通文件")
		}
		old, e := os.ReadFile(path)
		if e != nil {
			return "", e
		}
		if version == "" || Digest(old) != version {
			return "", ErrConflict
		}
		mode = st.Mode().Perm()
	} else if !os.IsNotExist(e) {
		return "", e
	} else if version != "" {
		return "", ErrConflict
	}
	if e := AtomicWrite(path, b, mode); e != nil {
		return "", e
	}
	return Digest(b), nil
}
func AtomicWrite(path string, b []byte, mode fs.FileMode) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".mynotebook-*")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if e = f.Chmod(mode); e != nil {
		f.Close()
		return e
	}
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(tmp, path)
}
func AttachImage(document string, b []byte) (string, error) {
	if document == "" {
		return "", errors.New("请先保存笔记，再粘贴图片")
	}
	if len(b) > 20<<20 {
		return "", errors.New("图片超过 20 MiB")
	}
	cfg, e := png.DecodeConfig(bytes.NewReader(b))
	if e != nil {
		return "", errors.New("剪贴板图片不是有效的 PNG")
	}
	if cfg.Width > 16384 || cfg.Height > 16384 || int64(cfg.Width)*int64(cfg.Height) > 64000000 {
		return "", errors.New("图片分辨率过大")
	}
	dir := filepath.Join(filepath.Dir(document), ".assets")
	if e = os.MkdirAll(dir, 0755); e != nil {
		return "", e
	}
	st, e := os.Lstat(dir)
	if e != nil {
		return "", e
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("附件目录不能是符号链接")
	}
	name := "image-" + Digest(b)[:20] + ".png"
	path := filepath.Join(dir, name)
	// Open exclusively: never follow an attachment symlink or replace a file.
	f, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(e) {
		existing, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(existing, b) {
			return "", errors.New("附件名称冲突")
		}
		return ".assets/" + name, nil
	}
	if e != nil {
		return "", e
	}
	_, e = f.Write(b)
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		os.Remove(path)
		return "", e
	}
	return ".assets/" + name, nil
}
func Search(ctx context.Context, roots []string, query string, limit int) (SearchResult, error) {
	out := SearchResult{Matches: []Match{}}
	query = strings.ToLower(query)
	if strings.TrimSpace(query) == "" {
		return out, nil
	}
	if limit <= 0 {
		limit = 500
	}
	seen := map[string]bool{}
	for _, root := range roots {
		e := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if e := ctx.Err(); e != nil {
				return e
			}
			if err != nil {
				out.Skipped++
				return nil
			}
			if d.IsDir() {
				if path != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules") {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 || !IsMarkdown(path) || seen[path] {
				return nil
			}
			seen[path] = true
			doc, e := Read(path)
			if e != nil {
				out.Skipped++
				return nil
			}
			offset := 0
			for i, line := range strings.Split(doc.Text, "\n") {
				if strings.Contains(strings.ToLower(line), query) {
					if len(out.Matches) >= limit {
						out.Truncated = true
						return fs.SkipAll
					}
					snippet := strings.TrimSpace(line)
					r := []rune(snippet)
					if len(r) > 180 {
						snippet = string(r[:180]) + "…"
					}
					out.Matches = append(out.Matches, Match{path, i + 1, offset, offset + len(line), snippet})
				}
				offset += len(line) + 1
			}
			return nil
		})
		if e != nil {
			return out, e
		}
		if out.Truncated {
			break
		}
	}
	return out, nil
}
func LoadSession(path string) (Session, error) {
	b, e := os.ReadFile(path)
	if os.IsNotExist(e) {
		return Session{SidebarWidth: 270}, nil
	}
	if e != nil {
		return Session{}, e
	}
	var s Session
	e = json.Unmarshal(b, &s)
	if s.SidebarWidth < 180 {
		s.SidebarWidth = 270
	}
	return s, e
}
func SaveSession(path string, s Session) error {
	b, e := json.MarshalIndent(s, "", "  ")
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	if e = AtomicWrite(path, b, 0600); e != nil {
		return fmt.Errorf("保存工作区: %w", e)
	}
	return nil
}
