package main

import (
	"context"
	"fmt"

	"github.com/tomyan/hubcap/internal/chrome"
)

func cmdInfo(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.GetPageInfo(ctx, target.ID)
	})
}

type SourceResult struct {
	HTML string `json:"html"`
}

func cmdSource(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		html, err := client.GetPageSource(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return SourceResult{HTML: html}, nil
	})
}

type LinkInfo struct {
	Href string `json:"href"`
	Text string `json:"text"`
}

type LinksResult struct {
	Links []LinkInfo `json:"links"`
}

func cmdLinks(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		result, err := client.Eval(ctx, target.ID, `
			Array.from(document.querySelectorAll('a[href]')).map(a => ({
				href: a.href,
				text: a.textContent.trim()
			}))
		`)
		if err != nil {
			return nil, err
		}

		// Parse the result
		links := []LinkInfo{}
		if arr, ok := result.Value.([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					link := LinkInfo{
						Href: fmt.Sprintf("%v", m["href"]),
						Text: fmt.Sprintf("%v", m["text"]),
					}
					links = append(links, link)
				}
			}
		}

		return LinksResult{Links: links}, nil
	})
}

// MetaInfo represents a single meta tag.
type MetaInfo struct {
	Name       string `json:"name,omitempty"`
	Property   string `json:"property,omitempty"`
	Content    string `json:"content,omitempty"`
	Charset    string `json:"charset,omitempty"`
	HTTPEquiv  string `json:"httpEquiv,omitempty"`
}

// MetaResult is returned by the meta command.
type MetaResult struct {
	Tags []MetaInfo `json:"tags"`
}

func cmdMeta(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		result, err := client.Eval(ctx, target.ID, `
			Array.from(document.querySelectorAll('meta')).map(m => ({
				name: m.getAttribute('name') || '',
				property: m.getAttribute('property') || '',
				content: m.getAttribute('content') || '',
				charset: m.getAttribute('charset') || '',
				httpEquiv: m.getAttribute('http-equiv') || ''
			}))
		`)
		if err != nil {
			return nil, err
		}

		// Parse the result
		tags := []MetaInfo{}
		if arr, ok := result.Value.([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					tag := MetaInfo{
						Name:      fmt.Sprintf("%v", m["name"]),
						Property:  fmt.Sprintf("%v", m["property"]),
						Content:   fmt.Sprintf("%v", m["content"]),
						Charset:   fmt.Sprintf("%v", m["charset"]),
						HTTPEquiv: fmt.Sprintf("%v", m["httpEquiv"]),
					}
					tags = append(tags, tag)
				}
			}
		}

		return MetaResult{Tags: tags}, nil
	})
}

func cmdScripts(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		return client.GetScripts(ctx, target.ID)
	})
}

// ImagesResult is returned by the images command.
type ImagesResult struct {
	Images []chrome.ImageInfo `json:"images"`
	Count  int             `json:"count"`
}

func cmdImages(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		images, err := client.GetImages(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return ImagesResult{Images: images, Count: len(images)}, nil
	})
}

// TableInfo represents a single table.
type TableInfo struct {
	ID      string     `json:"id,omitempty"`
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// TablesResult is returned by the tables command.
type TablesResult struct {
	Tables []TableInfo `json:"tables"`
}

func cmdTables(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		result, err := client.Eval(ctx, target.ID, `
			Array.from(document.querySelectorAll('table')).map(table => {
				const headers = Array.from(table.querySelectorAll('thead th, thead td, tr:first-child th')).map(th => th.textContent.trim());
				const bodyRows = table.querySelectorAll('tbody tr');
				const rows = Array.from(bodyRows.length > 0 ? bodyRows : table.querySelectorAll('tr')).map(tr => {
					// Skip header row if no tbody
					if (bodyRows.length === 0 && tr.querySelector('th')) return null;
					return Array.from(tr.querySelectorAll('td, th')).map(cell => cell.textContent.trim());
				}).filter(r => r !== null);
				return {
					id: table.id || '',
					headers: headers,
					rows: rows
				};
			})
		`)
		if err != nil {
			return nil, err
		}

		// Parse the result
		tables := []TableInfo{}
		if arr, ok := result.Value.([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					tableInfo := TableInfo{
						ID:      fmt.Sprintf("%v", m["id"]),
						Headers: []string{},
						Rows:    [][]string{},
					}

					// Parse headers
					if headers, ok := m["headers"].([]interface{}); ok {
						for _, h := range headers {
							tableInfo.Headers = append(tableInfo.Headers, fmt.Sprintf("%v", h))
						}
					}

					// Parse rows
					if rows, ok := m["rows"].([]interface{}); ok {
						for _, row := range rows {
							if cells, ok := row.([]interface{}); ok {
								rowData := []string{}
								for _, cell := range cells {
									rowData = append(rowData, fmt.Sprintf("%v", cell))
								}
								tableInfo.Rows = append(tableInfo.Rows, rowData)
							}
						}
					}

					tables = append(tables, tableInfo)
				}
			}
		}

		return TablesResult{Tables: tables}, nil
	})
}

type FormsResult struct {
	Forms []chrome.FormInfo `json:"forms"`
	Count int            `json:"count"`
}

func cmdForms(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		forms, err := client.GetForms(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return FormsResult{Forms: forms, Count: len(forms)}, nil
	})
}

// FramesResult is returned by the frames command.
type FramesResult struct {
	Frames []chrome.FrameInfo `json:"frames"`
	Count  int             `json:"count"`
}

func cmdFrames(cfg *Config) int {
	return withClientTarget(cfg, func(ctx context.Context, client *chrome.Client, target *chrome.TargetInfo) (interface{}, error) {
		frames, err := client.GetFrames(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		return FramesResult{Frames: frames, Count: len(frames)}, nil
	})
}
