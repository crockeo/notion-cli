package prompt

import (
	"strings"
	"time"

	"github.com/jomei/notionapi"
	"github.com/manifoldco/promptui"

	"github.com/crockeo/notion-capture/errors"
	"github.com/crockeo/notion-capture/parse"
)

func Property(title string, propName string, propConfig notionapi.PropertyConfig) (notionapi.Property, error) {
	var property notionapi.Property
	var err error

	if _, ok := propConfig.(*notionapi.TitlePropertyConfig); ok {
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
	} else if textPropConfig, ok := propConfig.(*notionapi.RichTextPropertyConfig); ok {
		property, err = promptText(propName, textPropConfig)
	} else if numberPropConfig, ok := propConfig.(*notionapi.NumberPropertyConfig); ok {
		property, err = promptNumber(propName, numberPropConfig)
	} else if selectPropConfig, ok := propConfig.(*notionapi.SelectPropertyConfig); ok {
		property, err = promptSelect(propName, selectPropConfig)
	} else if multiSelectPropConfig, ok := propConfig.(*notionapi.MultiSelectPropertyConfig); ok {
		property, err = promptMultiSelect(propName, multiSelectPropConfig)
	} else if datePropConfig, ok := propConfig.(*notionapi.DatePropertyConfig); ok {
		property, err = promptDate(propName, datePropConfig)
	} else if checkboxPropConfig, ok := propConfig.(*notionapi.CheckboxPropertyConfig); ok {
		property, err = promptCheckbox(propName, checkboxPropConfig)
	} else if URLPropConfig, ok := propConfig.(*notionapi.URLPropertyConfig); ok {
		property, err = promptURL(propName, URLPropConfig)
	} else if emailPropConfig, ok := propConfig.(*notionapi.EmailPropertyConfig); ok {
		property, err = promptEmail(propName, emailPropConfig)
	} else if phoneNumberPropConfig, ok := propConfig.(*notionapi.PhoneNumberPropertyConfig); ok {
		property, err = promptPhoneNumber(propName, phoneNumberPropConfig)
	} else {
		err = errors.ErrInvalidPropertyConfig
	}

	if err != nil {
		return nil, err
	}

	return property, nil
}

func promptText(propertyName string, property *notionapi.RichTextPropertyConfig) (*notionapi.TextProperty, error) {
	// TODO: implement, and fix function signature
	return nil, nil
}

func promptNumber(propertyName string, property *notionapi.NumberPropertyConfig) (*notionapi.NumberProperty, error) {
	// TODO: implement
	return nil, nil
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
	return nil, nil
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
	// TODO: implement
	return nil, nil
}

func promptURL(propertyName string, property *notionapi.URLPropertyConfig) (*notionapi.URLProperty, error) {
	// TODO: implement
	return nil, nil
}

func promptEmail(propertyName string, property *notionapi.EmailPropertyConfig) (*notionapi.EmailProperty, error) {
	// TODO: implement
	return nil, nil
}

func promptPhoneNumber(propertyName string, property *notionapi.PhoneNumberPropertyConfig) (*notionapi.PhoneNumberProperty, error) {
	// TODO: implement
	return nil, nil
}

func normalizeSelect(str string) string {
	return strings.Replace(strings.ToLower(str), " ", "", -1)
}
