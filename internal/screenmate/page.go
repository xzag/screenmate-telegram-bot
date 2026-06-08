package screenmate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type PageForm struct {
	URL    string
	Action string
	Values url.Values
	Doc    *goquery.Document
	Body   string
}

func (c *Client) getPageForm(ctx context.Context, pageURL string) (*PageForm, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "screenmate-telegram-bot/0.1")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get %s: %s", pageURL, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parsePageForm(resp.Request.URL, data)
}

func (c *Client) postPageForm(
	ctx context.Context,
	pageURL string,
	form url.Values,
	referer string,
) (*PageForm, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		pageURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "screenmate-telegram-bot/0.1")
	req.Header.Set("Referer", referer)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("post %s: %s", pageURL, resp.Status)
	}

	return parsePageForm(resp.Request.URL, data)
}

func cloneValues(src url.Values) url.Values {
	dst := make(url.Values, len(src))

	for key, values := range src {
		copied := make([]string, len(values))
		copy(copied, values)
		dst[key] = copied
	}

	return dst
}

func parsePageForm(finalURL *url.URL, data []byte) (*PageForm, error) {
	body := string(data)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	formSelection := doc.Find("form").First()

	values := url.Values{}

	formSelection.Find("input").Each(func(_ int, s *goquery.Selection) {
		name, ok := s.Attr("name")
		if !ok || name == "" {
			return
		}

		inputType, _ := s.Attr("type")
		inputType = strings.ToLower(strings.TrimSpace(inputType))

		switch inputType {
		case "", "hidden", "text", "password":
			value, _ := s.Attr("value")
			values.Set(name, value)

		case "checkbox", "radio":
			if _, checked := s.Attr("checked"); checked {
				value, ok := s.Attr("value")
				if !ok {
					value = "on"
				}
				values.Set(name, value)
			}

		case "submit", "image", "button", "reset":
			// Не добавляем автоматически.
			// Конкретную кнопку добавляем явно в login/selectRoom/click.
			return

		default:
			value, _ := s.Attr("value")
			values.Set(name, value)
		}
	})

	formSelection.Find("select").Each(func(_ int, s *goquery.Selection) {
		name, ok := s.Attr("name")
		if !ok || name == "" {
			return
		}

		var value string

		selected := s.Find("option[selected]").First()
		if selected.Length() == 0 {
			selected = s.Find("option").First()
		}

		if selected.Length() > 0 {
			value, _ = selected.Attr("value")
		}

		values.Set(name, value)
	})

	action := finalURL.String()

	if formAction, ok := formSelection.Attr("action"); ok && formAction != "" {
		actionURL, err := finalURL.Parse(formAction)
		if err != nil {
			return nil, fmt.Errorf("parse form action %q: %w", formAction, err)
		}

		action = actionURL.String()
	}

	return &PageForm{
		URL:    finalURL.String(),
		Action: action,
		Values: values,
		Doc:    doc,
		Body:   body,
	}, nil
}
