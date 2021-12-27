package parse

import (
	"fmt"
	"regexp"
	"time"

	"github.com/olebedev/when/rules"
	"github.com/olebedev/when/rules/en"
	"github.com/jomei/notionapi"
)

func ExactMonthDateBiasNextYear(s rules.Strategy) rules.Rule {
	rule := en.ExactMonthDate(s).(*rules.F)
	return &rules.F{
		// copied directly from olebedev/when
		RegExp: regexp.MustCompile("(?i)" +
			"(?:\\W|^)" +
			"(?:(?:(" + en.ORDINAL_WORDS_PATTERN[3:] + "(?:\\s+of)?|([0-9]+))\\s*)?" +
			"(" + en.MONTH_OFFSET_PATTERN[3:] + // skip '(?:'
			"(?:\\s*(?:(" + en.ORDINAL_WORDS_PATTERN[3:] + "|([0-9]+)))?" +
			"(?:\\W|$)",
		),

		Applier: func(m *rules.Match, c *rules.Context, o *rules.Options, ref time.Time) (bool, error) {
			applied, err := rule.Applier(m, c, o, ref)
			if err != nil {
				return false, err
			}

			if !applied {
				return false, nil
			}

			year, month, day := ref.Date()
			var parsedDay int
			if c.Day == nil {
				parsedDay = day
			} else {
				parsedDay = *c.Day
			}
			parsedDate := time.Date(year, time.Month(*c.Month), parsedDay, 0, 0, 0, 0, ref.Location())
			roundedDate := time.Date(year, month, day, 0, 0, 0, 0, ref.Location())
			fmt.Println(parsedDate, roundedDate)
			if parsedDate.Before(roundedDate) {
				year = year + 1
				c.Year = &year
			}

			return true, nil
		},
	}
}

// this cursed block here replicates the API of notion
// while allowing us to serialize datetimes without the time part
// so that we can schedule tasks without assigning times of days
type DateProperty struct {
	ID   notionapi.ObjectID     `json:"id,omitempty"`
	Type notionapi.PropertyType `json:"type,omitempty"`
	Date DateObject             `json:"date"`
}

func (dp *DateProperty) GetType() notionapi.PropertyType {
	return dp.Type
}

type DateObject struct {
	Start *TimelessDate `json:"start"`
	End   *TimelessDate `json:"end"`
}

type TimelessDate time.Time

func (td *TimelessDate) MarshalJSON() ([]byte, error) {
	var format string
	date := (*time.Time)(td)
	if date.Hour() != 0 || date.Minute() != 0 || date.Second() != 0 || date.Nanosecond() != 0 {
		format = time.RFC3339
	} else {
		format = "2006-01-02"
	}
	result := date.Format(format)
	return []byte("\"" + result + "\""), nil
}
