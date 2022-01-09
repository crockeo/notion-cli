package database

import (
	"context"

	"github.com/jomei/notionapi"

	"github.com/crockeo/notion-cli/config"
)

func Get(config *config.Config, client *notionapi.Client) (chan *notionapi.Database, chan error) {
	databaseChan := make(chan *notionapi.Database)
	errChan := make(chan error)
	go func() {
		database, err := client.Database.Get(context.Background(), notionapi.DatabaseID(config.DatabaseID))
		if err != nil {
			errChan <- err
		} else {
			databaseChan <- database
		}
	}()
	return databaseChan, errChan
}

func GetSync(config *config.Config, client *notionapi.Client) (*notionapi.Database, error) {
	databaseChan, errChan := Get(config, client)
	select {
	case database := <-databaseChan:
		return database, nil
	case err := <-errChan:
		return nil, err
	}
}

func Join(databaseChan chan *notionapi.Database, errChan chan error) (*notionapi.Database, error) {
	select {
	case database := <-databaseChan:
		return database, nil
	case err := <-errChan:
		return nil, err
	}
}
