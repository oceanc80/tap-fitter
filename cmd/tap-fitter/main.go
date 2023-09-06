package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/operator-framework/operator-registry/alpha/template/composite"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type TapFitterTemplate struct {
	withCompositeTemplate bool
	compositePath         string
	catalogPath           string
	compositeSpec         *composite.Template
	writer                io.Writer
}

func newRootCmd() (*cobra.Command, error) {
	var (
		withCompositeTemplate bool
		compositePath         string
		catalogPath           string
	)
	var rootCmd = &cobra.Command{
		Short: "tap-fitter",
		Long:  `tap-fitter reads a composite template to prepare a repository for a catalog production pipeline`,

		RunE: func(cmd *cobra.Command, args []string) error {
			if debug, _ := cmd.Flags().GetBool("debug"); debug {
				logrus.SetLevel(logrus.DebugLevel)
			}
			if cmd.Flag("with-composite-template").Changed {
				withCompositeTemplate = true
			}
			p := TapFitterTemplate{withCompositeTemplate: withCompositeTemplate, compositePath: compositePath, catalogPath: catalogPath}
			if err := p.validateFlags(); err != nil {
				return err
			}
			err := p.ingestComposite(cmd)
			if err != nil {
				return err
			}

			if err := p.generateDockerfile(); err != nil {
				return err
			}
			if err := p.generateDevfile(); err != nil {
				return err
			}

			return nil
		},
	}
	f := rootCmd.Flags()
	f.Bool("debug", false, "enable debug logging")
	f.Bool("with-composite-template", false, "leverage composite template to generate pipeline fit (defaults to false; requires composite-path and catalog-path if set)")
	f.StringVar(&compositePath, "composite-path", "", "the path to the composite template used for configuration (required if with-composite-template is set)")
	f.StringVar(&catalogPath, "catalog-path", "", "the path/URL to the catalog template used for configuration (required if with-composite-template is set)")
	// cmd.Flags().StringVar(&minEdge, "minimum-edge", "", "the channel edge to be used as the lower bound of the set of edges composing the upgrade graph; default is to include all edges")

	if err := f.MarkHidden("debug"); err != nil {
		logrus.Panic(err.Error())
	}

	return rootCmd, nil
}

func main() {
	cmd, err := newRootCmd()
	if err != nil {
		logrus.Panic(err)
	}
	if err := cmd.Execute(); err != nil {
		logrus.Panic(err)
	}
}

func (p *TapFitterTemplate) validateFlags() error {
	fmt.Printf(">>> flags: withCompositeTemplate(%v), catalogPath(%q), compositePath(%q)\n", p.withCompositeTemplate, p.catalogPath, p.compositePath)
	if p.withCompositeTemplate && (p.catalogPath == "" || p.compositePath == "") {
		return fmt.Errorf("with-composite-template requires also providing 'catalog-path' and 'composite-path' flags")
	}
	return nil
}

func (p *TapFitterTemplate) ingestComposite(cmd *cobra.Command) error {

	cacheDir, err := os.MkdirTemp("", "tap-fitter-")
	if err != nil {
		return err
	}

	reg, err := containerdregistry.NewRegistry(
		containerdregistry.WithCacheDir(cacheDir),
	)
	if err != nil {
		return err
	}
	defer reg.Destroy()

	// operator author's 'composite.yaml' file
	compositeReader, err := os.Open(p.compositePath)
	if err != nil {
		return fmt.Errorf("opening composite config file %q: %v", p.compositePath, err)
	}
	defer compositeReader.Close()

	// catalog maintainer's 'catalogs.yaml' file
	tempCatalog, err := composite.FetchCatalogConfig(p.catalogPath, http.DefaultClient)
	if err != nil {
		return err
	}
	defer tempCatalog.Close()

	template := composite.NewTemplate(
		composite.WithCatalogFile(tempCatalog),
		composite.WithContributionFile(compositeReader),
		composite.WithRegistry(reg),
	)
	if err := template.Parse(); err != nil {
		return err
	}

	p.compositeSpec = template

	return nil
}

func (p *TapFitterTemplate) generateDockerfile() error {

	// TODO: extract enough cmd.generate.dockerfile to trigger the alpha.action.generate_dockerfile based on template fields
	return nil
}

func (p *TapFitterTemplate) generateDevfile() error {
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
