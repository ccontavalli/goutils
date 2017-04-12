// Adapter for goutils/templates.StaticTemplates to be used in a gin gonic
// router.
//
// Usage is simple. Create your static templates as usual:
//
//         render, err := templates.NewStaticTemplatesFromDir("./templates", nil)
//         [...]
//
// and then typecast into the router:
//
//         router.HTMLRender = (*gtemplates.HTMLRender)(render)
//
// and use as usual.

package gtemplates

import (
        "github.com/ccontavalli/goutils/templates"
        "github.com/gin-gonic/gin/render"
        "html/template"
        "fmt"
        "net/http"
)

type HTMLRender templates.StaticTemplates

type Render struct {
  template *template.Template
  data interface{}
}

func (r *Render) Render(w http.ResponseWriter) error {
  if r.template == nil {
    return fmt.Errorf("Template not found")
  }

  return r.template.ExecuteTemplate(w, "start", r.data)
}

func (self *HTMLRender) Instance(name string, data interface{}) render.Render {
	return &Render{(*templates.StaticTemplates)(self).Get(name), data}
}
