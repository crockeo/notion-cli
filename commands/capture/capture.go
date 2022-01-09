package capture

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"

	"github.com/jomei/notionapi"
	"github.com/manifoldco/promptui"

	"github.com/crockeo/notion-cli/commands"
	"github.com/crockeo/notion-cli/config"
	"github.com/crockeo/notion-cli/database"
	"github.com/crockeo/notion-cli/markdown"
	"github.com/crockeo/notion-cli/parse"
	"github.com/crockeo/notion-cli/prompt"
)

type PropInfo struct {
	Title      *string           `json:"title,omitempty"`
	Body       *string           `json:"body,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

func Capture(config *config.Config, client *notionapi.Client, args []string) {
	interactive := flag.Bool("interactive", true, "Controls if notion-cli prompts for property values when not provided.")
	propInfoStr := flag.String("propinfo", "", "Additional property info to in JSON format.")
	flag.CommandLine.Parse(args)

	var propInfo *PropInfo
	if len(*propInfoStr) > 0 {
		propInfo = &PropInfo{}
		err := json.Unmarshal([]byte(*propInfoStr), propInfo)
		commands.Guard(err)
		args = args[1:]
	}

	// pulling the database takes a moment
	// so we disguise the API call latency
	// behind a prompt for the page title
	databaseChan, errChan := database.Get(config, client)

	title, err := getTitle(propInfo, *interactive)
	commands.Guard(err)

	database, err := database.Join(databaseChan, errChan)
	commands.Guard(err)

	properties := map[string]notionapi.Property{}
	if propInfo != nil {
		propInfoProperties, err := getPropInfoProperties(database, properties, propInfo)
		commands.Guard(err)
		for propName, property := range propInfoProperties {
			properties[propName] = property
		}
	}

	defaultProperties, err := getDefaultProperties(database, properties, config)
	commands.Guard(err)
	for propName, property := range defaultProperties {
		properties[propName] = property
	}

	if *interactive {
		interactiveProps, err := getInteractiveProperties(database, properties, title, config)
		commands.Guard(err)
		for propName, property := range interactiveProps {
			properties[propName] = property
		}
	}

	contents := []byte{}
	if *interactive {
		contents, err = getInteractiveBody()
		commands.Guard(err)
	} else if propInfo != nil && propInfo.Body != nil {
		contents = []byte(*propInfo.Body)
	}

	children := []notionapi.Block{}
	if len(contents) > 0 {
		children, err = markdown.ToBlocks(contents)
		commands.Guard(err)
	}

	// we must remove null values from the list
	// so that they're not accidentally serialized
	// and sent to the API
	for propName, property := range properties {
		if property == nil || reflect.ValueOf(property).Kind() == reflect.Ptr && reflect.ValueOf(property).IsNil() {
			delete(properties, propName)
		}
	}

	_, err = client.Page.Create(
		context.Background(),
		&notionapi.PageCreateRequest{
			Parent: notionapi.Parent{
				Type:       notionapi.ParentTypeDatabaseID,
				DatabaseID: notionapi.DatabaseID(database.ID),
			},
			Properties: properties,
			Children:   children,
		},
	)
	commands.Guard(err)
}

func getTitle(propInfo *PropInfo, interactive bool) (string, error) {
	if propInfo != nil && propInfo.Title != nil {
		title, ok := propInfo.Properties[*propInfo.Title]
		if ok {
			return title, nil
		}
	}

	if interactive {
		titlePrompt := promptui.Prompt{Label: "Title"}
		return titlePrompt.Run()
	}

	return "", fmt.Errorf("failed to get a title! :(")
}

func getPropInfoProperties(database *notionapi.Database, properties notionapi.Properties, propInfo *PropInfo) (notionapi.Properties, error) {
	if propInfo == nil {
		return nil, fmt.Errorf("passed a null propInfo :(")
	}

	propInfoProperties := notionapi.Properties{}
	for propName, propValue := range propInfo.Properties {
		propConfig, ok := database.Properties[propName]
		if !ok {
			return nil, fmt.Errorf("provided JSON arg %s does not exist in the database", propName)
		}

		property, err := parse.Property(propName, propConfig, propValue)
		if err != nil {
			return nil, err
		}

		propInfoProperties[propName] = property
	}
	return propInfoProperties, nil
}

func getDefaultProperties(database *notionapi.Database, properties notionapi.Properties, config *config.Config) (notionapi.Properties, error) {
	defaultProperties := notionapi.Properties{}
	for propName, propValue := range config.Capture.Defaults {
		if _, ok := properties[propName]; ok {
			continue
		}

		propConfig, ok := database.Properties[propName]
		if !ok {
			return nil, fmt.Errorf("Config.Capture.Defaults contains propName which doesn't exist '%s'", propName)
		}

		property, err := parse.Property(propName, propConfig, propValue)
		if err != nil {
			return nil, err
		}

		defaultProperties[propName] = property
	}
	return defaultProperties, nil
}

func getInteractiveProperties(database *notionapi.Database, properties notionapi.Properties, title string, config *config.Config) (notionapi.Properties, error) {
	order := []string{}
	for _, propName := range config.Capture.Order {
		if _, ok := properties[propName]; !ok {
			order = append(order, propName)
		}
	}
	for propName := range database.Properties {
		if _, ok := properties[propName]; !ok {
			order = append(order, propName)
		}
	}

	interactiveProperties := notionapi.Properties{}
	for _, propName := range order {
		if _, ok := properties[propName]; ok {
			continue
		}

		propConfig, ok := database.Properties[propName]
		if !ok {
			return nil, fmt.Errorf("Capture.Capture.Order contains propName which doesn't exist '%s'", propName)
		}
		if _, ok := propConfig.(*notionapi.FormulaPropertyConfig); ok {
			// we can't populate anything for a formula
			// so we just keep on rolling
			continue
		}

		property, err := prompt.Property(title, propName, propConfig)
		commands.Guard(err)

		interactiveProperties[propName] = property
	}
	return interactiveProperties, nil
}

func getInteractiveBody() ([]byte, error) {
	contents := []byte{}

	editor, ok := os.LookupEnv("EDITOR")
	if !ok {
		fmt.Println("Body:")
		body := make([]byte, 512)
		for {
			n, err := os.Stdin.Read(body)
			if err != nil && err != io.EOF {
				return nil, err
			}
			contents = append(contents, body[:n]...)
			if err == io.EOF {
				break
			}
		}
	} else {
		file, err := ioutil.TempFile("", "*.md")
		if err != nil {
			return nil, err
		}
		defer os.Remove(file.Name())

		cmd := exec.Command(editor, file.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return nil, err
		}

		contents, err = os.ReadFile(file.Name())
		if err != nil {
			return nil, err
		}
	}

	return contents, nil
}
