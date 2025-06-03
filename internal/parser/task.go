package parser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Houeta/us-api-provider/internal/models"
	"github.com/PuerkitoBio/goquery"
)

var ErrScrapeTask = errors.New("failed to scrape tasks")

func ParseTasksByDate(ctx context.Context, client *http.Client, date time.Time, destURL string) ([]models.Task, error) {
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

	resp, err := getHTMLResponse(ctx, client, &data, destURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get response html reposnse: %w", err)
	}
	defer resp.Body.Close()

	return parseTasksFromBody(resp.Body)
}

func ParseTaskTypes(ctx context.Context, client *http.Client, destURL string) ([]string, error) {
	data := url.Values{}
	var taskTypes []string

	// Set data payload
	data.Set("core_section", "task")
	data.Set("action", "group_task_type_list")

	task_id_counter := 3
	for i := range task_id_counter {
		data.Set("id", strconv.Itoa(i+1))

		resp, err := getHTMLResponse(ctx, client, &data, destURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get response which should retrieve task types: %w", err)
		}
		defer resp.Body.Close()

		taskTypesbyID, err := parseTaskTypes(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to parse task types for id '%d': %w", i+1, err)
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

func parseTasksFromBody(in io.ReadCloser) ([]models.Task, error) {
	var tasks []models.Task

	doc, err := goquery.NewDocumentFromReader(in)
	if err != nil {
		return nil, fmt.Errorf("data cannot be parsed as HTML: %w", err)
	}

	doc.Find(`tr[tag^="row_"]`).Each(func(_ int, s *goquery.Selection) {
		task := models.Task{}

		task.ID, err = strconv.Atoi(strings.TrimSpace(s.Find("td:nth-child(7) a").Text()))
		if err != nil {
			log.Printf("Failed to convert task `ID` string to integer type: %v", err)
		}

		task.CreatedAt, err = time.Parse(
			"02.01.2006", strings.TrimSpace(s.Find("td:nth-child(8)").Contents().First().Text()),
		)
		if err != nil {
			log.Printf("failed to convert string createdAt to go type time.Time: %v", err)
		}
		task.ClosedAt, err = time.Parse(
			"02.01.2006", strings.TrimSpace(s.Find("td:nth-child(9)").Contents().First().Text()),
		)
		if err != nil {
			log.Printf("failed to convert string closedAt to go type time.Time: %v", err)
		}

		task.Address = strings.TrimSpace(s.Find("td:nth-child(10)").Text())

		customerInfo, err := s.Find("td:nth-child(11)").Html()
		if err != nil {
			log.Printf("Failed to get customer info: %v", err)
		}

		task.CustomerName, task.CustomerLogin = parseCustomerInfo(customerInfo)
		task.Type = strings.TrimSpace(s.Find("td:nth-child(13) b").First().Text())
		task.Description = strings.TrimSpace(s.Find("td:nth-child(13) .div_journal_opis").Text())

		rawExecutors, err := s.Find("td:nth-child(14)").Html()
		if err != nil {
			log.Printf("Failed to get executors: %v", err)
		}
		task.Executors = parseExecutors(rawExecutors)

		rawComments, err := s.Find("td:nth-child(5)").Html()
		if err != nil {
			log.Printf("Failed to get comments: %v", err)
		}
		task.Comments = parseExecutors(rawComments)

		tasks = append(tasks, task)
	})

	return tasks, nil
}

func getHTMLResponse(ctx context.Context, client *http.Client, data *url.Values, destURL string) (*http.Response, error) {
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

func parseCustomerInfo(rawHTML string) (string, string) {
	const lenParts = 2
	var customerName string
	var customerLogin string

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(rawHTML))
	if err != nil {
		log.Printf("failed to parse customer info: %v", err)
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

	log.Printf("Nothing found in client info: %v", doc.Text())

	return "", ""
}

func parseExecutors(rawHTML string) []string {
	var executors []string

	parts := strings.Split(rawHTML, "<br/>")
	for _, part := range parts {
		text := strings.TrimSpace(part)
		if strings.Contains(text, "<i>") {
			text = strings.Split(text, "<i>")[0]
		}
		if text != "" {
			executors = append(executors, text)
		}
	}

	return executors
}
