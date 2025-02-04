package util

import (
	"html/template"
	"log"
)

var Templates *template.Template

// LoadTemplates parses all HTML files in the templates folder at startup.
func LoadTemplates() {
	var err error
	Templates, err = template.ParseGlob("web/templates/*.html")
	if err != nil {
		log.Fatalf("Error loading templates: %v", err)
	}
}
