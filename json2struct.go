package jsongen

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type StructGen struct {
	Name     string
	Data     map[string]any
	Meta     map[string]string
	Children []*StructGen
}

func (s *StructGen) Json2Struct(jsondata []byte, writer io.Writer, options *Options) error {
	if options.Unmarshal == nil {
		options.Unmarshal = json.Unmarshal
	}
	if err := options.Unmarshal(jsondata, &s.Data); err != nil {
		return err
	}
	for k, v := range s.Data {
		s.Meta[k] = s.loadMeta(k, v)
	}
	buf := &bytes.Buffer{}
	buf.WriteString("package " + options.Package + "\n\n")
	if err := s.render(buf); err != nil {
		return err
	}
	bytes, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	_, err = writer.Write(bytes)
	return err
}

func (s *StructGen) loadMeta(key string, val any) string {
	switch underlying := val.(type) {
	case bool:
		return "bool"
	case float64:
		return "float64"
	case string:
		return "string"
	case map[string]any:
		child := &StructGen{Name: Title(key), Data: underlying, Meta: map[string]string{}}
		for k, v := range child.Data {
			child.Meta[k] = child.loadMeta(k, v)
		}
		s.Children = append(s.Children, child)
		return Title(key)
	case []any:
		if len(underlying) == 0 {
			return "[]any"
		} else {
			return "[]" + s.loadMeta(key, underlying[0])
		}
	default:
		return "any"
	}
}

var _ embed.FS

//go:embed stub.tmpl
var templateText string

var tmpl = func() *template.Template {
	funcs := template.FuncMap{
		"Title": Title,
	}
	temp, err := template.New("").Funcs(funcs).Parse(templateText)
	if err != nil {
		panic(err)
	}
	return temp
}()

func (s *StructGen) render(w io.Writer) error {
	for _, n := range s.Children {
		err := n.render(w)
		if err != nil {
			return err
		}
	}
	return tmpl.Execute(w, s)
}

func Title(name string) string {
	return strings.ToUpper(name)[0:1] + name[1:]
}

type Options struct {
	Package   string
	Type      string
	Unmarshal func(data []byte, v any) error
}

// convert json to struct and write to writer
func Json2Struct(jsontext []byte, writer io.Writer, options *Options) error {
	if options == nil || options.Package == "" || options.Type == "" {
		return fmt.Errorf("missing package or type options")
	}
	sg := &StructGen{Meta: map[string]string{}, Name: options.Type}
	return sg.Json2Struct(jsontext, writer, options)
}

// convert jsonfiles in jsondir to go structs and put the in dir
// if options have no value, Package and Type will be guessed from folder and file name
func Json2StructDir(jsondir string, dir string, options *Options) error {
	jsondir = filepath.Clean(jsondir)
	dir = filepath.Clean(dir)
	return filepath.WalkDir(jsondir, func(path string, d fs.DirEntry, _ error) error {
		if filepath.Ext(path) == ".json" && !d.IsDir() {
			if options != nil {
				options.Type = "" // clean type as name contention
			}
			// convert some/path/name.json => new/path/name.json
			conv1 := strings.Replace(path, jsondir, dir, 1)
			// convert new/path/name.json => new/path/name.go
			conv2 := strings.TrimSuffix(conv1, ".json") + ".go"
			return Json2StructFile(path, conv2, options)
		}
		return nil
	})
}

// convert jsonfile to gofile
// if options doesn't have value, Package and Type will be guessed as per folder and file name
func Json2StructFile(jsonfile string, gofile string, options *Options) error {
	dir, name := filepath.Split(gofile)
	if options == nil {
		options = &Options{}
	}
	if options.Type == "" {
		name = strings.TrimSuffix(name, filepath.Ext(name))
		options.Type = Title(name)
	}
	if options.Package == "" {
		base := filepath.Base(dir)
		if base == "." || base == ".." {
			abs, err := filepath.Abs(dir)
			if err != nil {
				return fmt.Errorf("Get absolute path: %s. %w", dir, err)
			}
			dir = abs
			options.Package = filepath.Base(dir)
		} else {
			options.Package = base
		}
	}
	err := os.MkdirAll(dir, 0o666)
	if err != nil {
		return fmt.Errorf("Make dir: %s. %w", dir, err)
	}
	file, err := os.Create(gofile)
	if err != nil {
		return fmt.Errorf("Create file: %s, %w", gofile, err)
	}
	defer file.Close()
	data, err := os.ReadFile(jsonfile)
	if err != nil {
		return fmt.Errorf("ReadFile: %s. %w", jsonfile, err)
	}
	return Json2Struct(data, file, options)
}
