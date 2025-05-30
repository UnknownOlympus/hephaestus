package parser_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Houeta/us-api-provider/internal/models"
	"github.com/Houeta/us-api-provider/internal/parser"
)

// Helper function for RoundTripper mocking.
type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestParseEmployees_Success(t *testing.T) {
	// Create mock http server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that request has right parameters
		if r.URL.Query().Get("core_section") != "staff_unit" {
			t.Errorf("Expected 'core_section' parameter to be 'staff_unit', got %s", r.URL.Query().Get("core_section"))
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		if r.Header.Get("User-Agent") != models.UserAgent {
			t.Errorf("Expected User-Agent to be %s, got %s", models.UserAgent, r.Header.Get("User-Agent"))
		}

		// mock http response
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`
			<table>
				<tr tag="row_1">
					<td></td>
					<td><input value="101"></td>
					<td> John Doe </td>
					<td>Software Engineer</td>
					<td>john.doe@example.com</td>
					<td>123-456-7890</td>
				</tr>
				<tr tag="row_2">
					<td></td>
					<td><input value="102"></td>
					<td> Jane Smith </td>
					<td>Project Manager</td>
					<td>jane.smith@example.com</td>
					<td>987-654-3210</td>
				</tr>
				<tr tag="row_non_numeric">
					<td></td>
					<td><input value="abc"></td>
					<td> Invalid ID </td>
					<td>Tester</td>
					<td>invalid.id@example.com</td>
					<td>111-222-3333</td>
				</tr>
			</table>
		`))
		if err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer ts.Close()

	client := ts.Client()
	ctx := context.Background()

	employees, err := parser.ParseEmployees(ctx, client, ts.URL)
	if err != nil {
		t.Fatalf("ParseEmployees returned an error: %v", err)
	}

	if len(employees) != 2 { // Expected 2 employyes, because one has undigital ID
		t.Fatalf("Expected 2 employees, got %d", len(employees))
	}

	// Check first employee
	expectedEmployee1 := models.Employee{
		ID:       101,
		FullName: "John Doe",
		Position: "Software Engineer",
		Email:    "john.doe@example.com",
		Phone:    "123-456-7890",
	}
	if employees[0] != expectedEmployee1 {
		t.Errorf("Expected employee 1: %+v, got %+v", expectedEmployee1, employees[0])
	}

	// Check second employee
	expectedEmployee2 := models.Employee{
		ID:       102,
		FullName: "Jane Smith",
		Position: "Project Manager",
		Email:    "jane.smith@example.com",
		Phone:    "987-654-3210",
	}
	if employees[1] != expectedEmployee2 {
		t.Errorf("Expected employee 2: %+v, got %+v", expectedEmployee2, employees[1])
	}
}

func TestParseEmployees_HTTPRequestError(t *testing.T) {
	// Create client which always returning error
	client := &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, http.ErrHandlerTimeout // Example error
		}),
	}
	ctx := context.Background()

	_, err := parser.ParseEmployees(ctx, client, "http://example.com")
	if err == nil {
		t.Error("Expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), "failed to request") {
		t.Errorf("Expected error to contain 'failed to request', got %v", err)
	}
}

func TestParseEmployees_InvalidURL(t *testing.T) {
	client := &http.Client{}
	ctx := context.Background()

	_, err := parser.ParseEmployees(ctx, client, "://invalid-url")
	if err == nil {
		t.Error("Expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse destination URL") {
		t.Errorf("Expected error to contain 'failed to parse destination URL', got %v", err)
	}
}

func TestParseEmployees_Non200StatusCode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // Returns server error
	}))
	defer ts.Close()

	client := ts.Client()
	ctx := context.Background()

	_, err := parser.ParseEmployees(ctx, client, ts.URL)
	if err == nil {
		t.Error("Expected an error, but got nil")
	}
	if !errors.Is(err, parser.ErrScrapeEmployee) {
		t.Errorf("Expected error to be ErrScrapeEmployee, got %v", err)
	}
	if !strings.Contains(err.Error(), "received status code: 500") {
		t.Errorf("Expected error to contain 'received status code: 500', got %v", err)
	}
}

func TestParseEmployees_EmptyBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Empty body response
	}))
	defer ts.Close()

	client := ts.Client()
	ctx := context.Background()

	employees, err := parser.ParseEmployees(ctx, client, ts.URL)
	if err != nil {
		t.Fatalf("ParseEmployees returned an error for empty body: %v", err)
	}

	if len(employees) != 0 {
		t.Errorf("Expected 0 employees for empty body, got %d", len(employees))
	}
}

func TestParseEmployeeFromBody_NoRows(t *testing.T) {
	htmlContent := `
		<table>
			<tr>
				<td>No rows with tag attribute</td>
			</tr>
		</table>
	`
	reader := io.NopCloser(strings.NewReader(htmlContent))

	employees, err := parser.ParseEmployeeFromBody(reader)
	if err != nil {
		t.Fatalf("parseEmployeeFromBody returned an error: %v", err)
	}

	if len(employees) != 0 {
		t.Errorf("Expected 0 employees when no matching rows, got %d", len(employees))
	}
}
