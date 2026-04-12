package core

import (
	"fmt"
	"html"
	"strings"
)

// executeSEOMethod handles SEO class methods
func (r *Runtime) executeSEOMethod(instance *Instance, method string, args []interface{}) interface{} {
	if r.SEO == nil {
		r.SEO = &SEOData{
			Meta: make(map[string]string),
			OG:   make(map[string]string),
		}
	}

	switch method {
	case "title":
		if len(args) >= 1 {
			r.SEO.Title = fmt.Sprintf("%v", args[0])
		}
		return nil
	case "description":
		if len(args) >= 1 {
			r.SEO.Description = fmt.Sprintf("%v", args[0])
		}
		return nil
	case "keywords":
		if len(args) >= 1 {
			switch v := args[0].(type) {
			case string:
				r.SEO.Keywords = strings.Split(v, ",")
			case []interface{}:
				for _, item := range v {
					r.SEO.Keywords = append(r.SEO.Keywords, fmt.Sprintf("%v", item))
				}
			}
		}
		return nil
	case "canonical":
		if len(args) >= 1 {
			r.SEO.Canonical = fmt.Sprintf("%v", args[0])
		}
		return nil
	case "og":
		if len(args) >= 2 {
			prop := fmt.Sprintf("%v", args[0])
			content := fmt.Sprintf("%v", args[1])
			r.SEO.OG[prop] = content
		}
		return nil
	case "meta":
		if len(args) >= 2 {
			name := fmt.Sprintf("%v", args[0])
			content := fmt.Sprintf("%v", args[1])
			r.SEO.Meta[name] = content
		}
		return nil
	case "render":
		return r.RenderSEOTags()
	}

	return nil
}

// RenderSEOTags generates the HTML block for <head>
func (r *Runtime) RenderSEOTags() string {
	if r.SEO == nil {
		return ""
	}

	var sb strings.Builder

	// Title
	if r.SEO.Title != "" {
		sb.WriteString(fmt.Sprintf("<title>%s</title>\n", html.EscapeString(r.SEO.Title)))
	}

	// Description
	if r.SEO.Description != "" {
		sb.WriteString(fmt.Sprintf("<meta name=\"description\" content=\"%s\">\n", html.EscapeString(r.SEO.Description)))
	}

	// Keywords
	if len(r.SEO.Keywords) > 0 {
		sb.WriteString(fmt.Sprintf("<meta name=\"keywords\" content=\"%s\">\n", html.EscapeString(strings.Join(r.SEO.Keywords, ", "))))
	}

	// Canonical
	if r.SEO.Canonical != "" {
		sb.WriteString(fmt.Sprintf("<link rel=\"canonical\" href=\"%s\">\n", html.EscapeString(r.SEO.Canonical)))
	}

	// Standard Meta
	for name, content := range r.SEO.Meta {
		sb.WriteString(fmt.Sprintf("<meta name=\"%s\" content=\"%s\">\n", html.EscapeString(name), html.EscapeString(content)))
	}

	// Open Graph
	for prop, content := range r.SEO.OG {
		// Ensure og: prefix
		p := prop
		if !strings.HasPrefix(p, "og:") {
			p = "og:" + p
		}
		sb.WriteString(fmt.Sprintf("<meta property=\"%s\" content=\"%s\">\n", html.EscapeString(p), html.EscapeString(content)))
	}

	// Automatic OG Title/Desc/URL if missing
	if _, ok := r.SEO.OG["og:title"]; !ok && r.SEO.Title != "" {
		sb.WriteString(fmt.Sprintf("<meta property=\"og:title\" content=\"%s\">\n", html.EscapeString(r.SEO.Title)))
	}
	if _, ok := r.SEO.OG["og:description"]; !ok && r.SEO.Description != "" {
		sb.WriteString(fmt.Sprintf("<meta property=\"og:description\" content=\"%s\">\n", html.EscapeString(r.SEO.Description)))
	}

	// Twitter Card (Default)
	sb.WriteString("<meta name=\"twitter:card\" content=\"summary_large_image\">\n")

	return sb.String()
}

