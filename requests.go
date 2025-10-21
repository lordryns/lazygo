package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type GolangIndexPackage struct {
	Path      string
	Version   string
	Timestamp string
}

var GolangPackages []GolangIndexPackage

func fetchPackagesFromIndex() {
	var response, err = http.Get("https://index.golang.org/index?include=all")
	if err != nil {
		return
	}

	var decoder = json.NewDecoder(response.Body)
	for decoder.More() {
		var pkg GolangIndexPackage
		if err := decoder.Decode(&pkg); err == nil {
			GolangPackages = append(GolangPackages, pkg)
		}
	}

	defer response.Body.Close()
}

func filterGolangPackages(name string) []GolangIndexPackage {
	var pkgs []GolangIndexPackage

	for _, pkg := range GolangPackages {
		if strings.Contains(pkg.Path, name) {
			pkgs = append(pkgs, pkg)
		}
	}

	return pkgs
}
