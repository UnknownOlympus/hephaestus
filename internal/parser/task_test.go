package parser_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Houeta/us-api-provider/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// HTML mock for the task page.
const completedTasksHTML = `
<table>
    <tbody>
        <tr tag="row_12345">
            <td></td>
			<td>Scheduled work</td>
			<td></td><td></td>
            <td>Comment 1<br/>Comment 2</td>
            <td></td>
            <td><a>12345</a></td>
            <td>01.06.2025</td>
            <td>02.06.2025</td>
            <td>Test street, 1</td>
            <td><a href="#">Test Client - testlogin</a></td>
            <td></td>
            <td><b>Scheduled work</b><div class="div_journal_opis">Task description</div></td>
            <td>Executor 1<br/>Executor 2<i>(additional)</i></td>
        </tr>
        <tr tag="row_12346">
            <td></td>
			<td>Emergency work</td>
			<td></td><td></td>
            <td></td>
            <td></td>
            <td><a>12346</a></td>
            <td>03.06.2025</td>
            <td>04.06.2025</td>
            <td>Central str, 5</td>
            <td>only client name</td>
            <td></td>
            <td><b>Emergency work</b><div class="div_journal_opis">Other description</div></td>
            <td>Executor 1</td>
        </tr>
		<tr tag="row_invalid">
			<td></td>
			<td>Scheduled work</td>
			<td></td><td></td>
            <td></td>
            <td></td>
            <td><a>not-an-integer</a></td>
            <td>not-a-date</td>
            <td>not-a-date</td>
            <td></td><td></td><td></td><td></td><td></td>
        </tr>
		<tr tag="row_invalid">
            <td></td><td></td><td></td><td></td>
            <td></td>
            <td></td>
            <td><a></a></td>
            <td>not-a-date</td>
            <td>not-a-date</td>
            <td></td><td></td><td></td><td></td><td></td>
        </tr>
        <tr tag="row_Invalid">
            <td></td><td></td><td></td><td></td>
            <td></td>
            <td></td>
            <td><a>12346</a></td>
            <td></td>
            <td>asdasdasd</td>
            <td>Central str, 5</td>
            <td>only client name</td>
            <td></td>
            <td><b>Emergency work</b><div class="div_journal_opis">Other �description</div></td>
            <td>Executor 1</td>
        </tr>
    </tbody>
</table>`

const uncompletedTasksHTML = `
<table>
    <tbody>
        <tr tag="row_12345">
			<td></td>
			<td>Scheduled work</td>
			<td></td><td></td>
            <td>Comment 1<br/>Comment 2</td>
            <td></td>
            <td><a>12345</a></td>
            <td>01.06.2025</td>
            <td>Test street, 1</td>
            <td><a href="#">Test Client - testlogin</a></td>
            <td></td>
            <td><b>Scheduled work</b><div class="div_journal_opis">Task description</div></td>
            <td>Executor 1<br/>Executor 2<i>(additional)</i></td>
        </tr>
		<tr tag="row_invalid">
            <td></td><td></td><td></td><td></td>
            <td></td>
            <td></td>
            <td><a>not-an-integer</a></td>
            <td>not-a-date</td>
            <td>not-a-date</td>
            <td></td><td></td><td></td><td></td><td></td>
        </tr>
    </tbody>
</table>`

// HTML-mock for the page with task types.
const taskTypesHTML = `
<div>
    <a title="Добавить задание">Task Type 1</a>
    <a title="Добавить задание">Task Type 2</a>
    <a title="Other title">Task type 3</a>
</div>
`

// TestParseTasksByDate checks the main function of getting tasks.
func TestParseTasksByDate(t *testing.T) {
	t.Parallel()

	// We create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Checking whether the request contains the expected parameters
		assert.Equal(t, "task_list", r.URL.Query().Get("core_section"))
		assert.Equal(t, "07.06.2025", r.URL.Query().Get("date_update2_date1"))

		w.WriteHeader(http.StatusOK)
		if r.URL.Query().Get("task_state0_value") == "2" {
			_, _ = w.Write([]byte(completedTasksHTML))
		} else {
			_, _ = w.Write([]byte(uncompletedTasksHTML))
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil)) // Using default logger
	testDate, _ := time.Parse("02.01.2006", "07.06.2025")
	taskParser := parser.NewTaskParser(server.Client(), logger, server.URL)

	tasks, err := taskParser.ParseTasksByDate(context.Background(), testDate)

	require.NoError(t, err)
	// We expect 3 tasks because one of them has invalid data but does not cause an error
	require.Len(t, tasks, 4, "4 tasks must be found")

	// Checking the first, fully completed task
	task1 := tasks[0]
	assert.Equal(t, 12345, task1.ID)
	assert.Equal(t, "01.06.2025", task1.CreatedAt.Format("02.01.2006"))
	assert.Equal(t, "02.06.2025", task1.ClosedAt.Format("02.01.2006"))
	assert.Equal(t, "Test street, 1", task1.Address)
	assert.Equal(t, "Test Client", task1.CustomerName)
	assert.Equal(t, "testlogin", task1.CustomerLogin)
	assert.Equal(t, "Scheduled work", task1.Type)
	assert.Equal(t, "Task description", task1.Description)
	assert.Equal(t, []string{"Comment 1", "Comment 2"}, task1.Comments)
	assert.Equal(t, []string{"Executor 1", "Executor 2"}, task1.Executors)

	// Checking the second task with partial data
	task2 := tasks[1]
	assert.Equal(t, 12346, task2.ID)
	assert.Equal(t, "only client name", task2.CustomerName)
	assert.Equal(t, "n/a", task2.CustomerLogin) // Expected 'n/a' by default
	assert.Equal(t, []string{"Executor 1"}, task2.Executors)

	// Verify that invalid data did not cause a crash but created empty fields
	task3 := tasks[3]
	assert.Equal(t, 12345, task3.ID)
	assert.Equal(t, "01.06.2025", task3.CreatedAt.Format("02.01.2006"))
	assert.Equal(t, "02.06.2025", task1.ClosedAt.Format("02.01.2006"))
	assert.Equal(t, "Test street, 1", task3.Address)
	assert.Equal(t, "Test Client", task3.CustomerName)
	assert.Equal(t, "testlogin", task3.CustomerLogin)
	assert.Equal(t, "Scheduled work", task3.Type)
	assert.Equal(t, "Task description", task3.Description)
	assert.Equal(t, []string{"Comment 1", "Comment 2"}, task3.Comments)
	assert.Equal(t, []string{"Executor 1", "Executor 2"}, task3.Executors)
}

