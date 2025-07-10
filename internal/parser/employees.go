package parser

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Houeta/us-api-provider/internal/metrics"
	"github.com/Houeta/us-api-provider/internal/models"
	"github.com/PuerkitoBio/goquery"
)

type employeeShortname struct {
	ID        int
	Shortname string
}

type EmployeeParser struct {
	client  *http.Client
	destURL string
	metrics *metrics.Metrics
}

type EmployeeParserIface interface {
	ParseEmployees(ctx context.Context) ([]models.Employee, error)
}

func NewEmployeeParser(client *http.Client, metrics *metrics.Metrics, destURL string) EmployeeParserIface {
	return &EmployeeParser{client: client, destURL: destURL, metrics: metrics}
}

func (ep *EmployeeParser) ParseEmployees(ctx context.Context) ([]models.Employee, error) {
	staff, err := ep.parseActualStaff(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse staff: %w", err)
	}

	dismissedStaff, err := ep.parseDismissedStaff(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dismissed staff: %w", err)
	}

	staff = append(staff, dismissedStaff...)

	staffShortName, err := ep.parseStaffShortNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse staff short names: %w", err)
	}

	return updateEmployeeShortNames(staff, staffShortName), nil
}

// ParseStaff parses employee data from the specified destination URL using the provided HTTP client.
// It returns a slice of models.Employee and an error if any.
func (ep *EmployeeParser) parseActualStaff(ctx context.Context) ([]models.Employee, error) {
	data := url.Values{}
	tds := []int{2, 3, 4, 5, 6}

	data.Set("core_section", "staff_unit")

	resp, err := GetHTMLResponse(ctx, ep.client, &data, ep.destURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get response html reposnse: %w", err)
	}
	defer resp.Body.Close()

	return ParseEmployeeFromBody(resp.Body, ep.metrics, tds[0], tds[1], tds[2], tds[3], tds[4])
}

func (ep *EmployeeParser) parseDismissedStaff(ctx context.Context) ([]models.Employee, error) {
	data := url.Values{}
	tds := []int{3, 4, 5, 6, 7}

	data.Set("core_section", "staff_unit")
	data.Set("is_with_leaved", "1")

	resp, err := GetHTMLResponse(ctx, ep.client, &data, ep.destURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get response html reposnse: %w", err)
	}
	defer resp.Body.Close()

	return ParseEmployeeFromBody(resp.Body, ep.metrics, tds[0], tds[1], tds[2], tds[3], tds[4])
}

func (ep *EmployeeParser) parseStaffShortNames(ctx context.Context) ([]employeeShortname, error) {
	data := url.Values{}
	var employees []employeeShortname

	data.Set("core_section", "staff")
	data.Set("action", "division")

	resp, err := GetHTMLResponse(ctx, ep.client, &data, ep.destURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get response which should retrieve task types: %w", err)
	}
	defer resp.Body.Close()

	doc, _ := goquery.NewDocumentFromReader(resp.Body)

	doc.Find(`div.div_space`).Each(func(_ int, div *goquery.Selection) {
		link := div.Find("a")
		if link.Length() > 0 {
			href, exists := link.Attr("href")
			if exists {
				id, errID := parseIDFromHref(href)
				if id > 0 && errID == nil {
					employee := employeeShortname{
						ID:        id,
						Shortname: strings.TrimSpace(link.Text()),
					}
					employees = append(employees, employee)
				}
			}
		}
	})

	return employees, nil
}

func updateEmployeeShortNames(employees []models.Employee, shortnames []employeeShortname) []models.Employee {
	shortNameMap := make(map[int]string)
	for _, sn := range shortnames {
		shortNameMap[sn.ID] = sn.Shortname
	}

	updatedEmployees := make([]models.Employee, len(employees))
	copy(updatedEmployees, employees)

	for i := range updatedEmployees {
		if shortname, ok := shortNameMap[updatedEmployees[i].ID]; ok {
			updatedEmployees[i].ShortName = shortname
		}
	}

	return updatedEmployees
}

func ParseEmployeeFromBody(in io.ReadCloser, metric *metrics.Metrics,
	tdID, tdFullname, tdPosition, tdEmail, tdPhone int,
) ([]models.Employee, error) {
	var employees []models.Employee

	doc, _ := goquery.NewDocumentFromReader(in)

	doc.Find(`tr[tag^="row_"]`).Each(func(_ int, row *goquery.Selection) {
		employee := models.Employee{}

		employeeID, _ := strconv.Atoi(
			row.Find(fmt.Sprintf("td:nth-child(%d) input", tdID)).AttrOr("value", "0"))
		employee.ID = employeeID

		employee.FullName = strings.TrimSpace(row.Find(fmt.Sprintf("td:nth-child(%d)", tdFullname)).Text())
		employee.Position = strings.TrimSpace(row.Find(fmt.Sprintf("td:nth-child(%d)", tdPosition)).Text())
		employee.Email = strings.TrimSpace(row.Find(fmt.Sprintf("td:nth-child(%d)", tdEmail)).Text())
		employee.Phone = strings.TrimSpace(row.Find(fmt.Sprintf("td:nth-child(%d)", tdPhone)).Text())

		employees = append(employees, employee)
		metric.ItemsParsed.WithLabelValues("employee").Inc()
	})

	return employees, nil
}

func parseIDFromHref(href string) (int, error) {
	parts := strings.Split(href, "&")
	for _, part := range parts {
		if strings.HasPrefix(part, "id=") {
			var identifier int

			_, err := fmt.Sscanf(part, "id=%d", &identifier)
			if err != nil {
				return 0, fmt.Errorf("failed to scan the string '%s':%w", part, err)
			}

			return identifier, nil
		}
	}

	return 0, nil
}
