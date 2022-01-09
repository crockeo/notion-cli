package dump

import (
	"encoding/json"
	"os"

	"github.com/jomei/notionapi"

	"github.com/crockeo/notion-cli/commands"
	"github.com/crockeo/notion-cli/config"
	"github.com/crockeo/notion-cli/database"
)

func Dump(config *config.Config, client *notionapi.Client) {
	database, err := database.GetSync(config, client)
	commands.Guard(err)

	propsToInfos := map[string]PropInfo{}
	for propName, propConfig := range database.Properties {
		propInfo := PropInfo{Type: string(propConfig.GetType())}
		if defaultValue, ok := config.Capture.Defaults[propName]; ok {
			propInfo.Default = &defaultValue
		}
		propsToInfos[propName] = propInfo
	}

	bytes, err := json.Marshal(propsToInfos)
	commands.Guard(err)

	_, err = os.Stdout.Write(bytes)
	commands.Guard(err)
}

type PropInfo struct {
	Type    string  `json:"type"`
	Default *string `json:"default,omitempty"`
}