// TestParseTaskTypes checks for receipt of task types.
func TestParseTaskTypes(t *testing.T) {
	t.Parallel()

	requestCount := 0
	// mock-server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		id := r.URL.Query().Get("id")
		assert.NotEmpty(t, id, "ID cannot be empty")

		// Return the same HTML for all requests for simplicity
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(taskTypesHTML))
	}))
	defer server.Close()

	taskTypes, err := parser.ParseTaskTypes(context.Background(), server.Client(), server.URL)

	require.NoError(t, err)
	// The function makes 3 queries, each returning 2 task types. Total 3 * 2 = 6
	assert.Len(t, taskTypes, 6, "6 task types were expected (2 types * 3 requests)")
	assert.Equal(t, 3, requestCount, "3 HTTP requests must be made")
	assert.Equal(t, "Task Type 1", taskTypes[0])
	assert.Equal(t, "Task Type 2", taskTypes[1])
}

// unit-test for parseExecutors.
func TestParseExecutors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		rawHTML  string
		expected []string
	}{
		{
			name:     "Several executors",
			rawHTML:  "John Doe<br/>Alex Ray",
			expected: []string{"John Doe", "Alex Ray"},
		},
		{
			name:     "Artist with additional information in <i>",
			rawHTML:  "Johny Sins <i>(main)</i><br/>Baby Yoda",
			expected: []string{"Johny Sins", "Baby Yoda"},
		},
		{
			name:     "Single executor",
			rawHTML:  "Yurii Relitskyi",
			expected: []string{"Yurii Relitskyi"},
		},
		{
			name:     "Empty row",
			rawHTML:  "",
			expected: nil, // Expected nil
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, parser.ParseLinks(tc.rawHTML))
		})
	}
}

// unit-test fot parseCustomerInfo.
func TestParseCustomerInfo(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("Name and login", func(t *testing.T) {
		t.Parallel()
		html := `<a href="#">Test client - client123</a>`
		name, login := parser.ParseCustomerInfo(html, logger)
		assert.Equal(t, "Test client", name)
		assert.Equal(t, "client123", login)
	})

	t.Run("Only name", func(t *testing.T) {
		t.Parallel()
		html := `Just client`
		name, login := parser.ParseCustomerInfo(html, logger)
		assert.Equal(t, "Just client", name)
		assert.Equal(t, "n/a", login)
	})

	t.Run("Empty row", func(t *testing.T) {
		t.Parallel()
		html := ``
		name, login := parser.ParseCustomerInfo(html, logger)
		assert.Empty(t, name)
		assert.Empty(t, login)
	})
}

func TestParseTasksbyDate_CompletedResponseError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil)) // Using default logger
	testDate, _ := time.Parse("02.01.2006", "07.06.2025")
	taskParser := parser.NewTaskParser(server.Client(), logger, server.URL)

	_, err := taskParser.ParseTasksByDate(context.Background(), testDate)
	require.Error(t, err)
	require.ErrorIs(t, err, parser.ErrScrapeTask)
	assert.ErrorContains(t, err, "failed to get html response")
}

func TestParseTasksbyDate_UncompletedResponseError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("task_state0_value") == "2" {
			_, _ = w.Write([]byte(completedTasksHTML))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil)) // Using default logger
	testDate, _ := time.Parse("02.01.2006", "07.06.2025")
	taskParser := parser.NewTaskParser(server.Client(), logger, server.URL)

	_, err := taskParser.ParseTasksByDate(context.Background(), testDate)
	require.Error(t, err)
	require.ErrorIs(t, err, parser.ErrScrapeTask)
	assert.ErrorContains(t, err, "failed to get html response")
}

func TestParseTaskTypes_ResponseError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := parser.ParseTaskTypes(context.Background(), server.Client(), server.URL)
	require.Error(t, err)
	require.ErrorIs(t, err, parser.ErrScrapeTask)
	assert.ErrorContains(t, err, "failed to get response which should retrieve task types")
}
