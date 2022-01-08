package capture

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"io"

	"github.com/jomei/notionapi"
	"github.com/manifoldco/promptui"

	"github.com/crockeo/notion-cli/config"
	"github.com/crockeo/notion-cli/commands"
	"github.com/crockeo/notion-cli/database"
	"github.com/crockeo/notion-cli/markdown"
	"github.com/crockeo/notion-cli/parse"
	"github.com/crockeo/notion-cli/prompt"
)


func Capture(config *config.Config, client *notionapi.Client) {
	// pulling the database takes a moment
	// so we disguise the API call latency
	// behind a prompt for the page title
	databaseChan, errChan := database.Get(config, client)

	titlePrompt := promptui.Prompt{Label: "Title"}
	title, err := titlePrompt.Run()
	commands.Guard(err)

	var database *notionapi.Database
	select {
	case database = <-databaseChan:
	case err := <-errChan:
		commands.Guard(err)
	}

	properties := map[string]notionapi.Property{}
	for propName, propValue := range config.Capture.Defaults {
		propConfig, ok := database.Properties[propName]
		if !ok {
			fmt.Println("Config.Capture.Defaults contains propName which doesn't exist", propName)
			os.Exit(1)
		}

		property, err := parse.Property(propName, propConfig, propValue)
		commands.Guard(err)

		properties[propName] = property
	}

	order := config.Capture.Order[:]
	for propName := range database.Properties {
		if !config.Capture.HasOrder(propName) {
			order = append(order, propName)
		}
	}

	for _, propName := range order {
		if config.Capture.HasDefault(propName) {
			continue
		}

		propConfig, ok := database.Properties[propName]
		if !ok {
			fmt.Println("Capture.Capture.Order contains propName which doesn't exist", propName)
			os.Exit(1)
		}
		if _, ok := propConfig.(*notionapi.FormulaPropertyConfig); ok {
			// we can't populate anything for a formula
			// so we just keep on rolling
			continue
		}

		property, err := prompt.Property(title, propName, propConfig)
		commands.Guard(err)

		properties[propName] = property
	}

	// we must remove null values from the list
	// so that they're not accidentally serialized
	// and sent to the API
	for propName, property := range properties {
		if property == nil || reflect.ValueOf(property).Kind() == reflect.Ptr && reflect.ValueOf(property).IsNil() {
			delete(properties, propName)
		}
	}

	contents := []byte{}
	editor, ok := os.LookupEnv("EDITOR")
	if !ok {
		fmt.Println("Body:")
		contents := []byte{}
		body := make([]byte, 512)
		for {
			n, err := os.Stdin.Read(body)
			if err != nil && err != io.EOF {
				commands.Guard(err)
			}
			contents = append(contents, body[:n]...)
			if err == io.EOF {
				break
			}
		}
	} else {
		file, err := ioutil.TempFile("", "*.md")
		commands.Guard(err)
		defer os.Remove(file.Name())

		cmd := exec.Command(editor, file.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		commands.Guard(err)

		contents, err = os.ReadFile(file.Name())
		commands.Guard(err)
	}

	children, err := markdown.ToBlocks(contents)
	commands.Guard(err)

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
