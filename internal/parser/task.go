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
	"unicode/utf8"

	"github.com/Houeta/us-api-provider/internal/models"
	"github.com/PuerkitoBio/goquery"
)

var ErrScrapeTask = errors.New("failed to scrape tasks")

type parserConfig struct {
	id          string
	createdAt   string
	closedAt    string
	address     string
	customer    string
	taskType    string
	description string
	executors   string
	comments    string
}

type TaskParser struct {
	client  *http.Client
	log     *slog.Logger
	destURL string
}

type TaskInterface interface {
	ParseTasksByDate(ctx context.Context, date time.Time) ([]models.Task, error)
}

func NewTaskParser(client *http.Client, log *slog.Logger, destURL string) *TaskParser {
	return &TaskParser{client: client, log: log, destURL: destURL}
}

func (tp *TaskParser) ParseTasksByDate(ctx context.Context, date time.Time) ([]models.Task, error) {
	data := url.Values{}

	// Set data payload
	data.Set("core_section", "task_list")
	data.Set("filter_selector0", "task_state")
	data.Set("filter_selector1", "task_group")
	data.Set("task_group1_value", "3")
	data.Set("filter_selector2", "date_update")
	data.Set("date_update2", "3")
	data.Set("date_update2_date1", date.Format("02.01.2006"))
	data.Set("date_update2_date2", date.Format("02.01.2006"))

	completedTasks, err := tp.parseCompletedTasks(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse completed tasks: %w", err)
	}

	uncompletedTasks, err := tp.parseUncompletedTasks(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uncompleted tasks: %w", err)
	}

	return append(completedTasks, uncompletedTasks...), nil
}

func (tp *TaskParser) parseCompletedTasks(ctx context.Context, data url.Values) ([]models.Task, error) {
	data.Set("task_state0_value", "2")

	resp, err := GetHTMLResponse(ctx, tp.client, &data, tp.destURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get html response: %w", err)
	}
	defer resp.Body.Close()

	return tp.parseTasksFromBody(resp.Body, true)
}

func (tp *TaskParser) parseUncompletedTasks(ctx context.Context, data url.Values) ([]models.Task, error) {
	data.Set("task_state0_value", "1")

	resp, err := GetHTMLResponse(ctx, tp.client, &data, tp.destURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get html response: %w", err)
	}
	defer resp.Body.Close()

	return tp.parseTasksFromBody(resp.Body, false)
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

func (tp *TaskParser) parseTasksFromBody(inp io.ReadCloser, isCompleted bool) ([]models.Task, error) {
	var tasks []models.Task
	var err error
	var parseErrors []error
	var config parserConfig

	completedTasksConfig := parserConfig{
		id:          "td:nth-child(7) a",
		createdAt:   "td:nth-child(8)",
		closedAt:    "td:nth-child(9)",
		address:     "td:nth-child(10)",
		customer:    "td:nth-child(11)",
		taskType:    "td:nth-child(13)",
		description: "td:nth-child(13) .div_journal_opis",
		executors:   "td:nth-child(14)",
		comments:    "td:nth-child(5)",
	}

	activeTasksConfig := parserConfig{
		id:          "td:nth-child(7) a",
		createdAt:   "td:nth-child(8)",
		closedAt:    "",
		address:     "td:nth-child(9)",
		customer:    "td:nth-child(10)",
		taskType:    "td:nth-child(12)",
		description: "td:nth-child(12) .div_journal_opis",
		executors:   "td:nth-child(13)",
		comments:    "td:nth-child(5)",
	}

	if isCompleted {
		config = completedTasksConfig
	} else {
		config = activeTasksConfig
	}

	doc, err := goquery.NewDocumentFromReader(inp)
	if err != nil {
		return nil, fmt.Errorf("data cannot be parsed as HTML: %w", err)
	}

	doc.Find(`tr[tag^="row_"]`).Each(func(_ int, row *goquery.Selection) {
		task := models.Task{}

		task.ID, err = extractInt(row, config.id)
		if err != nil {
			tp.log.Debug("Failed to convert task `ID` string to integer type", "error", err)
			parseErrors = append(parseErrors, fmt.Errorf("failed to parse task ID: %w", err))
			return
		}

		task.CreatedAt, err = extractDate(row, config.createdAt, "02.01.2006")
		if err != nil {
			tp.log.Debug("Failed to convert string createdAt to go type time.Time", "id", task.ID, "error", err)
			parseErrors = append(parseErrors, fmt.Errorf("task ID %d: failed to parse CreatedAt: %w", task.ID, err))
		}
		if isCompleted {
			task.ClosedAt, err = extractDate(row, config.closedAt, "02.01.2006")
			if err != nil {
				tp.log.Debug("Failed to convert string closedAt to go type time.Time", "id", task.ID, "error", err)
				parseErrors = append(parseErrors, fmt.Errorf("task ID %d: failed to parse ClosedAt: %w", task.ID, err))
			}
		}

		task.Address = extractText(row, config.address)
		task.Type = extractText(row, config.taskType+" b")
		task.Description = extractText(row, config.description)
		if !utf8.ValidString(task.Description) {
			task.Description = ""
			tp.log.Warn("Description contains invalid UTF-8 symbols, cleared.", "id", task.ID)
		}

		customerHTML, _ := row.Find(config.customer).Html()
		task.CustomerName, task.CustomerLogin = ParseCustomerInfo(customerHTML, tp.log)

		executorsHTML, _ := row.Find(config.executors).Html()
		task.Executors = ParseLinks(executorsHTML)

		commentsHTML, _ := row.Find(config.comments).Html()
		task.Comments = ParseLinks(commentsHTML)

		tasks = append(tasks, task)
	})

	if len(parseErrors) > 0 {
		tp.log.Warn("encountered errors during parsing", "count", len(parseErrors), "first error", parseErrors[0])
	}

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

func ParseLinks(rawHTML string) []string {
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

func extractText(selection *goquery.Selection, selector string) string {
	return strings.TrimSpace(selection.Find(selector).Text())
}

func extractInt(selection *goquery.Selection, selector string) (int, error) {
	text := extractText(selection, selector)
	if text == "" {
		return 0, errors.New("element not found or text is empty")
	}

	integer, err := strconv.Atoi(text)
	if err != nil {
		return integer, fmt.Errorf("failed to convert string to integer: %w", err)
	}

	return integer, nil
}

func extractDate(selection *goquery.Selection, selector string, layout string) (time.Time, error) {
	text := strings.TrimSpace(selection.Find(selector).Contents().First().Text())
	if text == "" {
		return time.Time{}, errors.New("element not found or text is empty")
	}

	date, err := time.Parse(layout, text)
	if err != nil {
		return date, fmt.Errorf("faile to parse date from string: %w", err)
	}

	return date, nil
}
