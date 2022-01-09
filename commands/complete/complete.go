package complete

import (
	"context"
	"fmt"
	"time"

	"github.com/jomei/notionapi"

	"github.com/crockeo/notion-cli/commands"
	"github.com/crockeo/notion-cli/config"
	"github.com/crockeo/notion-cli/database"
	"github.com/crockeo/notion-cli/parse"
)

func Complete(config *config.Config, client *notionapi.Client, args []string) {
	database, err := database.GetSync(config, client)
	commands.Guard(err)

	_, ok := database.Properties[config.Complete.CompletedProperty]
	commands.GuardOk(ok, fmt.Sprintf("config.Complete.CompletedProperty does not exist: %v", config.Complete.CompletedProperty))

	propConfig, ok := database.Properties[config.Complete.StatusProperty]
	commands.GuardOk(ok, fmt.Sprintf("config.Complete.StatusProperty does not exist: %v", config.Complete.StatusProperty))

	_, ok = propConfig.(*notionapi.CheckboxPropertyConfig)
	commands.GuardOk(ok, "config.Complete.StatusProperty is not a checkbox")

	checkboxProp, err := parse.ParseCheckbox(config.Complete.DoneStatus)
	commands.Guard(err)

	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// note that this section is intentionally serial
	// despite being parallelizable
	// because we generally run this in the background
	// and don't want to be rate limited by the Notion API
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
							Property: config.Complete.StatusProperty,
							Checkbox: &notionapi.CheckboxFilterCondition{
								Equals: checkboxProp.Checkbox,
							},
						},
						{
							Property: config.Complete.CompletedProperty,
							Date: &notionapi.DateFilterCondition{
								IsEmpty: true,
							},
						},
					},
				},
			},
		)
		commands.Guard(err)

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
			commands.Guard(err)
		}
	}
}
