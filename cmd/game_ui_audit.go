package cmd

import (
	"fmt"
	"os"
)

func runGameUIAudit(params map[string]any) int {
	c, err := dialEditor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "game ui audit: %v\n", err)
		return 1
	}
	resp, err := c.Post("game", params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "game ui audit: %v\n", err)
		return 1
	}
	if !resp.OK {
		fmt.Fprintf(os.Stderr, "game ui audit: %s\n", resp.Error)
		return 1
	}
	passed, err := gameUIAuditPassed(resp.Data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "game ui audit: %v\n", err)
		return 1
	}
	if code := printData(resp); code != 0 {
		return code
	}
	if !passed {
		return 1
	}
	return 0
}

func gameUIAuditPassed(data any) (bool, error) {
	values, ok := data.(map[string]any)
	if !ok {
		return false, fmt.Errorf("unexpected response data")
	}
	passed, ok := values["ok"].(bool)
	if !ok {
		return false, fmt.Errorf("response is missing boolean ok")
	}
	return passed, nil
}

func gameUIAuditFailure(data any) string {
	values, ok := data.(map[string]any)
	if !ok {
		return "UI audit failed"
	}
	errors, _ := numericField(values, "errors")
	warnings, _ := numericField(values, "warnings")
	return fmt.Sprintf(
		"UI audit failed: %d %s, %d %s",
		errors,
		countLabel(errors, "error"),
		warnings,
		countLabel(warnings, "warning"),
	)
}

func countLabel(count int, singular string) string {
	if count == 1 {
		return singular
	}
	return singular + "s"
}
