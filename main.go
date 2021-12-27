package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"time"

	"github.com/jomei/notionapi"
	"github.com/manifoldco/promptui"

	"github.com/crockeo/notion-cli/config"
	"github.com/crockeo/notion-cli/markdown"
	"github.com/crockeo/notion-cli/parse"
	"github.com/crockeo/notion-cli/prompt"
)

func main() {
	config, err := config.Load()
	guard(err)
	client := notionapi.NewClient(config.Token)

	args := os.Args[1:]
	if len(args) != 1 {
		printHelp()
		os.Exit(1)
	}

	command := args[0]
	if command == "capture" {
		capture(config, client)
	} else if command == "complete" {
		complete(config, client)
	} else {
		printHelp()
		os.Exit(1)
	}
}

func guard(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Proper usage:", os.Args[0], "<command>")
	fmt.Println("  capture     Interactively capture a task from the terminal")
	fmt.Println("  complete    Complete items with the time at which they were completed")
}

func complete(config *config.Config, client *notionapi.Client) {
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var cursor notionapi.Cursor
	hasMore := true
	for hasMore {
		resp, err := client.Database.Query(
			context.Background(),
			notionapi.DatabaseID(config.DatabaseID),
			&notionapi.DatabaseQueryRequest{
				StartCursor: cursor,
				CompoundFilter: &notionapi.CompoundFilter{
					notionapi.FilterOperatorAND: {
						{
							Property: "Status",
							Select: &notionapi.SelectFilterCondition{
								Equals: "DONE",
							},
						},
						{
							Property: "Completed",
							Date: &notionapi.DateFilterCondition{
								IsEmpty: true,
							},
						},
					},
				},
			},
		)
		guard(err)

		cursor = resp.NextCursor
		hasMore = resp.HasMore

		for _, result := range resp.Results {
			_, err := client.Page.Update(
				context.Background(),
				notionapi.PageID(result.ID),
				&notionapi.PageUpdateRequest{
					Properties: notionapi.Properties{
						"Completed": &parse.DateProperty{
							Date: parse.DateObject{
								Start: (*parse.TimelessDate)(&now),
							},
						},
					},
				},
			)
			guard(err)
		}
	}
}

func capture(config *config.Config, client *notionapi.Client) {
	// pulling the database takes a moment
	// so we disguise the API call latency
	// behind a prompt for the page title
	databaseChan := make(chan *notionapi.Database)
	errChan := make(chan error)
	go func() {
		database, err := client.Database.Get(context.Background(), notionapi.DatabaseID(config.DatabaseID))
		guard(err)
		if err != nil {
			errChan <- err
		} else {
			databaseChan <- database
		}
	}()

	titlePrompt := promptui.Prompt{Label: "Title"}
	title, err := titlePrompt.Run()
	guard(err)

	var database *notionapi.Database
	select {
	case database = <-databaseChan:
	case err := <-errChan:
		guard(err)
	}

	properties := map[string]notionapi.Property{}
	for propName, propValue := range config.Defaults {
		propConfig, ok := database.Properties[propName]
		if !ok {
			fmt.Println("CaptureConfig.Defaults contains propName which doesn't exist", propName)
		}

		property, err := parse.Property(propName, propConfig, propValue)
		guard(err)

		properties[propName] = property
	}

	for _, propName := range config.Order {
		if _, ok := properties[propName]; ok {
			fmt.Println(
				"CaptureConfig.Order and CaptureConfig.Defaults contain the same propName",
				propName,
			)
			os.Exit(1)
		}

		propConfig, ok := database.Properties[propName]
		if !ok {
			fmt.Println("CaptureConfig.Order contains propName which doesn't exist", propName)
			os.Exit(1)
		}

		property, err := prompt.Property(title, propName, propConfig)
		guard(err)

		properties[propName] = property
	}

	for propName, propConfig := range database.Properties {
		if _, ok := properties[propName]; ok {
			continue
		}

		property, err := prompt.Property(title, propName, propConfig)
		guard(err)

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
			if err == io.EOF {
				break
			}
			guard(err)
			contents = append(contents, body[:n]...)
		}
	} else {
		file, err := ioutil.TempFile("", "*.md")
		guard(err)
		defer os.Remove(file.Name())

		cmd := exec.Command(editor, file.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		guard(err)

		contents, err = os.ReadFile(file.Name())
		guard(err)
	}

	children, err := markdown.ToBlocks(contents)
	guard(err)

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
	guard(err)
}
