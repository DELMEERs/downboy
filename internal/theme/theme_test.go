package theme

import (
	"testing"
)

func TestThemeColors(t *testing.T) {
	if ColorPrimaryBrand != "#FF023D" {
		t.Errorf("expected PrimaryBrand #FF023D, got %s", ColorPrimaryBrand)
	}
	if ColorPrimaryHoverActive != "#D60032" {
		t.Errorf("expected PrimaryHoverActive #D60032, got %s", ColorPrimaryHoverActive)
	}
	if ColorAccentSecondary != "#800020" {
		t.Errorf("expected AccentSecondary #800020, got %s", ColorAccentSecondary)
	}
	if ColorDarkBackground != "#120206" {
		t.Errorf("expected DarkBackground #120206, got %s", ColorDarkBackground)
	}
	if ColorDarkBackgroundAlt != "#1A0A0E" {
		t.Errorf("expected DarkBackgroundAlt #1A0A0E, got %s", ColorDarkBackgroundAlt)
	}
}

func TestLinearGradientCSS(t *testing.T) {
	expected := "linear-gradient(135deg, #FF023D 0%, #800020 100%)"
	if grad := LinearGradientCSS(); grad != expected {
		t.Errorf("expected gradient %q, got %q", expected, grad)
	}
}

func TestThemeStyles(t *testing.T) {
	if TitleStyle.GetForeground() == nil {
		t.Error("expected TitleStyle to have foreground set")
	}
	rendered := SummaryStyle.Render("test")
	if rendered == "" {
		t.Error("expected SummaryStyle.Render to return styled text")
	}
}
