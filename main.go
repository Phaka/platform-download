package main

import (
	"flag"
	"fmt"
	"github.com/phaka/platform-go"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func main() {
	flag.Parse()

	var oses []platform.OperatingSystem
	for _, path := range flag.Args() {
		os, err := platform.LoadOperatingSystem(path)
		if err != nil {
			fmt.Printf("Error loading operating system: %s\n", err)
			continue
		}
		oses = append(oses, os)
	}

	// ParseFiles
	// ParseGlob

	destinationPathTemplate := `{{ .OS.Name }}/{{if .OS.Release }}{{ .OS.Release }}/{{end}}{{ .OS.Architecture }}/{{ .Base }}`
	tmpl, err := template.New("test").Parse(destinationPathTemplate)
	if err != nil {
		panic(err)
	}

	for _, operatingSystem := range oses {
		fmt.Printf("%s\n", operatingSystem.GetName())
		urls := operatingSystem.GetDownloadURLs()
		if urls != nil {
			for _, url := range urls {
				targetPath, err := getTargetPath(tmpl, operatingSystem, url)
				if err != nil {
					fmt.Printf("Error getting target path: %s\n", err)
					continue
				}

				if pathExists(targetPath) {
					fmt.Printf("  \"%s\" already exists\n", targetPath)
					continue
				}

				err = mkdir(targetPath)
				if err != nil {
					fmt.Printf("Error creating directory: %s\n", err)
					continue
				}

				err = safeDownload(url, targetPath)
				if err != nil {
					fmt.Println(err)
					continue
				}
			}
		}
	}
}

func safeDownload(url, path string) error {
	var err error
	temporaryPath := path + ".download"

	// delete the file in case previous download failed
	deleteFile(temporaryPath)

	// delete the file in case the download process failed
	defer deleteFile(temporaryPath)

	err = download(url, temporaryPath)
	if err != nil {
		return err
	}

	err = os.Rename(temporaryPath, path)
	if err != nil {
		return fmt.Errorf("error renaming file: %w", err)
	}
	return nil
}

func download(url, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file \"%s\": %w", path, err)
	}
	defer out.Close()

	return downloadFile(url, out)
}

func downloadFile(url string, out *os.File) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error downloading file \"%s\": %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error downloading file: %s", resp.Status)
	}

	var written int64
	written, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("error saving file: %w", err)
	}
	fmt.Printf("  %d bytes written\n", written)
	return nil
}

func deleteFile(path string) {
	// check if path exists and then delete it
	if pathExists(path) {
		err := os.Remove(path)
		if err != nil {
			fmt.Printf("Error deleting file: %s\n", err)
		}
	}
}

func mkdir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

func pathExists(targetPath string) bool {
	if _, err := os.Stat(targetPath); err == nil {
		return true
	}
	return false
}

func getTargetPath(tmpl *template.Template, operatingSystem platform.OperatingSystem, url string) (targetPath string, err error) {
	data := map[string]interface{}{
		"OS":        operatingSystem,
		"Base":      filepath.Base(url),
		"Extension": filepath.Ext(url),
	}
	builder := &strings.Builder{}
	err = tmpl.Execute(builder, data)
	if err != nil {
		return "", err
	}
	targetPath = builder.String()
	return targetPath, nil
}
