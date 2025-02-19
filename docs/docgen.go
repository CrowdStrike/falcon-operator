package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"golang.org/x/exp/slices"
)

var (
	// The directory containing the source files to be parsed.
	// This is set by the -srcdir flag.
	srcDir string

	// The directory to which the generated files will be written.
	// This is set by the -outdir flag.
	outDir string

	// Generate local resource docs for a product e.g. OpenShift
	resourceDocs []string

	// k8s distro e.g. OpenShift
	distros []string

	fileDisAllowList = []string{".go", ".tmpl", ".tpl"}
)

type Config struct {
	Distro          string
	KubeCmd         string
	DistroResources bool
}

func main() {
	conf := Config{}

	flag.Usage = func() {
		log.Printf("Usage: %s [flags]\n", path.Base(os.Args[0]))
		flag.PrintDefaults()
	}

	flag.StringVar(&srcDir, "src", "", "The directory containing the source files to be parsed")
	flag.StringVar(&outDir, "out", "", "The directory to which the generated files will be written")
	flag.Func("include-resource-docs", "Generate local resource docs for a k8s distro e.g. OpenShift", func(flagValue string) error {
		resourceDocs = strings.Fields(flagValue)
		return nil
	})
	flag.Func("distros", "Generate local resource docs for a k8s distro e.g. OpenShift", func(flagValue string) error {
		distros = strings.Fields(flagValue)
		return nil
	})
	flag.Bool("help", false, "Show this help message")
	flag.Parse()

	if flag.NArg() > 0 {
		log.Print("Unexpected positional arguments")
		flag.Usage()
		os.Exit(1)
	}

	if flag.NFlag() == 0 || flag.NFlag() < 2 || flag.Lookup("help").Value.String() == "true" {
		flag.Usage()
		os.Exit(1)
	}

	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		log.Fatalf("Directory does not exist: %s", srcDir)
	}

	files, err := getFiles(srcDir, outDir)
	if err != nil {
		log.Fatal(err)
	}

	templateFiles := getTemplates(files)
	nonTplFiles := getNonTemplates(files)

	for _, path := range templateFiles {
		name := filepath.Base(path)
		baseName := filepath.Base(path)
		pathOnly := strings.Replace(strings.TrimSuffix(path, baseName), srcDir, "", 1)
		resourcePath := strings.TrimSuffix(baseName, ".md.tmpl")
		mdName := baseName
		if strings.Contains(path, ".md.tmpl") {
			mdName = strings.TrimSuffix(strings.Replace(baseName, baseName, "README.md.tmpl", 1), ".tmpl")
		}

		// Parse template files
		tpl, err := template.New(name).Funcs(sprig.FuncMap()).Funcs(customFuncs()).ParseFiles(templateFiles...)
		if err != nil {
			log.Fatal(err)
		}

		// Process distro specific templates
		for _, distro := range distros {
			conf.Distro = distro

			if distro == "openshift" {
				conf.KubeCmd = "oc"
			} else {
				conf.KubeCmd = "kubectl"
			}

			// Process deployment templates for distro specific directories
			if strings.Contains(path, "deployment") && !strings.Contains(path, "falcondeployment") {
				tempPath := fmt.Sprintf("%s/%s%s/%s", outDir, pathOnly, distro, name)
				outFile := strings.TrimSuffix(tempPath, ".tmpl")
				err = createFileUsingTemplate(tpl, outFile, conf)
				if err != nil {
					log.Fatal(err)
				}
			}

			// Add resources to distro specific directories when specified
			if slices.Contains(resourceDocs, distro) && strings.Contains(path, "resources") && !strings.Contains(path, "falcondeployment") {
				conf.DistroResources = true
				dstPath := fmt.Sprintf("%s/%s/%s/%s/%s/%s", outDir, "deployment", distro, "resources", resourcePath, mdName)
				err = createFileUsingTemplate(tpl, dstPath, conf)
				if err != nil {
					log.Fatal(err)
				}
			}
		}

		// Process non-distro specific templates
		if strings.Contains(path, "resources") && !strings.Contains(path, "templates") {
			conf.KubeCmd = "kubectl"
			conf.DistroResources = false
			conf.Distro = ""
			outFile := fmt.Sprintf("%s/%s/%s/%s", outDir, "resources", resourcePath, mdName)
			err = createFileUsingTemplate(tpl, outFile, conf)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	for _, file := range nonTplFiles {
		err = copy(file, fmt.Sprintf("%s/%s", outDir, strings.Replace(file, srcDir, "", 1)))
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getFiles(srcDir string, outDir string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(srcDir, func(path string, file fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !file.IsDir() && strings.Split(path, "/")[0] != outDir {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func getTemplates(files []string) []string {
	templates := []string{}
	for _, file := range files {
		if strings.Contains(file, ".tmpl") {
			templates = append(templates, file)
		}
	}
	return templates
}

func getNonTemplates(files []string) []string {
	templates := []string{}
	for _, file := range files {
		if !slices.Contains(fileDisAllowList, filepath.Ext(file)) {
			templates = append(templates, file)
		}
	}
	return templates
}

func createFileUsingTemplate(t *template.Template, filename string, data interface{}) error {
	if err := os.MkdirAll(path.Dir(filename), os.ModePerm); err != nil {
		log.Fatal(err)
	}

	log.Printf("Creating file: %s\n", filename)
	f, err := os.Create(filename) //#nosec
	if err != nil {
		return err
	}
	defer f.Close()

	err = t.Execute(f, data)
	if err != nil {
		return err
	}

	return nil
}

func copy(src, dst string) error {
	if err := os.MkdirAll(path.Dir(dst), os.ModePerm); err != nil {
		log.Fatal(err)
	}

	source, err := os.Open(src) //#nosec
	if err != nil {
		return err
	}
	defer source.Close()

	log.Printf("Creating file: %s\n", dst)
	destination, err := os.Create(dst) //#nosec
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}

	return nil
}

func customFuncs() template.FuncMap {
	return map[string]interface{}{
		// Add custom functions here
	}
}
