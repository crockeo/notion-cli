package prompt

import (
	"fmt"
	"strings"
	"time"

	"github.com/jomei/notionapi"
	"github.com/manifoldco/promptui"

	"github.com/crockeo/notion-cli/errors"
	"github.com/crockeo/notion-cli/parse"
)

func Property(title string, propName string, propConfig notionapi.PropertyConfig) (notionapi.Property, error) {
	var property notionapi.Property
	var err error

	switch propConfig := propConfig.(type) {
	case *notionapi.TitlePropertyConfig:
		// we already prompted for the title above
		// before we had reference to the actual database
		// so we just populate it here, instead of re-prompting
		property = notionapi.TitleProperty{
			Title: []notionapi.RichText{
				{
					Text: notionapi.Text{
						Content: title,
					},
				},
			},
		}
	case *notionapi.RichTextPropertyConfig:
		property, err = promptRichText(propName, propConfig)
	case *notionapi.NumberPropertyConfig:
		property, err = promptNumber(propName, propConfig)
	case *notionapi.SelectPropertyConfig:
		property, err = promptSelect(propName, propConfig)
	case *notionapi.MultiSelectPropertyConfig:
		property, err = promptMultiSelect(propName, propConfig)
	case *notionapi.DatePropertyConfig:
		property, err = promptDate(propName, propConfig)
	case *notionapi.CheckboxPropertyConfig:
		property, err = promptCheckbox(propName, propConfig)
	case *notionapi.URLPropertyConfig:
		property, err = promptURL(propName, propConfig)
	case *notionapi.EmailPropertyConfig:
		property, err = promptEmail(propName, propConfig)
	case *notionapi.PhoneNumberPropertyConfig:
		property, err = promptPhoneNumber(propName, propConfig)
	default:
		err = errors.NewInvalidPropertyConfig(string(propConfig.GetType()))
	}
	return property, err
}

func promptRichText(propertyName string, property *notionapi.RichTextPropertyConfig) (*notionapi.RichTextProperty, error) {
	prompt := promptui.Prompt{
		Label: propertyName,
		Validate: func(candidate string) error {
			_, err := parse.ParseRichText(candidate)
			return err
		},
	}
	richTextStr, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	return parse.ParseRichText(richTextStr)
}

func promptNumber(propertyName string, property *notionapi.NumberPropertyConfig) (*notionapi.NumberProperty, error) {
	prompt := promptui.Prompt{
		Label: propertyName,
		Validate: func(candidate string) error {
			_, err := parse.ParseNumber(candidate)
			return err
		},
	}
	numberStr, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	return parse.ParseNumber(numberStr)
}

func promptSelect(propertyName string, property *notionapi.SelectPropertyConfig) (*notionapi.SelectProperty, error) {
	// FIXME: use the templating system
	// i couldn't figure out how to get templating working
	// to display nice names instead of full structs
	// so using this temporarily to work around it
	optionNames := make([]string, len(property.Select.Options))
	for i, option := range property.Select.Options {
		optionNames[i] = option.Name
	}

	prompt := promptui.Select{
		Items: optionNames,
		Label: propertyName,
		Searcher: func(input string, index int) bool {
			option := normalizeSelect(property.Select.Options[index].Name)
			input = normalizeSelect(input)
			return strings.Contains(option, input)
		},
		StartInSearchMode: true,
	}
	_, name, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	return parse.ParseSelect(name, property.Select.Options)
}

func promptMultiSelect(propertyName string, property *notionapi.MultiSelectPropertyConfig) (*notionapi.MultiSelectProperty, error) {
	// TODO: implement
	return parse.ParseMultiSelect("", property.MultiSelect.Options)
}

func promptDate(propertyName string, property *notionapi.DatePropertyConfig) (*parse.DateProperty, error) {
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	prompt := promptui.Prompt{
		Label: propertyName,
		Validate: func(candidate string) error {
			_, err := parse.ParseDate(candidate, now)
			return err
		},
	}
	dateStr, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	return parse.ParseDate(dateStr, now)
}

func promptCheckbox(propertyName string, property *notionapi.CheckboxPropertyConfig) (*notionapi.CheckboxProperty, error) {
	prompt := promptui.Prompt{
		Label: fmt.Sprintf("%s (y/n)", propertyName),
		Validate: func(candidate string) error {
			_, err := parse.ParseCheckbox(candidate)
			return err
		},
	}
	checkboxStr, err := prompt.Run()
	if err != nil && err != promptui.ErrAbort {
		return nil, err
	}
	return parse.ParseCheckbox(checkboxStr)
}

func promptURL(propertyName string, property *notionapi.URLPropertyConfig) (*notionapi.URLProperty, error) {
	prompt := promptui.Prompt{
		Label: propertyName,
		Validate: func(candidate string) error {
			_, err := parse.ParseURL(candidate)
			return err
		},
	}
	urlStr, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	return parse.ParseURL(urlStr)
}

func promptEmail(propertyName string, property *notionapi.EmailPropertyConfig) (*notionapi.EmailProperty, error) {
	prompt := promptui.Prompt{
		Label: propertyName,
		Validate: func(candidate string) error {
			_, err := parse.ParseEmail(candidate)
			return err
		},
	}
	emailStr, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	return parse.ParseEmail(emailStr)
}

func promptPhoneNumber(propertyName string, property *notionapi.PhoneNumberPropertyConfig) (*notionapi.PhoneNumberProperty, error) {
	prompt := promptui.Prompt{
		Label: propertyName,
		Validate: func(candidate string) error {
			_, err := parse.ParseEmail(candidate)
			return err
		},
	}
	phoneNumberStr, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	return parse.ParsePhoneNumber(phoneNumberStr)
}

func normalizeSelect(str string) string {
	return strings.Replace(strings.ToLower(str), " ", "", -1)
}
