package cmd

import "testing"

func TestParseThemeSetArgs_collectsItemsByKind(t *testing.T) {
	// Given
	args := []string{
		"res://ui/theme.tres",
		"--type", "Label",
		"--color", "font_color=Color(0.3, 0.8, 1, 1)",
		"--constant", "line_spacing=4",
		"--font-size", "font_size=16",
	}

	// When
	params, err := parseThemeSetArgs(args)

	// Then
	if err != nil {
		t.Fatalf("parseThemeSetArgs error: %v", err)
	}
	if params["action"] != "set" || params["path"] != "res://ui/theme.tres" || params["type"] != "Label" {
		t.Fatalf("params = %#v", params)
	}
	colors, _ := params["colors"].(map[string]any)
	if colors["font_color"] != "Color(0.3, 0.8, 1, 1)" {
		t.Fatalf("colors = %#v, want the Color text kept intact", colors)
	}
	constants, _ := params["constants"].(map[string]any)
	if constants["line_spacing"] != "4" {
		t.Fatalf("constants = %#v", constants)
	}
	sizes, _ := params["font_sizes"].(map[string]any)
	if sizes["font_size"] != "16" {
		t.Fatalf("font_sizes = %#v", sizes)
	}
}

func TestParseThemeSetArgs_requiresTypeAndAtLeastOneItem(t *testing.T) {
	// Given / When / Then
	if _, err := parseThemeSetArgs([]string{"res://ui/theme.tres", "--color", "font_color=Color(1, 1, 1, 1)"}); err == nil {
		t.Fatal("expected an error when --type is missing")
	}
	if _, err := parseThemeSetArgs([]string{"res://ui/theme.tres", "--type", "Label"}); err == nil {
		t.Fatal("expected an error when no item flag is given")
	}
}

func TestSplitThemeItem_splitsOnFirstEqualsOnly(t *testing.T) {
	// Given: a Color value is comma-and-paren heavy; only the first "=" separates.
	name, value, err := splitThemeItem("--color", "font_color=Color(0.5, 0.5, 0.5, 1)")

	// Then
	if err != nil {
		t.Fatalf("splitThemeItem error: %v", err)
	}
	if name != "font_color" || value != "Color(0.5, 0.5, 0.5, 1)" {
		t.Fatalf("got (%q, %q)", name, value)
	}
}

func TestSplitThemeItem_rejectsMalformedPairs(t *testing.T) {
	for _, arg := range []string{"font_color", "=Color(1, 1, 1, 1)", "font_color="} {
		if _, _, err := splitThemeItem("--color", arg); err == nil {
			t.Fatalf("expected an error for %q", arg)
		}
	}
}

func TestParseThemeGetArgs_acceptsOptionalTypeFilter(t *testing.T) {
	// Given / When
	params, err := parseThemeGetArgs([]string{"res://ui/theme.tres", "--type", "Button"})

	// Then
	if err != nil {
		t.Fatalf("parseThemeGetArgs error: %v", err)
	}
	if params["action"] != "get" || params["type"] != "Button" {
		t.Fatalf("params = %#v", params)
	}

	bare, err := parseThemeGetArgs([]string{"res://ui/theme.tres"})
	if err != nil {
		t.Fatalf("parseThemeGetArgs error: %v", err)
	}
	if _, ok := bare["type"]; ok {
		t.Fatalf("bare get should not set a type filter: %#v", bare)
	}
}

func TestParseThemeArgs_rejectsUnknownSubcommand(t *testing.T) {
	if _, err := parseThemeArgs([]string{"delete", "res://ui/theme.tres"}); err == nil {
		t.Fatal("expected an error for an unknown subcommand")
	}
}
