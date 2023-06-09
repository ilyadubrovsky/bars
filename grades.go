package bars

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"regexp"
	"strings"
)

// ProgressTable Таблица оценок в БАРС.
type ProgressTable struct {
	// Tables табели успеваемости по предметам.
	Tables []SubjectTable `json:"tables"`
}

func (pt *ProgressTable) String() string {
	var b strings.Builder

	for _, st := range pt.Tables {
		fmt.Fprintf(&b, "%s\n", st.String())
	}

	return b.String()
}

// SubjectTable Таблица оценок предмета в БАРС.
type SubjectTable struct {
	// Name название предмета.
	Name string `json:"name"`
	// ControlEvents информация об успеваемости по предмету (контрольные мероприятия).
	ControlEvents []ControlEvent `json:"control_events"`
}

func (st *SubjectTable) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Дисциплина: %s\n", st.Name)

	for _, event := range st.ControlEvents {
		fmt.Fprintf(&b, "%s\n", event.String())
	}

	return b.String()
}

// ControlEvent информация о контрольном мероприятии.
type ControlEvent struct {
	// Name название контрольного мероприятия.
	Name string `json:"name"`
	// Grades оценки за выполнение контрольного мероприятия.
	Grades string `json:"grades"`
}

func (c *ControlEvent) String() string {
	return fmt.Sprintf("Контрольное мероприятие: %s\nОценка: %s\n", c.Name, c.Grades)
}

// GetProgressTable получает табель успеваемости авторизованного студента и возращает JSON encode.
// Возращает ErrWrongGradesPage, если страница с оценками пользователя не является основной.
// Возвращает ошибку с информацией, если полезная нагрузка таблицы оказалась пустой или возникла другая непредвиденная ошибка.
// Заменяет пустые поля для оценок на "отсутствует".
func (c *Client) GetProgressTable(ctx context.Context) (*ProgressTable, error) {
	response, err := c.getPage(ctx, http.MethodGet, GradesPageURL, nil)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(response.Request.URL.String(), PersonalGradesPageURL) {
		return nil, ErrWrongGradesPage
	}

	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, err
	}

	ptLen := document.Find("tbody").Length()
	ptObject := &ProgressTable{Tables: make([]SubjectTable, ptLen)}

	// extract subject table names
	document.Find(".my-2").Find("div:first-child").Clone().Children().Remove().End().EachWithBreak(func(nameID int, name *goquery.Selection) bool {
		processedString := strings.TrimSpace(name.Text())
		if isEmptyData(processedString) {
			err = fmt.Errorf("part of received data is empty. nameID := %d", nameID)
			return false
		}
		ptObject.Tables[nameID].Name = processedString
		return true
	})
	if err != nil {
		return nil, err
	}

	filterTrSelection := func(i int, tr *goquery.Selection) bool {
		trLen := tr.Find("td").Length()
		return trLen == 4 || trLen == 2
	}

	// extract subject tables data
	flag := true
	document.Find("tbody").EachWithBreak(func(tbodyID int, tbody *goquery.Selection) bool {
		trSelection := tbody.Find("tr").FilterFunction(filterTrSelection)

		stLen := trSelection.Length()
		stObject := SubjectTable{ControlEvents: make([]ControlEvent, stLen)}

		trSelection.EachWithBreak(func(trID int, tr *goquery.Selection) bool {
			eventObject := ControlEvent{}
			tdSelection := tr.Find("td")

			tdSelection.EachWithBreak(func(tdID int, td *goquery.Selection) bool {
				processedString := removeExtraSpaces(strings.TrimSpace(td.Text()))
				switch tdID {
				case 0:
					if isEmptyData(processedString) {
						err = fmt.Errorf("part of received data is empty. tdId: %d trId: %d tbodyId: %d", tdID, trID, tbodyID)
						flag = false
					} else {
						eventObject.Name = processedString
					}
				case tdSelection.Length() - 1:
					if isEmptyData(processedString) {
						eventObject.Grades = "отсутствует"
					} else {
						eventObject.Grades = processedString
					}
				}

				return flag
			})

			stObject.ControlEvents[trID] = eventObject
			return flag
		})

		ptObject.Tables[tbodyID].ControlEvents = stObject.ControlEvents
		return flag
	})
	if err != nil {
		return nil, err
	}

	return ptObject, nil
}

func removeExtraSpaces(s string) string {
	re := regexp.MustCompile(`\s{2,}`)

	return re.ReplaceAllString(s, " ")
}

func isEmptyData(data string) bool {
	return data == "" || data == " "
}
