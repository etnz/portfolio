package docs

// this file handles
// documentation topics.

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed *.md
var docs embed.FS

// GetTopic returns the content of a documentation topic.
func GetTopic(topic string) (string, error) {
	if topic == "*" {
		topics, err := GetAllTopics()
		if err != nil {
			return "", err
		}
		return GetTopics(topics...)
	}

	path := topic + ".md"

	content, err := docs.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("topic %q not found: %w", topic, err)
	}

	return string(content), nil
}

// GetTopics returns the content of multiple documentation topics concatenated together.
func GetTopics(topics ...string) (string, error) {
	var b bytes.Buffer
	for _, topic := range topics {
		if topic == "*" {
			// expand the star
			allTopics, err := GetAllTopics()
			if err != nil {
				return "", err
			}
			for _, t := range allTopics {
				content, err := GetTopic(t)
				if err != nil {
					return "", err
				}
				b.WriteString(content)
				b.WriteString("\n")
			}
			continue
		}
		content, err := GetTopic(topic)
		if err != nil {
			return "", err
		}
		b.WriteString(content)
		b.WriteString("\n")
	}
	return b.String(), nil
}

// GetAllTopics returns a list of all available documentation topics.
func GetAllTopics() ([]string, error) {
	var topics []string
	err := fs.WalkDir(docs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if base == "readme" {
			return nil
		}
		topics = append(topics, base)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(topics)
	return topics, nil
}
