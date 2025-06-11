package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Houeta/us-api-provider/internal/models"
	"github.com/PuerkitoBio/goquery"
)

var ErrScrapeTask = errors.New("failed to scrape tasks")

func ParseTasksByDate(
	ctx context.Context,
	log *slog.Logger,
	client *http.Client,
	date time.Time,
	destURL string,
) ([]models.Task, error) {
	data := url.Values{}

	// Set data payload
	data.Set("core_section", "task_list")
	data.Set("filter_selector0", "task_state")
	data.Set("task_state0_value", "2")
	data.Set("filter_selector1", "task_group")
	data.Set("task_group1_value", "3")
	data.Set("filter_selector2", "date_update")
	data.Set("date_update2", "3")
	data.Set("date_update2_date1", date.Format("02.01.2006"))
	data.Set("date_update2_date2", date.Format("02.01.2006"))

	resp, err := GetHTMLResponse(ctx, client, &data, destURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get html response: %w", err)
	}
	defer resp.Body.Close()

	return parseTasksFromBody(resp.Body, log)
}

func ParseTaskTypes(ctx context.Context, client *http.Client, destURL string) ([]string, error) {
	data := url.Values{}
	var taskTypes []string

	// Set data payload
	data.Set("core_section", "task")
	data.Set("action", "group_task_type_list")

	taskIDCounter := 3
	for index := range taskIDCounter {
		data.Set("id", strconv.Itoa(index+1))

		resp, err := GetHTMLResponse(ctx, client, &data, destURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get response which should retrieve task types: %w", err)
		}
		defer resp.Body.Close()

		taskTypesbyID, err := parseTaskTypes(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to parse task types for id '%d': %w", index+1, err)
		}

		taskTypes = append(taskTypes, taskTypesbyID...)
	}

	return taskTypes, nil
}

func parseTaskTypes(in io.ReadCloser) ([]string, error) {
	var taskTypes []string

	doc, err := goquery.NewDocumentFromReader(in)
	if err != nil {
		return nil, fmt.Errorf("data cannot be parsed as HTML: %w", err)
	}

	doc.Find(`a[title="Добавить задание"]`).Each(func(_ int, s *goquery.Selection) {
		taskTypeName := strings.TrimSpace(s.Text())

		taskTypes = append(taskTypes, taskTypeName)
	})

	return taskTypes, nil
}

func parseTasksFromBody(in io.ReadCloser, log *slog.Logger) ([]models.Task, error) {
	var tasks []models.Task
	var err error

	doc, err := goquery.NewDocumentFromReader(in)
	if err != nil {
		return nil, fmt.Errorf("data cannot be parsed as HTML: %w", err)
	}

	doc.Find(`tr[tag^="row_"]`).Each(func(_ int, row *goquery.Selection) {
		task := models.Task{}

		task.ID, err = strconv.Atoi(strings.TrimSpace(row.Find("td:nth-child(7) a").Text()))
		if err != nil {
			log.Debug("Failed to convert task `ID` string to integer type", "error", err)
		}

		task.CreatedAt, err = time.Parse(
			"02.01.2006", strings.TrimSpace(row.Find("td:nth-child(8)").Contents().First().Text()),
		)
		if err != nil {
			log.Debug("Failed to convert string createdAt to go type time.Time", "error", err)
		}
		task.ClosedAt, err = time.Parse(
			"02.01.2006", strings.TrimSpace(row.Find("td:nth-child(9)").Contents().First().Text()),
		)
		if err != nil {
			log.Debug("Failed to convert string closedAt to go type time.Time", "error", err)
		}

		task.Address = strings.TrimSpace(row.Find("td:nth-child(10)").Text())

		customerInfo, errCustomer := row.Find("td:nth-child(11)").Html()
		if errCustomer != nil {
			log.Debug("Failed to get customer info", "error", err)
		}

		task.CustomerName, task.CustomerLogin = ParseCustomerInfo(customerInfo, log)
		task.Type = strings.TrimSpace(row.Find("td:nth-child(13) b").First().Text())
		task.Description = strings.TrimSpace(row.Find("td:nth-child(13) .div_journal_opis").Text())

		rawExecutors, errExecutor := row.Find("td:nth-child(14)").Html()
		if errExecutor != nil {
			log.Debug("Failed to get executors", "error", err)
		}
		task.Executors = ParseExecutors(rawExecutors)

		rawComments, errComments := row.Find("td:nth-child(5)").Html()
		if errComments != nil {
			log.Debug("Failed to get comments", "error", err)
		}
		task.Comments = ParseExecutors(rawComments)

		tasks = append(tasks, task)
	})

	return tasks, nil
}

func GetHTMLResponse(
	ctx context.Context,
	client *http.Client,
	data *url.Values,
	destURL string,
) (*http.Response, error) {
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w, received status code: %d", ErrScrapeTask, resp.StatusCode)
	}

	return resp, nil
}

func ParseCustomerInfo(rawHTML string, log *slog.Logger) (string, string) {
	const lenParts = 2
	var customerName string
	var customerLogin string

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		log.Debug("failed to parse customer info", "error", err)
		return "", ""
	}

	customerLoginNode := doc.Find("a")
	if customerLoginNode.Length() != 0 {
		customerData := strings.TrimSpace(customerLoginNode.Text())

		parts := strings.Split(customerData, "-")
		if len(parts) == lenParts {
			customerName = strings.TrimSpace(parts[0])
			customerLogin = strings.TrimSpace(parts[1])

			return customerName, customerLogin
		}
	}

	customerName = strings.TrimSpace(doc.Text())
	if customerName != "" {
		return customerName, "n/a"
	}

	return "", ""
}

func ParseExecutors(rawHTML string) []string {
	var executors []string

	parts := strings.Split(rawHTML, "<br/>")
	for _, part := range parts {
		text := strings.TrimSpace(part)
		if strings.Contains(text, "<i>") {
			text = strings.Split(text, "<i>")[0]
		}
		if text != "" {
			executors = append(executors, strings.TrimSpace(text))
		}
	}

	return executors
}
