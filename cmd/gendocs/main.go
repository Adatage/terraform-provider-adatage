// gendocs writes the registry documentation (docs/) from the resource definitions: go run ./cmd/gendocs
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Adatage/terraform-provider-adatage/internal/provider"
)

func write(path, text string) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		panic(err)
	}
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	for _, r := range provider.Resources() {
		write(filepath.Join(root, "docs", "resources", r.Name+".md"), provider.ResourceDoc(r, provider.Examples["resource."+r.Name]))
	}
	for _, d := range provider.DataSources() {
		write(filepath.Join(root, "docs", "data-sources", d.Name+".md"), provider.DataSourceDoc(d, provider.Examples["data."+d.Name]))
	}
	fmt.Println("docs written")
}
