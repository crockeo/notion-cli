package dump

import (
	"encoding/json"
	"os"

	"github.com/jomei/notionapi"

	"github.com/crockeo/notion-cli/commands"
	"github.com/crockeo/notion-cli/config"
	"github.com/crockeo/notion-cli/database"
)

func Dump(config *config.Config, client *notionapi.Client, complete []string) {
	database, err := database.GetSync(config, client)
	commands.Guard(err)

	info := NotionCliInfo{
		Order:      config.Capture.Order,
		Properties: map[string]PropInfo{},
	}
	for propName, propConfig := range database.Properties {
		if _, ok := propConfig.(*notionapi.FormulaPropertyConfig); ok {
			// we don't prompt for Formulas during capture
			// so we don't want to tell people that they exist
			continue
		}

		propInfo := PropInfo{Type: string(propConfig.GetType())}
		if defaultValue, ok := config.Capture.Defaults[propName]; ok {
			propInfo.Default = &defaultValue
		}

		switch propConfig := propConfig.(type) {
		case *notionapi.SelectPropertyConfig:
			propInfo.Options = make([]string, len(propConfig.Select.Options))
			for i, option := range propConfig.Select.Options {
				propInfo.Options[i] = option.Name
			}

		case *notionapi.MultiSelectPropertyConfig:
			propInfo.Options = make([]string, len(propConfig.MultiSelect.Options))
			for i, option := range propConfig.MultiSelect.Options {
				propInfo.Options[i] = option.Name
			}
		}

		info.Properties[propName] = propInfo
	}

	bytes, err := json.Marshal(info)
	commands.Guard(err)

	_, err = os.Stdout.Write(bytes)
	commands.Guard(err)
}

type NotionCliInfo struct {
	Order      []string            `json:"order"`
	Properties map[string]PropInfo `json:"properties"`
}

type PropInfo struct {
	Type    string   `json:"type"`
	Default *string  `json:"default,omitempty"`
	Options []string `json:"options,omitempty"`
}
