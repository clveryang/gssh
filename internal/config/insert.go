package config

import (
	"bytes"
	"fmt"
	"os"

	"github.com/clveryang/gssh/internal/model"
	"gopkg.in/yaml.v3"
)

// Insert adds a host to the YAML file while preserving comments, key order and
// formatting. A plain Marshal of the whole config would silently discard every
// comment the user wrote, so the document is edited as a node tree instead.
//
// group == "" appends to the top-level `hosts:` list. A named group that does
// not exist yet is created.
func Insert(h *model.Host, group string) error {
	path := Path()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		data = []byte("hosts: []\n")
	} else if err != nil {
		return err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	// An empty file unmarshals to a zero node; start a fresh mapping.
	if len(doc.Content) == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{
			{Kind: yaml.MappingNode, Tag: "!!map"},
		}}
	}
	rootMap := doc.Content[0]
	if rootMap.Kind != yaml.MappingNode {
		return fmt.Errorf("%s: expected a mapping at the top level", path)
	}

	var entry yaml.Node
	if err := entry.Encode(h); err != nil {
		return err
	}

	seq, err := targetSequence(rootMap, group)
	if err != nil {
		return err
	}
	seq.Kind = yaml.SequenceNode
	seq.Tag = "!!seq"
	seq.Style = 0 // a flow-style `[]` placeholder must become a block list
	seq.Content = append(seq.Content, &entry)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2) // Marshal defaults to 4 and would reindent the whole file
	if err := enc.Encode(&doc); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o600)
}

// targetSequence returns the hosts sequence to append to, creating the keys or
// the group as needed.
func targetSequence(rootMap *yaml.Node, group string) (*yaml.Node, error) {
	if group == "" {
		return findOrCreateKey(rootMap, "hosts"), nil
	}

	groups := findOrCreateKey(rootMap, "groups")
	groups.Kind = yaml.SequenceNode
	groups.Tag = "!!seq"
	groups.Style = 0

	for _, g := range groups.Content {
		if g.Kind != yaml.MappingNode {
			continue
		}
		if name := mapValue(g, "name"); name != nil && name.Value == group {
			return findOrCreateKey(g, "hosts"), nil
		}
	}

	// New group: {name: <group>, hosts: []}
	gm := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	gm.Content = append(gm.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "name"},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: group},
	)
	groups.Content = append(groups.Content, gm)
	return findOrCreateKey(gm, "hosts"), nil
}

func mapValue(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func findOrCreateKey(m *yaml.Node, key string) *yaml.Node {
	if v := mapValue(m, key); v != nil {
		return v
	}
	val := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		val,
	)
	return val
}
