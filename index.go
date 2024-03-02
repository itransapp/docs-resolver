package main

import (
	"github.com/BurntSushi/toml"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

type IndexModulePage struct {
	Path    string   `toml:"path" json:"path"`
	Title   string   `toml:"title" json:"title"`
	Desc    string   `toml:"description" json:"description"`
	Authors []string `toml:"authors" json:"authors"`
}

type IndexModule struct {
	Path  string            `toml:"path" json:"path"`
	Name  string            `toml:"name" json:"name"`
	Desc  string            `toml:"description" json:"description"`
	Pages []IndexModulePage `toml:"pages" json:"pages"`
}

type Index struct {
	Modules []string                `toml:"modules" json:"modules"`
	Pages   map[string]*IndexModule `toml:"pages" json:"pages"`
}

type Page struct {
	_path string `toml:"-"`

	Naming  string   `toml:"naming"`
	Title   string   `toml:"title"`
	Desc    string   `toml:"desc"`
	Authors []string `toml:"authors"`
}

type PageIndex struct {
	_naming string           `toml:"-"`
	_path   string           `toml:"-"`
	_index  map[string]*Page `toml:"-"`

	Module string  `toml:"module"`
	Desc   string  `toml:"desc"`
	Pages  []*Page `toml:"pages"`
}

func (idx *Index) Append(pidx *PageIndex) {
	if _, found := slices.BinarySearch(idx.Modules, pidx._naming); found {
		return
	}

	idx.Modules = append(idx.Modules, pidx._naming)

	idxModule := &IndexModule{
		Path:  pidx._path,
		Name:  pidx.Module,
		Desc:  pidx.Desc,
		Pages: make([]IndexModulePage, 0, len(pidx.Pages)),
	}
	for _, page := range pidx.Pages {
		idxModule.Pages = append(idxModule.Pages, IndexModulePage{
			Path:    page._path,
			Title:   page.Title,
			Desc:    page.Desc,
			Authors: page.Authors,
		})
	}
	idx.Pages[pidx._naming] = idxModule
}

func ReadPageIndex(path string) (*PageIndex, error) {
	var idx PageIndex
	if _, err := toml.DecodeFile(path, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

type PathMetadata struct {
	Path  string
	XPath string
	Ext   int
}

func deepResolver(module string, mp []*PathMetadata, idx *Index) error {
	pidx := &PageIndex{
		_index: make(map[string]*Page),
		Pages:  make([]*Page, 0, len(mp)-1),
	}
	for _, metadata := range mp {
		if metadata.Ext == 1 {
			pidx._naming = module
			pidx._path, _ = filepath.Split(metadata.Path)
			_pidx, err := ReadPageIndex(metadata.Path)
			if err != nil {
				return err
			}
			pidx.Module = _pidx.Module
			pidx.Desc = _pidx.Desc
			pidx.Pages = _pidx.Pages
			for _, page := range _pidx.Pages {
				pidx._index[strings.ToLower(page.Naming)] = page
			}
		}
	}

	for _, metadata := range mp {
		if metadata.Ext != 0 {
			continue
		}

		if _, found := pidx._index[metadata.Path]; found {
			continue
		}
		_, naming := filepath.Split(metadata.Path)

		if naming == "index.md" {
			pidx._index[""]._path = metadata.Path
			continue
		}

		if page, found := pidx._index[strings.ToLower(naming[:len(naming)-len(filepath.Ext(naming))])]; found {
			page._path = metadata.Path
		}
	}

	idx.Append(pidx)
	return nil
}

func resolver(perfixs map[string][]*PathMetadata, idx *Index) error {
	for module, metadata := range perfixs {
		if err := deepResolver(module, metadata, idx); err != nil {
			return nil
		}
	}
	return nil
}

func WalkWithResolver(base string) (*Index, error) {
	var (
		idx = &Index{
			Modules: make([]string, 0, 24),
			Pages:   make(map[string]*IndexModule),
		}

		prefixs = make(map[string][]*PathMetadata)
	)

	if err := filepath.Walk(base, func(p string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(info.Name())
		switch ext {
		case ".md", ".toml":
			prefix := path.Clean(filepath.Dir(p)[len(base)-1:])
			if prefix[0] == '/' {
				prefix = prefix[1:]
			}

			if prefixs[prefix] == nil {
				prefixs[prefix] = make([]*PathMetadata, 0, 8)
			}

			prefixs[prefix] = append(prefixs[prefix], &PathMetadata{
				Path:  p,
				XPath: prefix,
				Ext: func() int {
					if ext == ".md" {
						return 0
					}
					return 1
				}(),
			})

		}
		return nil
	}); err != nil {
		return nil, err
	}

	return idx, resolver(prefixs, idx)
}
