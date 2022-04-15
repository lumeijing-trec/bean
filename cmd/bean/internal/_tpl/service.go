package services

import (
	"context"

	// "github.com/retail-ai-inc/bean/trace"
	{{if .RepoExists}}"{{.ProjectObject.PkgPath}}/repositories"{{end}}
)

type {{.ServiceNameUpper}}Service interface {
	{{.ServiceNameUpper}}ServiceExampleFunc(ctx context.Context) (string, error)
}

{{if .RepoExists}}type {{.ServiceNameLower}}Service struct {
	{{.ServiceNameLower}}Repository repositories.{{.ServiceNameUpper}}Repository
}{{else}}type {{.ServiceNameLower}}Service struct {}{{end}}

{{if .RepoExists}}func New{{.ServiceNameUpper}}Service({{.ServiceNameLower}}Repo repositories.{{.ServiceNameUpper}}Repository) *{{.ServiceNameLower}}Service {
	return &{{.ServiceNameLower}}Service{{"{\n\t\t"}}{{.ServiceNameLower}}Repository: {{.ServiceNameLower}}Repo,{{"\n\t}"}}
}{{else}}func New{{.ServiceNameUpper}}Service() *{{.ServiceNameLower}}Service {
	return &{{.ServiceNameLower}}Service{{"{}\n}"}}{{end}}

func (service *{{.ServiceNameLower}}Service) {{.ServiceNameUpper}}ServiceExampleFunc(ctx context.Context) (string, error) {
	// IMPORTANT: If you wanna trace the performance of your handler function then uncomment following 3 lines
	// finish := trace.Start(ctx, "http.service")
	// defer finish()
	return "{{.ServiceNameUpper}}Service", nil
}
