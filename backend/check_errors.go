package main

import (
	"fmt"
	"os"

	"golang.org/x/tools/go/packages"
)

func main() {
	f, _ := os.Create("diag.txt")
	defer f.Close()
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo}
	pkgs, err := packages.Load(cfg, "./internal/generator")
	if err != nil {
		fmt.Fprintf(f, "load: %v\n", err)
		os.Exit(1)
	}
	for _, p := range pkgs {
		for _, e := range p.Errors {
			fmt.Fprintf(f, "%s\n", e)
		}
	}
	fmt.Fprintf(f, "Done.\n")
}
