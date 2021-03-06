package golang

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Jumpscale/go-raml/codegen/commons"
	"github.com/Jumpscale/go-raml/codegen/resource"
	"github.com/Jumpscale/go-raml/raml"
)

const (
	langGo = "go"
)

// Client represents a Golang client
type Client struct {
	apiDef         *raml.APIDefinition
	Name           string
	BaseURI        string
	libraries      map[string]*raml.Library
	PackageName    string
	RootImportPath string
	Services       map[string]*ClientService
	TargetDir      string
	libsRootURLs   []string
}

// NewClient creates a new Golang client
func NewClient(apiDef *raml.APIDefinition, packageName, rootImportPath, targetDir string,
	libsRootURLs []string) (Client, error) {
	// rootImportPath only needed if we use libraries
	if rootImportPath == "" && len(apiDef.Libraries) > 0 {
		return Client{}, fmt.Errorf("--import-path can't be empty when we use libraries")
	}

	rootImportPath = setRootImportPath(rootImportPath, targetDir)
	globRootImportPath = rootImportPath
	globAPIDef = apiDef
	globLibRootURLs = libsRootURLs

	services := map[string]*ClientService{}
	for k, v := range apiDef.Resources {
		rd := resource.New(apiDef, commons.NormalizeURITitle(apiDef.Title), packageName)
		rd.GenerateMethods(&v, langGo, newServerMethod, newGoClientMethod)
		services[k] = &ClientService{
			rootEndpoint: k,
			PackageName:  packageName,
			Methods:      rd.Methods,
		}
	}
	client := Client{
		apiDef:         apiDef,
		Name:           escapeIdentifier(commons.NormalizeURI(apiDef.Title)),
		BaseURI:        apiDef.BaseURI,
		libraries:      apiDef.Libraries,
		PackageName:    packageName,
		RootImportPath: rootImportPath,
		Services:       services,
		TargetDir:      targetDir,
		libsRootURLs:   libsRootURLs,
	}

	if strings.Index(client.BaseURI, "{version}") > 0 {
		client.BaseURI = strings.Replace(client.BaseURI, "{version}", apiDef.Version, -1)
	}
	return client, nil
}

// Generate generates all Go client files
func (gc Client) Generate() error {
	// helper package
	gh := goramlHelper{
		packageName: "goraml",
		packageDir:  "goraml",
	}
	if err := gh.generate(gc.TargetDir); err != nil {
		return err
	}

	// generate struct
	if err := generateAllStructs(gc.apiDef, gc.TargetDir, gc.PackageName); err != nil {
		return err
	}

	// libraries
	if err := generateLibraries(gc.libraries, gc.TargetDir, gc.libsRootURLs); err != nil {
		return err
	}

	if err := gc.generateHelperFile(gc.TargetDir); err != nil {
		return err
	}

	if err := gc.generateSecurity(gc.TargetDir); err != nil {
		return err
	}

	if err := gc.generateServices(gc.TargetDir); err != nil {
		return err
	}
	return gc.generateClientFile(gc.TargetDir)
}

// generate Go client helper
func (gc *Client) generateHelperFile(dir string) error {
	fileName := filepath.Join(dir, "/client_utils.go")
	return commons.GenerateFile(gc, "./templates/client_utils_go.tmpl", "client_utils_go", fileName, false)
}

func (gc *Client) generateServices(dir string) error {
	for _, s := range gc.Services {
		sort.Sort(resource.ByEndpoint(s.Methods))
		if err := commons.GenerateFile(s, "./templates/client_service_go.tmpl", "client_service_go", s.filename(dir), false); err != nil {
			return err
		}
	}
	return nil
}

// generate security related files
// it currently only supports itsyou.online oauth2
func (c *Client) generateSecurity(dir string) error {
	for name, ss := range c.apiDef.SecuritySchemes {
		if v, ok := ss.Settings["accessTokenUri"]; ok {
			ctx := map[string]string{
				"ClientName":     c.Name,
				"AccessTokenURI": fmt.Sprintf("%v", v),
				"PackageName":    c.PackageName,
			}
			filename := filepath.Join(dir, "oauth2_client_"+name+".go")
			if err := commons.GenerateFile(ctx, "./templates/oauth2_client_go.tmpl", "oauth2_client_go", filename, true); err != nil {
				return err
			}
		}
	}
	return nil
}

// generate Go client lib file
func (gc *Client) generateClientFile(dir string) error {
	fileName := filepath.Join(dir, "/client_"+strings.ToLower(gc.Name)+".go")
	return commons.GenerateFile(gc, "./templates/client_go.tmpl", "client_go", fileName, false)
}
