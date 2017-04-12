// Sugar wrapper around html/template to preload templates and provide simple inheritance.

/*
This package allows to preload a set of templates from a directory (or memory, or ...) while
configuring inheritance and providing a simple expansion mechanism.

Basics

The starting point is that you have a set of templates. For example, you may have a "base.tmpl"
file defining the structure of your html page, with a navigation bar, some js, and a few columns.
In those columns, the content is filled by expanding other templates.

You now have, for example, a "home.tmpl" page, that by using "base.tmpl", shows the main page
of your web site. Or a "news.tmpl" or "projects.tmpl", that by always using the skeleton in the
"base.tmpl" and same layout, show different things.

Traditionally, you'd have to configure the template inheritance mechanisms manually, or by
invoking methods on the templates themselves.

With this package, if you need inheritance, you just need to use a simple naming convention.
Following the previous example, you'd have a directory with the following files:

    // Free standing, base file. The template name used internally is "base".
    base.tmpl

    // Home page, based of "base.tmpl". Internal name is "home", inherits from "base".
    home=base.tmpl

    // News page, named "news", inherits from "base".
    news=base.tmpl

    // Projects page, shows some news and other custom texts.
    // Named "projects", inherits from "news" and "base".
    projects=news,base.tmpl


To make use of those templates, all you have to do in code is:

    // Load and parse - once - all the templates in the "./templates" directory.
    //
    // nil is an optional callback function run on each template. You can use it,
    // for example, to configure your own template functions or delimiters.
    templates, err := NewStaticTemplatesFromDir("./templates", nil)

    // Expand a template into an io.Writer of choice, passing in some data.
    templates.Expand(
        "news", struct { Name, Address, Zip }{ "Mr. Bean", "987 Broadway", "00111" }, writer)

If you need to do more esoteric stuff on the template objects themselves, you can
just use the Get() method to get back a template.Template.

In general, this package is a pretty thin layer on top of html/templates that tries not
to get in your way, while saving you a few lines of code on every project, and moving
some flexibility from code (having to manually configure inheritance) to external config
(your web developer being free of changing how inheritance works by just naming files).

This package is also friendly to go-binddata users (https://github.com/jteeuwen/go-bindata)
or any other mechanism you may like to provide templates.
*/
package templates

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type StaticTemplates struct {
	templates map[string]*template.Template
}

// Function to read a template.
//
// The parameter is the name of the template, it returns the content of a template or an error.
type TemplateReader func(string) ([]byte, error)

// Function to configure a template.
//
// Useful to set specific parameters on each template loaded. For example, to define custom
// functions or delimiters. It is passed in the loaded template, and is expected to return
// the modified template, or an error.
type TemplateConfigurer func(*template.Template) (*template.Template, error)

// Creates a new StaticTemplates object from a map of templates.
//
// The map uses the name of the template as a key, exactly as if the template was read from disk,
// while the value is the content of the template as a byte array.
// Returns a StaticTemplates object, or an error.
func NewStaticTemplatesFromMap(templates map[string][]byte, configurer TemplateConfigurer) (*StaticTemplates, error) {
	names := make([]string, len(templates))
	index := 0
	for key, _ := range templates {
		names[index] = key
		index += 1
	}
	return NewStaticTemplates(names, configurer, func(filename string) ([]byte, error) {
		content, ok := templates[filename]
		if !ok {
			return []byte{}, os.ErrNotExist
		}

		return content, nil
	})
}

// Creates a new StaticTemplates object from a directory on the file system.
//
// Does not descend into subdirectories, does not expect anything else but templates in this directory.
func NewStaticTemplatesFromDir(directory string, configurer TemplateConfigurer) (*StaticTemplates, error) {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(files))
	for i, file := range files {
		names[i] = file.Name()
	}

	return NewStaticTemplates(names, configurer, func(filename string) ([]byte, error) {
		return ioutil.ReadFile(path.Join(directory, filename))
	})
}

// Creates a new StaticTemplates object from a list of filenames, and a function to read them.
func NewStaticTemplates(
	files []string,
	configureTemplate TemplateConfigurer,
	getFileContent TemplateReader) (*StaticTemplates, error) {

	type BaseData struct {
		content []byte
		parents []string
	}

	bases := make(map[string]BaseData)
	for _, file := range files {
		if file[0] == '.' {
			continue
		}

		content, err := getFileContent(file)
		if err != nil {
			return nil, err
		}

		// For a file like: "my-template=foo,bar,baz.tmpl" we want to have:
		// - basename (used as key) "my-template"
		// - parents "foo,bar,baz"
		filename := path.Base(file)
		filename = filename[0 : len(filename)-len(filepath.Ext(filename))]

		split := strings.SplitN(filename, "=", 2)
		basename := split[0]
		parents := []string{}
		if len(split) > 1 && len(split[1]) > 0 {
			parents = strings.Split(split[1], ",")
		}
		bases[basename] = BaseData{content, parents}
	}

	result := &StaticTemplates{make(map[string]*template.Template)}

	var err error
	for name, data := range bases {
		tpl := template.New(name)
		if configureTemplate != nil {
			tpl, err = configureTemplate(tpl)
			if err != nil {
				return nil, err
			}
		}

		tpl, err := tpl.Parse(string(data.content))
		if err != nil {
			return nil, err
		}

		for i := len(data.parents) - 1; i >= 0; i-- {
			base := data.parents[i]
			data, ok := bases[base]
			if !ok {
				return nil, &TemplateNotFoundError{base}
			}

			tpl, err = tpl.Parse(string(data.content))
			if err != nil {
				return nil, err
			}
		}
		result.templates[name] = tpl
	}

	return result, nil
}

// Error returned when no template can be found by the specified name.
type TemplateNotFoundError struct {
	// The name of the template that could not be found.
	Template string
}

func (t *TemplateNotFoundError) Error() string {
	return fmt.Sprintf("Could not find template %s", t.Template)
}

// Expands the specified template, using the supplied data into the supplied writer.
func (self *StaticTemplates) Expand(name string, data interface{}, writer io.Writer) error {
	tpl, ok := self.templates[name]
	if !ok {
		return &TemplateNotFoundError{name}
	}

	return tpl.ExecuteTemplate(writer, "start", data)
}

// Returns the template object corresponding to a name, or nil.
func (self *StaticTemplates) Get(name string) *template.Template {
	return self.templates[name]
}
