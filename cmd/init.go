// Copyright The RAI Inc.
// The RAI Authors
package cmd

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/cobra"
)

var (
	validationRule = `A module path must satisfy the following requirements:

	1. The path must consist of one or more path elements separated by slashes (/, U+002F). It must not begin or end with a slash.
	2. Each path element is a non-empty string made of up ASCII letters, ASCII digits, and limited ASCII punctuation (-, ., _).
	3. A path element may not begin or end with a dot (., U+002E).
	4. The leading path element (up to the first slash, if any), by convention a domain name, must contain only lower-case ASCII letters, ASCII digits, dots (., U+002E), and dashes (-, U+002D); it must contain at least one dot and cannot start with a dash.`

	// initCmd represents the init command
	initCmd = &cobra.Command{
		Use:   "init package_name",
		Short: "Initialize a project in current directory",
		Long: `Init generates all the directories and files structures needed in the current
directory. the suffix of the package_name should match the current directory.`,
		Example: "bean init github.com/retail-ai-inc/bean",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {

			pkgPath := args[0]
			pkgName, err := getProjectName(pkgPath)
			if err != nil {
				log.Fatalln(validationRule)
			}

			wd, err := os.Getwd()
			if err != nil {
				log.Fatalln(err)
			}

			p := &Project{
				Copyright: `// Copyright The RAI Inc.
// The RAI Authors`,
				PkgPath:     pkgPath,
				PkgName:     pkgName,
				RootDir:     wd,
				BeanVersion: rootCmd.Version,
			}

			// Set the relative root path of the internal FS.
			if p.RootFS, err = fs.Sub(InternalFS, "internal/project"); err != nil {
				log.Fatalln(err)
			}

			fmt.Println("initializing " + p.PkgName + "...")
			if err := fs.WalkDir(p.RootFS, ".", p.generateProjectFiles); err != nil {
				log.Fatalln(err)
			}

			fmt.Println("\ninitializing go mod...")
			goModInitCmd := exec.Command("go", "mod", "init", p.PkgPath)
			goModInitCmd.Stdout = os.Stdout
			goModInitCmd.Stderr = os.Stderr
			if err := goModInitCmd.Run(); err != nil {
				log.Fatalln(err)
			}

			fmt.Println("\ntidying go mod...")
			goModTidyCmd := exec.Command("go", "mod", "tidy")
			goModTidyCmd.Stdout = os.Stdout
			goModTidyCmd.Stderr = os.Stderr
			if err := goModTidyCmd.Run(); err != nil {
				log.Fatalln(err)
			}
		},
	}
)

func init() {
	initCmd.DisableFlagsInUseLine = true
	rootCmd.AddCommand(initCmd)
}

func getProjectName(pkgPath string) (string, error) {
	validate := validator.New()

	if errs := validate.Var(pkgPath, "required,max=100,startsnotwith=/,endsnotwith=/"); errs != nil {
		if errs, ok := errs.(validator.ValidationErrors); ok {
			return "", errs
		}
		log.Fatalln(errs)
	}

	pathElements := strings.Split(pkgPath, "/")
	for _, element := range pathElements {
		if errs := validate.Var(element, "required,printascii,excludesall=!\"#$%&'()*+0x2C:;<=>?@[\\]^`{0x7C~},startsnotwith=.,endsnotwith=."); errs != nil {
			if errs, ok := errs.(validator.ValidationErrors); ok {
				return "", errs
			}
			log.Fatalln(errs)
		}
	}

	if len(pathElements) > 1 {
		domain := pathElements[0]
		if errs := validate.Var(domain, "required,max=100,fqdn"); errs != nil {
			if errs, ok := errs.(validator.ValidationErrors); ok {
				return "", errs
			}
			log.Fatalln(errs)
		}
	}

	pkgName := pathElements[len(pathElements)-1]

	return pkgName, nil
}

func (p *Project) generateProjectFiles(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}

	if d.IsDir() {
		// Create the same directory under current directory.
		if err := os.Mkdir(p.RootDir+"/"+path, 0754); err != nil && d.Name() != "project" {
			return err
		}
	} else {
		// Create the files.
		fileName := strings.TrimSuffix(path, ".tpl")
		fmt.Println(fileName)
		file, err := os.Create(p.RootDir + "/" + fileName)
		if err != nil {
			return err
		}
		defer file.Close()

		if fileName == "bean.sh" {
			if err := file.Chmod(0755); err != nil {
				return err
			}
		}

		// Parse the template and write to the files.
		fileTemplate := template.Must(template.ParseFS(p.RootFS, path))
		err = fileTemplate.Execute(file, p)
		if err != nil {
			return err
		}
	}

	return nil
}