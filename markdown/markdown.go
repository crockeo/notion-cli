package markdown

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
	"github.com/jomei/notionapi"
)

func ToBlocks(contents []byte) ([]notionapi.Block, error) {
	node := markdown.Parse(contents, parser.New())
	blocks, err := transform(node)
	output, _ := json.MarshalIndent(blocks, "", "  ")
	fmt.Println(string(output))
	return blocks, err
}

func transform(node ast.Node) ([]notionapi.Block, error) {
	var blocks []notionapi.Block
	var err error

	switch node := node.(type) {
	case *ast.Document:
		blocks, err = transformDocument(node)
	case *ast.Heading:
		blocks, err = transformHeading(node)
	case *ast.Paragraph:
		blocks, err = transformParagraph(node)
	case *ast.List:
		blocks, err = transformList(node)
	default:
		if t := reflect.TypeOf(node); t.Kind() == reflect.Ptr {
			fmt.Println("*" + t.Elem().Name())
		} else {
			fmt.Println(t.Name())
		}
	}

	return blocks, err
}

func transformDocument(node *ast.Document) ([]notionapi.Block, error) {
	blocks := []notionapi.Block{}
	for _, node := range node.Children {
		nodeBlocks, err := transform(node)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, nodeBlocks...)
	}
	return blocks, nil
}

func transformHeading(node *ast.Heading) ([]notionapi.Block, error) {
	text, ok := node.Children[0].(*ast.Text)
	if !ok {
		return nil, errors.New("asdfasdf")
	}
	richText := transformText(text)

	level := node.Level
	if level < 1 {
		level = 1
	} else if level > 3 {
		level = 3
	}

	basicBlock := notionapi.BasicBlock{
		Object: "block",
		Type:   notionapi.BlockType(fmt.Sprintf("heading_%d", level)),
	}
	heading := notionapi.Heading{
		Text: []notionapi.RichText{richText},
	}

	blocks := []notionapi.Block{}
	if level == 1 {
		blocks = append(blocks, &notionapi.Heading1Block{
			BasicBlock: basicBlock,
			Heading1:   heading,
		})
	} else if level == 2 {
		blocks = append(blocks, &notionapi.Heading2Block{
			BasicBlock: basicBlock,
			Heading2:   heading,
		})
	} else if level >= 3 {
		blocks = append(blocks, &notionapi.Heading3Block{
			BasicBlock: basicBlock,
			Heading3:   heading,
		})
	}
	return blocks, nil
}

func transformParagraph(node *ast.Paragraph) ([]notionapi.Block, error) {
	text, err := buildRichText(node.Children)
	if err != nil {
		return nil, err
	}
	return []notionapi.Block{
		&notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: "block",
				Type:   "paragraph",
			},
			Paragraph: notionapi.Paragraph{
				Text: text,
			},
		},
	}, nil
}

func transformList(node *ast.List) ([]notionapi.Block, error) {
	blocks := []notionapi.Block{}
	for _, child := range node.Children {
		listItem, ok := child.(*ast.ListItem)
		if !ok {
			return nil, errors.New(fmt.Sprintf("expected *ast.ListItem, got %v", reflect.TypeOf(child)))
		}

		paragraph, ok := listItem.Children[0].(*ast.Paragraph)
		if !ok {
			return nil, errors.New(fmt.Sprintf("expected *ast.Paragraph, got %v", reflect.TypeOf(child)))
		}

		text, err := buildRichText(paragraph.Children)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, &notionapi.BulletedListItemBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: "block",
				Type:   "bulleted_list_item",
			},
			BulletedListItem: notionapi.ListItem{
				Text: text,
			},
		})
	}
	return blocks, nil
}

func buildRichText(nodes []ast.Node) ([]notionapi.RichText, error) {
	text := []notionapi.RichText{}
	for _, node := range nodes {
		switch node := node.(type) {
		case *ast.Emph:
			text = append(text, transformEmph(node))
		case *ast.Link:
			text = append(text, transformLink(node))
		case *ast.Strong:
			text = append(text, transformStrong(node))
		case *ast.Text:
			text = append(text, transformText(node))
		default:
			return nil, errors.New(fmt.Sprintf("unrecognized markdown node type %v", reflect.TypeOf(node)))
		}
	}
	return text, nil
}

func transformEmph(node *ast.Emph) notionapi.RichText {
	text := transformText(node.Children[0].(*ast.Text))
	text.Annotations = &notionapi.Annotations{
		Italic: true,
	}
	return text
}

func transformLink(node *ast.Link) notionapi.RichText {
	text := transformText(node.Children[0].(*ast.Text))
	text.Text.Link = &notionapi.Link{
		Url: string(node.Destination),
	}
	return text
}

func transformStrong(node *ast.Strong) notionapi.RichText {
	text := transformText(node.Children[0].(*ast.Text))
	text.Annotations = &notionapi.Annotations{
		Bold: true,
	}
	return text
}

func transformText(node *ast.Text) notionapi.RichText {
	return notionapi.RichText{
		Text: notionapi.Text{
			Content: string(node.Literal),
		},
	}
}
