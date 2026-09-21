package cmd

import (
	"strings"

	"github.com/clveryang/gssh/internal/model"
	"gopkg.in/yaml.v3"
)

func yamlPreview(c *model.Config) (string, error) {
	var sb strings.Builder
	enc := yaml.NewEncoder(&sb)
	enc.SetIndent(2)
	if err := enc.Encode(c); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return sb.String(), nil
}
