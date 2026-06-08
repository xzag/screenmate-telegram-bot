package screenmate

import (
	"html"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func ParseAirConditioners(doc *goquery.Document) ([]AirConditioner, error) {
	var result []AirConditioner

	doc.Find("a").Each(func(_ int, a *goquery.Selection) {
		id, ok := a.Attr("id")
		if !ok || !strings.HasPrefix(id, "dataList_toggle_") {
			return
		}

		itemTable := a.Closest("table.item")
		if itemTable.Length() == 0 {
			return
		}

		name := normalizeText(itemTable.Find("td.itemName").First().Text())

		number, ok := parseACNumber(name)
		if !ok {
			return
		}

		valueText := normalizeText(a.Find("span.itemValue").First().Text())

		href, _ := a.Attr("href")
		target, ok := postBackTargetFromHref(href)
		if !ok {
			// fallback, если вдруг href поменяется
			target, ok = postBackTargetFromToggleID(id)
			if !ok {
				return
			}
		}

		result = append(result, AirConditioner{
			Number:       number,
			Power:        valueText == "1",
			ToggleTarget: target,
		})
	})

	return result, nil
}

func postBackTargetFromHref(href string) (string, bool) {
	href = html.UnescapeString(href)

	const marker = `WebForm_PostBackOptions("`

	start := strings.Index(href, marker)
	if start < 0 {
		return "", false
	}

	start += len(marker)

	end := strings.Index(href[start:], `"`)
	if end < 0 {
		return "", false
	}

	target := href[start : start+end]
	if target == "" {
		return "", false
	}

	return target, true
}

func postBackTargetFromToggleID(id string) (string, bool) {
	const prefix = "dataList_toggle_"

	if !strings.HasPrefix(id, prefix) {
		return "", false
	}

	rawIndex := strings.TrimPrefix(id, prefix)
	if rawIndex == "" {
		return "", false
	}

	return "dataList:_ctl" + rawIndex + ":toggle", true
}

func parseACNumber(name string) (int, bool) {
	const prefix = "Кондиционер "

	if !strings.HasPrefix(name, prefix) {
		return 0, false
	}

	rest := strings.TrimPrefix(name, prefix)

	idx := strings.Index(rest, " ")
	if idx < 0 {
		idx = strings.Index(rest, "(")
	}
	if idx < 0 {
		return 0, false
	}

	n, err := strconv.Atoi(strings.TrimSpace(rest[:idx]))
	if err != nil {
		return 0, false
	}

	return n, true
}

func normalizeText(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}
