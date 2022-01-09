package capture

import (
	"context"
	"encoding/json"
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

type captureBlob struct {
	Title      *string           `json:"title,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

func Capture(config *config.Config, client *notionapi.Client, args []string) {
	var blob *captureBlob
	if len(args) > 0 {
		blob = &captureBlob{}
		err := json.Unmarshal([]byte(args[0]), blob)
		commands.Guard(err)
	}

	fmt.Println(blob)

	// pulling the database takes a moment
	// so we disguise the API call latency
	// behind a prompt for the page title
	databaseChan, errChan := database.Get(config, client)

	var title string
	if blob != nil && blob.Title != nil {
		var ok bool
		title, ok = blob.Properties[*blob.Title]
		commands.GuardOk(ok, "thing said thing but it was thing")
	} else {
		titlePrompt := promptui.Prompt{Label: "Title"}
		var err error
		title, err = titlePrompt.Run()
		commands.Guard(err)
	}

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

	if blob != nil {
		for propName, propValue := range blob.Properties {
			propConfig, ok := database.Properties[propName]
			commands.GuardOk(ok, fmt.Sprintf("Provided JSON arg %s does not exist in the database", propName))

			property, err := parse.Property(propName, propConfig, propValue)
			commands.Guard(err)

			properties[propName] = property
		}
	}

	// TODO: add a guard for non-interactive modes
	// so that if we would normally try to prompt
	// instead we just throw an error and cry

	order := config.Capture.Order[:]
	for propName := range database.Properties {
		if _, ok := properties[propName]; ok {
			order = append(order, propName)
		}
	}

	for _, propName := range order {
		if _, ok := properties[propName]; ok {
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

	// TODO: with the non-interactive thing in place
	// prompt for body only when it's not present
	// and it's in interactive mode

	children := []notionapi.Block{}
	if blob == nil {
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

		var err error
		children, err = markdown.ToBlocks(contents)
		commands.Guard(err)
	}

	_, err := client.Page.Create(
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
