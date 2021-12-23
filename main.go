package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/jomei/notionapi"
	"github.com/manifoldco/promptui"
	"gopkg.in/yaml.v2"

	"github.com/crockeo/notion-capture/parse"
	"github.com/crockeo/notion-capture/prompt"
)

func main() {
	config, err := LoadCaptureConfig()
	if err != nil {
		config = DefaultCaptureConfig()
	}

	client := notionapi.NewClient(config.Token)

	// pulling the database takes a moment
	// so we disguise the API call latency
	// behind a prompt for the page title
	databaseChan := make(chan *notionapi.Database)
	go func() {
		database, err := client.Database.Get(context.Background(), notionapi.DatabaseID(config.DatabaseID))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		databaseChan <- database
	}()

	titlePrompt := promptui.Prompt{Label: "Title"}
	title, err := titlePrompt.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	database := <-databaseChan

	properties := map[string]notionapi.Property{}
	for propName, propValue := range config.Defaults {
		propConfig, ok := database.Properties[propName]
		if !ok {
			fmt.Println("CaptureConfig.Defaults contains propName which doesn't exist", propName)
		}

		property, err := parse.Property(propName, propConfig, propValue)
		if err != nil {
			fmt.Print(propName, err)
			os.Exit(1)
		}

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
		if err != nil {
			fmt.Print(propName, err)
			os.Exit(1)
		}

		properties[propName] = property
	}

	for propName, propConfig := range database.Properties {
		if _, ok := properties[propName]; ok {
			continue
		}

		property, err := prompt.Property(title, propName, propConfig)
		if err != nil {
			fmt.Println(propName, err)
			os.Exit(1)
		}

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

	bodyPrompt := promptui.Prompt{Label: "Body"}
	body, err := bodyPrompt.Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	children := []notionapi.Block{}
	if len(body) > 0 {
		children = append(children, notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: "block",
				Type:   "paragraph",
			},
			Paragraph: notionapi.Paragraph{
				Text: []notionapi.RichText{
					{
						Text: notionapi.Text{
							Content: body,
						},
					},
				},
			},
		})
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
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type CaptureConfig struct {
	DatabaseID string            `yaml:"database"`
	Defaults   map[string]string `yaml:"defaults"`
	Order      []string          `yaml:"order"`
	Token      notionapi.Token   `yaml:"token"`
}

func DefaultCaptureConfig() *CaptureConfig {
	return &CaptureConfig{}
}

func LoadCaptureConfig() (*CaptureConfig, error) {
	home, ok := os.LookupEnv("HOME")
	if !ok {
		return nil, errors.New("program not provided HOME directory")
	}

	files := []string{
		filepath.Join(home, ".notion-capture"),
		filepath.Join(home, ".config", ".notion-capture"),
		".notion-capture",
	}
	suffixes := []string{
		".yaml",
		".yml",
	}

	paths := make([]string, len(files)*len(suffixes))
	for i, file := range files {
		for j, suffix := range suffixes {
			paths[i*len(suffixes)+j] = file + suffix
		}
	}

	var contents []byte
	var err error
	for _, path := range paths {
		contents, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}

	if len(contents) == 0 {
		return nil, errors.New("could not find a .notion-capture")
	}

	captureConfig := &CaptureConfig{}
	err = yaml.Unmarshal(contents, captureConfig)
	if err != nil {
		return nil, err
	}

	return captureConfig, nil
}