// executeSitemapMethod handles Sitemap class methods
func (r *Runtime) executeSitemapMethod(instance *Instance, method string, args []interface{}) interface{} {
	switch method {
	case "add":
		if len(args) >= 1 {
			entry := SitemapEntry{
				URL:        fmt.Sprintf("%v", args[0]),
				LastMod:    "",
				ChangeFreq: "weekly",
				Priority:   0.5,
			}
			if len(args) >= 2 {
				entry.LastMod = fmt.Sprintf("%v", args[1])
			}
			if len(args) >= 3 {
				entry.ChangeFreq = fmt.Sprintf("%v", args[2])
			}
			if len(args) >= 4 {
				// Priority
				if p, ok := args[3].(float64); ok {
					entry.Priority = p
				}
			}
			r.SitemapEntries = append(r.SitemapEntries, entry)
		}
		return nil
	case "generate":
		// Try to get baseUrl from current request if available
		baseUrl := ""
		if reqVal, ok := r.Variables["$__request"]; ok {
			if reqInstance, ok := reqVal.(*Instance); ok {
				scheme := "http"
				if s, ok := reqInstance.Fields["_scheme"].(string); ok {
					scheme = s
				}
				host := "localhost"
				if h, ok := reqInstance.Fields["_host"].(string); ok {
					host = h
				}
				baseUrl = scheme + "://" + host
			}
		}
		return r.GenerateSitemapXML(baseUrl)
	}
	return nil
}

// GenerateSitemapXML builds the full sitemap.xml buffer
func (r *Runtime) GenerateSitemapXML(baseUrl string) string {
	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	sb.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")

	// 1. Automatic Routes from routes.joss
	// We iterate over GET routes
	if getRoutes, ok := r.Routes["GET"]; ok {
		for path, infoVal := range getRoutes {
			if info, ok := infoVal.(map[string]interface{}); ok {
				// Filter logic:
				// - Must be from "routes"
				// - Must NOT have middleware
				// - Must NOT be dynamic (no : or {)
				source, _ := info["source"].(string)
				middleware, _ := info["middleware"].([]string)

				if source == "routes" && len(middleware) == 0 {
					if !strings.Contains(path, ":") && !strings.Contains(path, "{") {
						r.writeSitemapEntry(&sb, path, "", "weekly", 0.8, baseUrl)
					}
				}
			}
		}
	}

	// 2. Manual Entries
	for _, entry := range r.SitemapEntries {
		r.writeSitemapEntry(&sb, entry.URL, entry.LastMod, entry.ChangeFreq, entry.Priority, baseUrl)
	}

	sb.WriteString("</urlset>")
	return sb.String()
}

func (r *Runtime) writeSitemapEntry(sb *strings.Builder, urlPath, lastMod, freq string, priority float64, dynamicBase string) {
	// Root URL resolution logic:
	// 1. Dynamic Base (from request)
	// 2. APP_URL (from environment)
	// 3. http://localhost (fallback)
	appUrl := dynamicBase
	if appUrl == "" {
		if val, ok := r.Env["APP_URL"]; ok {
			appUrl = strings.TrimSuffix(val, "/")
		} else {
			appUrl = "http://localhost"
		}
	}

	fullUrl := appUrl + urlPath
	if !strings.HasPrefix(urlPath, "/") && !strings.HasPrefix(urlPath, "http") {
		fullUrl = appUrl + "/" + urlPath
	} else if strings.HasPrefix(urlPath, "http") {
		fullUrl = urlPath
	}

	sb.WriteString("  <url>\n")
	sb.WriteString(fmt.Sprintf("    <loc>%s</loc>\n", html.EscapeString(fullUrl)))
	if lastMod != "" {
		sb.WriteString(fmt.Sprintf("    <lastmod>%s</lastmod>\n", html.EscapeString(lastMod)))
	}
	sb.WriteString(fmt.Sprintf("    <changefreq>%s</changefreq>\n", html.EscapeString(freq)))
	sb.WriteString(fmt.Sprintf("    <priority>%.1f</priority>\n", priority))
	sb.WriteString("  </url>\n")
}
