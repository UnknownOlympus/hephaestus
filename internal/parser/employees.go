package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Houeta/us-api-provider/internal/models"
	"github.com/PuerkitoBio/goquery"
)

var ErrScrapeEmployee = errors.New("failed to scrape employee")

type employeeShortname struct {
	ID        int
	Shortname string
}

func ParseEmployees(ctx context.Context, client *http.Client, destURL string) ([]models.Employee, error) {
	staff, err := ParseStaff(ctx, client, destURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse staff: %w", err)
	}

	staffShortName, err := ParseStaffShortNames(ctx, client, destURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse staff short names: %w", err)
	}

	return UpdateEmployeeShortNames(staff, staffShortName), nil
}

// ParseEmployees parses employee data from the specified destination URL using the provided HTTP client.
// It returns a slice of models.Employee and an error if any.
func ParseStaff(ctx context.Context, client *http.Client, destURL string) ([]models.Employee, error) {
	data := url.Values{}

	data.Set("core_section", "staff_unit")

	reqURL, err := url.Parse(destURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse destination URL %s: %w", destURL, err)
	}

	reqURL.RawQuery = data.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new request %s: %w", reqURL.String(), err)
	}

	req.Header.Set("User-Agent", models.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to request %s: %w", destURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w, received status code: %d", ErrScrapeEmployee, resp.StatusCode)
	}

	return ParseEmployeeFromBody(resp.Body)
}

func ParseStaffShortNames(ctx context.Context, client *http.Client, destURL string) ([]employeeShortname, error) {
	data := url.Values{}
	var employees []employeeShortname

	data.Set("core_section", "staff")
	data.Set("action", "division")

	resp, err := getHTMLResponse(ctx, client, &data, destURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get response which should retrieve task types: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("data cannot be parsed as HTML: %w", err)
	}

	doc.Find(`div.div_space`).Each(func(i int, div *goquery.Selection) {
		link := div.Find("a")
		if link.Length() > 0 {
			href, exists := link.Attr("href")
			if exists {
				id, err := parseIDFromHref(href)
				if id > 0 && err == nil {
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

func UpdateEmployeeShortNames(employees []models.Employee, shortnames []employeeShortname) []models.Employee {
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

// ParseEmployeeFromBody parses the employee data from the provided io.ReadCloser and returns a slice of models.Employee.
func ParseEmployeeFromBody(in io.ReadCloser) ([]models.Employee, error) {
	var employees []models.Employee

	doc, err := goquery.NewDocumentFromReader(in)
	if err != nil {
		return nil, fmt.Errorf("data cannot be parsed as HTML: %w", err)
	}

	doc.Find(`tr[tag^="row_"]`).Each(func(_ int, row *goquery.Selection) {
		employee := models.Employee{}

		employeeID, employeeErr := strconv.Atoi(row.Find("td:nth-child(2) input").AttrOr("value", "0"))
		if employeeErr != nil {
			return
		}
		employee.ID = employeeID

		employee.FullName = strings.TrimSpace(row.Find("td:nth-child(3)").Text())
		employee.Position = strings.TrimSpace(row.Find("td:nth-child(4)").Text())
		employee.Email = strings.TrimSpace(row.Find("td:nth-child(5)").Text())
		employee.Phone = strings.TrimSpace(row.Find("td:nth-child(6)").Text())

		employees = append(employees, employee)
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
