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
	"unicode"
)

type StructGen struct {
	Name     string
	Data     map[string]any
	Meta     map[string]string
	Children []*StructGen
	Template *template.Template
}

func (s *StructGen) Json2Struct(jsondata []byte, writer io.Writer, options *Options) error {
	if options.Unmarshal == nil {
		if options.UseJsonNumber {
			options.Unmarshal = func(data []byte, v any) error {
				dec := json.NewDecoder(bytes.NewReader(data))
				dec.UseNumber()
				return dec.Decode(v)
			}
		} else {
			options.Unmarshal = json.Unmarshal
		}
	}
	if err := options.Unmarshal(jsondata, &s.Data); err != nil {
		return err
	}
	for k, v := range s.Data {
		s.Meta[k] = s.loadMeta(k, v, options)
	}
	buf := &bytes.Buffer{}
	buf.WriteString("package " + options.Package + "\n\n")
	if options.UseJsonNumber {
		buf.WriteString(`import (`)
		buf.WriteString("\n")
		buf.WriteString(`"encoding/json"`)
		buf.WriteString("\n")
		buf.WriteString(`)`)
		buf.WriteString("\n\n")
		buf.WriteString("var _ json.Number")
		buf.WriteString("\n\n")
	}

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

func (s *StructGen) loadMeta(key string, val any, options *Options) string {
	switch underlying := val.(type) {
	case bool:
		return "bool"
	case float64:
		return "float64"
	case string:
		return "string"
	case json.Number:
		return "json.Number"
	case map[string]any:
		typeName := options.Rename(key, nil)
		child := &StructGen{Name: typeName, Data: underlying, Meta: map[string]string{}, Template: s.Template}
		for k, v := range child.Data {
			child.Meta[k] = child.loadMeta(k, v, options)
		}
		s.Children = append(s.Children, child)
		return typeName
	case []any:
		if len(underlying) == 0 {
			return "[]any"
		} else {
			keyp := options.Rename(key, map[string]any{"plural-slice": true})
			return "[]" + s.loadMeta(keyp, underlying[0], options)
		}
	default:
		return "any"
	}
}

var _ embed.FS

//go:embed stub.tmpl
var templateText string

func (s *StructGen) render(w io.Writer) error {
	for _, n := range s.Children {
		err := n.render(w)
		if err != nil {
			return err
		}
	}
	return s.Template.Execute(w, s)
}

func NormalizeName(name string) string {
	rename := strings.Builder{}
	for _, v := range []rune(name) {
		if (v <= unicode.MaxASCII && unicode.IsLetter(v)) || unicode.IsNumber(v) || v == '_' {
			rename.WriteRune(v)
		}
	}
	return rename.String()
}

func Title(name string, ctx map[string]any) string {
	var words []string
	for _, v := range strings.Split(name, "_") {
		if len(v) == 1 {
			words = append(words, strings.ToUpper(v))
		} else if len(v) > 1 {
			words = append(words, strings.ToUpper(v)[0:1]+v[1:])
		}
	}
	return strings.Join(words, "")
}

type Options struct {
	Package       string
	Type          string
	UseJsonNumber bool
	Unmarshal     func(data []byte, v any) error
	Rename        func(name string, ctx map[string]any) string
}

// convert json to struct and write to writer
func Json2Struct(jsontext []byte, writer io.Writer, options *Options) error {
	if options == nil || options.Package == "" || options.Type == "" {
		return fmt.Errorf("missing package or type options")
	}
	if options.Rename == nil {
		options.Rename = Title
	}
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"Rename": options.Rename,
	}).Parse(templateText)
	if err != nil {
		return err
	}
	sg := &StructGen{
		Meta:     map[string]string{},
		Name:     options.Type,
		Template: tmpl,
	}
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
		options.Type = NormalizeName(name)
	}
	if options.Package == "" {
		base := filepath.Base(dir)
		if base == "." || base == ".." {
			abs, err := filepath.Abs(dir)
			if err != nil {
				return fmt.Errorf("Get absolute path: %s. %w", dir, err)
			}
			dir = abs
			options.Package = NormalizeName(filepath.Base(dir))
		} else {
			options.Package = NormalizeName(base)
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
