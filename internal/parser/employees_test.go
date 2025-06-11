package parser_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Houeta/us-api-provider/internal/models"
	"github.com/Houeta/us-api-provider/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	staffHTML = `
		<table>
			<tr tag="row_101">
				<td></td>
				<td><input value="101"></td>
				<td>John Doe</td>
				<td>Software Engineer</td>
				<td>john.doe@example.com</td>
				<td>123-456-7890</td>
			</tr>
			<tr tag="row_102">
				<td></td>
				<td><input value="102"></td>
				<td>Jane Smith</td>
				<td>Project Manager</td>
				<td>jane.smith@example.com</td>
				<td>987-654-3210</td>
			</tr>
		</table>`

	dismissedStaffHTML = `
		<table>
			<tr tag="row_103">
				<td></td>
				<td></td>
				<td><input value="103"></td>
				<td>Alex Ray</td>
				<td>Former Developer</td>
				<td>alex.ray@example.com</td>
				<td>555-555-5555</td>
			</tr>
		</table>`

	shortNamesHTML = `
		<div>
			<div class="div_space">
				<a href="?core_section=staff&action=show&id=101">JohnD</a>
			</div>
			<div class="div_space">
				<a href="?core_section=staff&action=show&id=103">AlexR</a>
			</div>
			<div class="div_space">
				<a href="?core_section=staff&action=show&id=999">NoEmployeeForThis</a>
			</div>
		</div>
	`

	invalidShortNamesHTML = `
	<div>
		<div class="div_space">
			<a href="?core_section=staff&action=show&id={\'error\'}">JohnD</a>
		</div>
		<div class="div_space">
			<a href="?core_section=staff&action=show&if=as@#$#@dasd">AlexR</a>
		</div>
		<div class="div_space">
			<a href="?core_section=staff&action=show&id=
		</div>
	</div>
`
)

func TestParseEmployees_Success(t *testing.T) {
	t.Parallel()
	// Create mock http server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		assert.NoError(t, err)

		coreSection := r.Form.Get("core_section")
		isWithLeaved := r.Form.Get("is_with_leaved")
		action := r.Form.Get("action")

		// Check that request has right parameters
		switch {
		case coreSection == "staff_unit" && isWithLeaved == "":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(staffHTML))
		case coreSection == "staff_unit" && isWithLeaved == "1":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(dismissedStaffHTML))
		case coreSection == "staff" && action == "division":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(shortNamesHTML))
		default:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Bad request parameters"))
		}
	}))
	defer ts.Close()

	eparser := parser.NewEmployeeParser(ts.Client(), ts.URL)
	employees, err := eparser.ParseEmployees(context.Background())
	require.NoError(t, err)
	require.Len(t, employees, 3, "Expected sum of active and terminated employees")

	employeeMap := make(map[int]models.Employee)
	for _, e := range employees {
		employeeMap[e.ID] = e
	}

	// Check first employee
	expectedJohn := models.Employee{
		ID:        101,
		FullName:  "John Doe",
		Position:  "Software Engineer",
		Email:     "john.doe@example.com",
		Phone:     "123-456-7890",
		ShortName: "JohnD",
	}
	assert.Equal(t, expectedJohn, employeeMap[101])

	// Check second employee
	expectedJane := models.Employee{
		ID:        102,
		FullName:  "Jane Smith",
		Position:  "Project Manager",
		Email:     "jane.smith@example.com",
		Phone:     "987-654-3210",
		ShortName: "",
	}
	assert.Equal(t, expectedJane, employeeMap[102])

	// Check third employee (terminated)
	expectedAlex := models.Employee{
		ID:        103,
		FullName:  "Alex Ray",
		Position:  "Former Developer",
		Email:     "alex.ray@example.com",
		Phone:     "555-555-5555",
		ShortName: "AlexR",
	}
	assert.Equal(t, expectedAlex, employeeMap[103])
}

func TestParseEmployees_Failures(t *testing.T) {
	t.Parallel()
	// Scenario 1: Error getting active employees
	t.Run("fail on parse staff", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		eparser := parser.NewEmployeeParser(server.Client(), server.URL)
		_, err := eparser.ParseEmployees(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse staff")
	})

	// Scenario 2: Error getting terminated employees
	t.Run("fail on parse dismissed staff", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseForm()
			assert.NoError(t, err)
			if r.Form.Get("is_with_leaved") == "1" {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(staffHTML))
			}
		}))
		defer server.Close()

		eparser := parser.NewEmployeeParser(server.Client(), server.URL)
		_, err := eparser.ParseEmployees(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse dismissed staff")
	})

	// Scenario 3: Error getting short names
	t.Run("fail on parse short names", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseForm()
			assert.NoError(t, err)
			if r.Form.Get("action") == "division" {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
				if r.Form.Get("is_with_leaved") == "1" {
					_, _ = w.Write([]byte(dismissedStaffHTML))
				} else {
					_, _ = w.Write([]byte(staffHTML))
				}
			}
		}))
		defer server.Close()

		eparser := parser.NewEmployeeParser(server.Client(), server.URL)
		_, err := eparser.ParseEmployees(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse staff short names")
	})

	t.Run("fail on parse ID from href", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseForm()
			assert.NoError(t, err)
			if r.Form.Get("action") == "division" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(invalidShortNamesHTML))
			} else {
				w.WriteHeader(http.StatusOK)
				if r.Form.Get("is_with_leaved") == "1" {
					_, _ = w.Write([]byte(dismissedStaffHTML))
				} else {
					_, _ = w.Write([]byte(staffHTML))
				}
			}
		}))
		defer server.Close()

		eparser := parser.NewEmployeeParser(server.Client(), server.URL)
		_, err := eparser.ParseEmployees(context.Background())
		require.NoError(t, err)
	})
}

func TestParseEmployeeFromBody(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		t.Parallel()
		reader := io.NopCloser(strings.NewReader(staffHTML))
		employees, err := parser.ParseEmployeeFromBody(reader, 2, 3, 4, 5, 6)

		require.NoError(t, err)
		require.Len(t, employees, 2)
		assert.Equal(t, "John Doe", employees[0].FullName)
		assert.Equal(t, 102, employees[1].ID)
	})

	t.Run("no rows with matching tag", func(t *testing.T) {
		t.Parallel()
		htmlContent := `<table><tr><td>No rows with tag attribute</td></tr></table>`
		reader := io.NopCloser(strings.NewReader(htmlContent))
		employees, err := parser.ParseEmployeeFromBody(reader, 1, 2, 3, 4, 5)

		require.NoError(t, err)
		assert.Empty(t, employees, 0)
	})
}

// Helper function for RoundTripper mocking.
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestParseEmployees_HTTPRequestError(t *testing.T) {
	t.Parallel()

	// Create client which always returning error
	client := &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, http.ErrHandlerTimeout // Example error
		}),
	}
	ctx := context.Background()

	eparser := parser.NewEmployeeParser(client, "http://example.com")
	_, err := eparser.ParseEmployees(ctx)
	if err == nil {
		t.Error("Expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), "failed to request") {
		t.Errorf("Expected error to contain 'failed to request', got %v", err)
	}
}

func TestParseEmployees_InvalidURL(t *testing.T) {
	t.Parallel()
	client := &http.Client{}
	ctx := context.Background()

	eparser := parser.NewEmployeeParser(client, "://invalid-url")
	_, err := eparser.ParseEmployees(ctx)
	if err == nil {
		t.Error("Expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse destination URL") {
		t.Errorf("Expected error to contain 'failed to parse destination URL', got %v", err)
	}
}
