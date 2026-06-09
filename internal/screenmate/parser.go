package screenmate

import (
	"html"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func ParseAirConditioners(doc *goquery.Document) ([]AirConditioner, error) {
	byNumber := make(map[int]*AirConditioner)

	doc.Find("table.item").Each(func(_ int, table *goquery.Selection) {
		name := normalizeText(table.Find("td.itemName").First().Text())

		number, ok := parseACNumber(name)
		if !ok {
			return
		}

		ac := byNumber[number]
		if ac == nil {
			ac = &AirConditioner{Number: number}
			byNumber[number] = ac
		}

		switch {
		case strings.Contains(name, "(ВКЛ/ВЫКЛ)"):
			valueText := normalizeText(table.Find(".itemValue span.itemValue").First().Text())
			ac.Power = valueText == "1"

			table.Find("a").Each(func(_ int, a *goquery.Selection) {
				id, _ := a.Attr("id")
				if !strings.HasPrefix(id, "dataList_toggle_") {
					return
				}

				href, _ := a.Attr("href")

				target, ok := postBackTargetFromHref(href)
				if !ok {
					target, ok = postBackTargetFromID(id)
				}
				if !ok {
					return
				}

				ac.ToggleTarget = target
			})

		case strings.Contains(name, "(уставка)"):
			valueText := normalizeText(table.Find(".itemValue span.itemValue").First().Text())
			unit := normalizeText(table.Find(".itemUnit span").First().Text())

			ac.HasSetpoint = true
			ac.Setpoint.Value = valueText
			ac.Setpoint.Unit = unit

			table.Find("a").Each(func(_ int, a *goquery.Selection) {
				id, _ := a.Attr("id")
				if id == "" {
					return
				}

				className, _ := a.Attr("class")
				if strings.Contains(className, "aspNetDisabled") {
					return
				}

				href, _ := a.Attr("href")

				target, ok := postBackTargetFromHref(href)
				if !ok {
					target, ok = postBackTargetFromID(id)
				}
				if !ok {
					return
				}

				switch {
				case strings.HasPrefix(id, "dataList_previous_"):
					ac.Setpoint.DecreaseTarget = target
				case strings.HasPrefix(id, "dataList_next_"):
					ac.Setpoint.IncreaseTarget = target
				}
			})
		}
	})

	numbers := make([]int, 0, len(byNumber))
	for number := range byNumber {
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)

	result := make([]AirConditioner, 0, len(numbers))
	for _, number := range numbers {
		result = append(result, *byNumber[number])
	}

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
	return target, target != ""
}

func postBackTargetFromID(id string) (string, bool) {
	const prefix = "dataList_"

	if !strings.HasPrefix(id, prefix) {
		return "", false
	}

	raw := strings.TrimPrefix(id, prefix)

	idx := strings.LastIndex(raw, "_")
	if idx < 0 {
		return "", false
	}

	action := raw[:idx]
	index := raw[idx+1:]

	if action == "" || index == "" {
		return "", false
	}

	return "dataList:_ctl" + index + ":" + action, true
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
