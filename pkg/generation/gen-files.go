package generation

import (
	"fmt"
	"net/http"
	"os"

	"github.com/operator-framework/operator-registry/alpha/template/composite"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	"github.com/sirupsen/logrus"
)

type TapFitterTemplate struct {
	withCompositeTemplate bool
	compositePath         string
	catalogPath           string
	compositeSpec         *composite.Template
	writer                io.Writer
}

func (p *TapFitterTemplate) GenerateDevfile() error {
	if err := p.validateFlags(); err != nil {
		return err
	}

	t, err := template.New("devfile").Parse(devfileTmpl)
	if err != nil {
		// The template is hardcoded in the binary, so if
		// there is a parse error, it was a programmer error.
		panic(err)
	}
	return t.Execute(p.writer, p)
}

const devfileTmpl = `schemaVersion: 2.2.0
metadata:
  name: {{.Name}}
  displayName: {{.Name}}
  description: 'File based catalog'
  language: fbc
  provider: {{.Provider}}
components:
  - name: image-build
    image:
      imageName: fbc:latest
      dockerfile:
        uri: {{.IndexDir}}.Dockerfile
        buildContext: {{.BuildCTX}}
commands:
  - id: build-image
    apply:
      component: image-build
`
