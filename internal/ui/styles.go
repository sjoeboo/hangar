package ui

import (
	"fmt"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// Theme represents the current color scheme
type Theme string

const (
	ThemeOasisLagoonDark  Theme = "oasis-lagoon-dark"
	ThemeOasisLagoonLight Theme = "oasis-lagoon-light"

	// Backward-compat aliases (used internally for system theme resolution)
	ThemeDark  = ThemeOasisLagoonDark
	ThemeLight = ThemeOasisLagoonLight
)

// colorPalette holds the semantic color slots for a theme.
// Fields use lipgloss.TerminalColor so palettes can mix hex (lipgloss.Color),
// ANSI palette slots (lipgloss.ANSIColor), and transparent (lipgloss.NoColor).
type colorPalette struct {
	Bg, Surface, Border, Text, TextDim  lipgloss.TerminalColor
	Accent, Purple, Cyan, Green, Yellow lipgloss.TerminalColor
	Orange, Red, Comment                lipgloss.TerminalColor

	// InvertFg is the foreground color for inverted/highlighted elements
	// (e.g. dark text on colored badge/pill/selection background).
	// For hex themes this equals Bg. For the terminal theme it is ANSIColor(0)
	// because Bg is NoColor{} (transparent) which can't be used as foreground.
	InvertFg lipgloss.TerminalColor
}

// currentTheme holds the active theme (set at init)
var currentTheme Theme = ThemeDark

// hex is shorthand for lipgloss.Color in palette definitions.
func hex(s string) lipgloss.TerminalColor { return lipgloss.Color(s) }

// ansi is shorthand for lipgloss.ANSIColor in palette definitions.
func ansi(n int) lipgloss.TerminalColor { return lipgloss.ANSIColor(n) }

// ── Oasis Lagoon (original) ─────────────────────────────────────────

var oasisLagoonDark = colorPalette{
	Bg:       hex("#101825"), // bg.core
	InvertFg: hex("#101825"), // same as Bg
	Surface:  hex("#22385C"), // bg.surface
	Border:   hex("#264870"), // mid-navy
	Text:     hex("#D9E6FA"), // fg.core
	TextDim:  hex("#8FB0D0"), // blue-gray dim
	Accent:   hex("#58B8FD"), // lagoon primary blue
	Purple:   hex("#C695FF"), // terminal magenta
	Cyan:     hex("#68C0B6"), // terminal cyan/teal
	Green:    hex("#53D390"), // cactus green
	Yellow:   hex("#F0E68C"), // khaki
	Orange:   hex("#F8B471"), // sunrise orange
	Red:      hex("#FF7979"), // terminal red
	Comment:  hex("#4D88A7"), // fg.comment
}

var oasisLagoonLight = colorPalette{
	Bg:       hex("#EEF4FF"), // lagoon[50] - lightest lagoon tint
	InvertFg: hex("#EEF4FF"), // same as Bg
	Surface:  hex("#D0E8FE"), // lagoon[100] - soft blue surface
	Border:   hex("#B2DCFE"), // lagoon[200] - medium lagoon border
	Text:     hex("#10426d"), // light_terminal.bright_blue - dark navy text
	TextDim:  hex("#1f3f71"), // light_terminal.bright_blue variant
	Accent:   hex("#1670AD"), // lagoon[800] - medium lagoon accent
	Purple:   hex("#46259f"), // light_terminal.magenta
	Cyan:     hex("#064658"), // light_terminal.cyan
	Green:    hex("#1b491d"), // light_terminal.green
	Yellow:   hex("#6b2e00"), // light_terminal.yellow
	Orange:   hex("#533c00"), // light_terminal.bright_yellow
	Red:      hex("#663021"), // light_terminal.red
	Comment:  hex("#0D4266"), // lagoon[900] - deep lagoon comment
}

// ── Terminal Adaptive ───────────────────────────────────────────────
// Uses ANSI palette slots 0-15 so colors follow the terminal's theme.
// Change your terminal theme (Tokyo Night, Catppuccin, Gruvbox, etc.)
// and Hangar adapts automatically.

const ThemeTerminal Theme = "terminal"

var terminalAdaptive = colorPalette{
	Bg:       lipgloss.NoColor{}, // transparent — terminal bg shows through
	InvertFg: ansi(0),            // ANSI black — dark text on colored backgrounds
	Surface:  lipgloss.NoColor{}, // transparent — no distinct elevation
	Border:   ansi(8),            // bright black — universally a muted gray
	Text:    ansi(15), // bright white — primary foreground
	TextDim: ansi(7),  // white (normal) — mid-gray in dark themes
	Accent:  ansi(12), // bright blue — primary accent
	Purple:  ansi(13), // bright magenta
	Cyan:    ansi(14), // bright cyan
	Green:   ansi(10), // bright green
	Yellow:  ansi(11), // bright yellow (often orange-ish in themes)
	Orange:  ansi(3),  // yellow (normal) — closest to orange in ANSI
	Red:     ansi(9),  // bright red
	Comment: ansi(8),  // bright black — muted/comment text
}

// palettes maps theme names to their color palette.
var palettes = map[Theme]colorPalette{
	ThemeOasisLagoonDark:  oasisLagoonDark,
	ThemeOasisLagoonLight: oasisLagoonLight,
	ThemeTerminal:         terminalAdaptive,
}

// Backward-compat: old names that users may have in config.toml
var themeAliases = map[string]Theme{
	"dark":  ThemeOasisLagoonDark,
	"light": ThemeOasisLagoonLight,
}

// Active color variables (set by InitTheme)
var (
	ColorBg       lipgloss.TerminalColor
	ColorInvertFg lipgloss.TerminalColor // dark text for inverted/highlighted elements
	ColorSurface  lipgloss.TerminalColor
	ColorBorder  lipgloss.TerminalColor
	ColorText    lipgloss.TerminalColor
	ColorTextDim lipgloss.TerminalColor
	ColorAccent  lipgloss.TerminalColor
	ColorPurple  lipgloss.TerminalColor
	ColorCyan    lipgloss.TerminalColor
	ColorGreen   lipgloss.TerminalColor
	ColorYellow  lipgloss.TerminalColor
	ColorOrange  lipgloss.TerminalColor
	ColorRed     lipgloss.TerminalColor
	ColorComment lipgloss.TerminalColor
)

// themeMu protects global color/style variables during live theme switches.
// Write lock held by InitTheme; read lock held by GetToolStyle (map access).
var themeMu sync.RWMutex

// ResolveThemeName normalizes a theme string to a canonical Theme value.
// Handles backward-compat aliases ("dark" → "oasis-lagoon-dark", "light" → "oasis-lagoon-light").
// Returns ThemeTerminal for unrecognized values.
func ResolveThemeName(name string) Theme {
	// Check aliases first (backward compat)
	if t, ok := themeAliases[name]; ok {
		return t
	}
	// Check canonical names
	t := Theme(name)
	if _, ok := palettes[t]; ok {
		return t
	}
	return ThemeTerminal
}

// InitTheme sets the active color palette based on theme name
// Must be called before any UI rendering
func InitTheme(theme string) {
	themeMu.Lock()
	defer themeMu.Unlock()
	currentTheme = ResolveThemeName(theme)
	p := palettes[currentTheme]
	ColorBg = p.Bg
	ColorInvertFg = p.InvertFg
	ColorSurface = p.Surface
	ColorBorder = p.Border
	ColorText = p.Text
	ColorTextDim = p.TextDim
	ColorAccent = p.Accent
	ColorPurple = p.Purple
	ColorCyan = p.Cyan
	ColorGreen = p.Green
	ColorYellow = p.Yellow
	ColorOrange = p.Orange
	ColorRed = p.Red
	ColorComment = p.Comment
	// Reinitialize styles with new colors
	initStyles()
}

// GetCurrentTheme returns the active theme
func GetCurrentTheme() Theme {
	return currentTheme
}

func init() {
	// Default to terminal-adaptive theme at package init
	InitTheme(string(ThemeTerminal))
}

// Base Styles
var (
	BaseStyle      lipgloss.Style
	TitleStyle     lipgloss.Style
	PanelStyle     lipgloss.Style
	HighlightStyle lipgloss.Style
	DimStyle       lipgloss.Style
	ErrorStyle     lipgloss.Style
	SuccessStyle   lipgloss.Style
	WarningStyle   lipgloss.Style
	InfoStyle      lipgloss.Style
)

// Status Indicator Styles
var (
	RunningStyle        lipgloss.Style
	WaitingStyle        lipgloss.Style
	IdleStyle           lipgloss.Style
	ErrorIndicatorStyle lipgloss.Style
)

// Menu Bar Styles
var (
	MenuBarStyle       lipgloss.Style
	MenuKeyStyle       lipgloss.Style
	MenuDescStyle      lipgloss.Style
	MenuSeparatorStyle lipgloss.Style
)

// Search Styles
var (
	SearchBoxStyle    lipgloss.Style
	SearchPromptStyle lipgloss.Style
	SearchMatchStyle  lipgloss.Style
)

// Dialog Styles
var (
	DialogBoxStyle          lipgloss.Style
	DialogTitleStyle        lipgloss.Style
	DialogButtonStyle       lipgloss.Style
	DialogButtonActiveStyle lipgloss.Style
)

// Preview Pane Styles
var (
	PreviewPanelStyle   lipgloss.Style
	PreviewTitleStyle   lipgloss.Style
	PreviewHeaderStyle  lipgloss.Style
	PreviewContentStyle lipgloss.Style
	PreviewMetaStyle    lipgloss.Style
)

// Tool Icons
const (
	IconClaude   = "🤖"
	IconGemini   = "✨"
	IconOpenCode = "🌐"
	IconCodex    = "💻"
	IconShell    = "🐚"
)

// MaxNameLength is the maximum allowed length for session and group names.
// Used by dialog CharLimits and Validate() methods to ensure consistency.
const MaxNameLength = 50

// List Item Styles (used by legacy list.go component in tests)
var (
	ListItemStyle       lipgloss.Style
	ListItemActiveStyle lipgloss.Style
)

// Tag Styles
var (
	TagStyle       lipgloss.Style
	TagActiveStyle lipgloss.Style
	TagErrorStyle  lipgloss.Style
)

// Timestamp Style
var TimestampStyle lipgloss.Style

// Folder Styles
var (
	FolderStyle          lipgloss.Style
	FolderCollapsedStyle lipgloss.Style
)

// Session Item Styles
var (
	SessionItemStyle         lipgloss.Style
	SessionItemSelectedStyle lipgloss.Style
)

// Session List Rendering Styles (PERFORMANCE: cached at package level)
// These styles are used by renderSessionItem() and renderGroupItem() to avoid
// repeated allocations on every View() call
var (
	// Tree connector styles
	TreeConnectorStyle    lipgloss.Style
	TreeConnectorSelStyle lipgloss.Style

	// Session status indicator styles
	SessionStatusRunning  lipgloss.Style
	SessionStatusWaiting  lipgloss.Style
	SessionStatusIdle     lipgloss.Style
	SessionStatusError    lipgloss.Style
	SessionStatusSelStyle lipgloss.Style

	// PR badge styles — colored by state, distinct from session status icons
	PRBadgeOpen   lipgloss.Style
	PRBadgeDraft  lipgloss.Style
	PRBadgeMerged lipgloss.Style
	PRBadgeClosed lipgloss.Style

	// Tower session badge
	TowerBadgeStyle lipgloss.Style

	// Session title styles by state
	SessionTitleDefault  lipgloss.Style
	SessionTitleActive   lipgloss.Style
	SessionTitleError    lipgloss.Style
	SessionTitleSelStyle lipgloss.Style

	// Selection indicator
	SessionSelectionPrefix   lipgloss.Style
	SessionCheckboxUnchecked lipgloss.Style

	// Group item styles
	GroupExpandStyle   lipgloss.Style
	GroupNameStyle     lipgloss.Style
	GroupCountStyle    lipgloss.Style
	GroupHotkeyStyle   lipgloss.Style
	GroupStatusRunning lipgloss.Style
	GroupStatusWaiting lipgloss.Style

	// Group selected styles
	GroupNameSelStyle   lipgloss.Style
	GroupCountSelStyle  lipgloss.Style
	GroupExpandSelStyle lipgloss.Style
)

// Preview Renderer Styles (PERFORMANCE: cached at package level)
// These styles are used by preview_renderer.go, list_renderer.go, and layout_renderer.go
// to avoid repeated allocations on every View() call.
var (
	// Section divider styles
	styleSectionDividerLine  lipgloss.Style // Foreground(ColorBorder)
	styleSectionDividerLabel lipgloss.Style // Foreground(ColorText).Bold(true)

	// General preview text styles
	stylePreviewLabel    lipgloss.Style // Foreground(ColorText) — label/value/info
	stylePreviewLabelDim lipgloss.Style // Foreground(ColorTextDim) — dimmed label
	stylePreviewDim      lipgloss.Style // Foreground(ColorText).Italic(true) — hints, elapsed, "more"
	stylePreviewKey      lipgloss.Style // Foreground(ColorAccent).Bold(true) — keyboard key hints
	stylePreviewAccent   lipgloss.Style // Foreground(ColorAccent) — dots, accented values
	stylePreviewBoldName lipgloss.Style // Bold(true).Foreground(ColorAccent) — session name header

	// Connection status styles
	stylePreviewConnected   lipgloss.Style // Foreground(ColorGreen).Bold(true) — "● Connected"
	stylePreviewDetecting   lipgloss.Style // Foreground(ColorYellow) — "◐ Detecting..." / pending MCPs
	stylePreviewNotFound    lipgloss.Style // Foreground(ColorText) — "○ Not connected" / "○ No session found"
	stylePreviewComment     lipgloss.Style // Foreground(ColorComment) — URLs, "checking..."
	stylePreviewPRNum       lipgloss.Style // Foreground(ColorAccent).Bold(true).Underline(true) — PR number
	stylePreviewTimeElapsed lipgloss.Style // Foreground(ColorYellow).Italic(true) — "Loading... Xs"

	// PR check badge styles
	stylePreviewChecksFailed  lipgloss.Style // Foreground(ColorRed)
	stylePreviewChecksPending lipgloss.Style // Foreground(ColorYellow)
	stylePreviewChecksPassed  lipgloss.Style // Foreground(ColorGreen)

	// Loading state animation styles
	styleSpinnerLaunch lipgloss.Style // Foreground(ColorAccent).Bold(true)
	styleTitleLaunch   lipgloss.Style // Foreground(ColorPurple).Bold(true)
	styleSpinnerMCP    lipgloss.Style // Foreground(ColorCyan).Bold(true)
	styleTitleMCP      lipgloss.Style // Foreground(ColorCyan).Bold(true)
	styleSpinnerFork   lipgloss.Style // Foreground(ColorPurple).Bold(true) — same as titleLaunch
	styleDotsMCP       lipgloss.Style // Foreground(ColorCyan)
	styleDotsFork      lipgloss.Style // Foreground(ColorPurple)

	// Error state styles
	stylePreviewWarn lipgloss.Style // Foreground(ColorYellow) — "⚠" warning

	// Session info card styles
	styleInfoCardHeader lipgloss.Style // Bold(true).Foreground(ColorAccent)

	// Badge styles
	styleToolBadge  lipgloss.Style // Foreground(ColorInvertFg).Background(ColorPurple).Padding(0,1)
	styleGroupBadge lipgloss.Style // Foreground(ColorInvertFg).Background(ColorCyan).Padding(0,1)

	// List renderer styles
	styleListEmptyBorder lipgloss.Style // Border(RoundedBorder).BorderForeground(ColorBorder)
	styleYoloBadge       lipgloss.Style // Foreground(ColorYellow).Bold(true)

	// Layout renderer styles
	styleLayoutSeparator lipgloss.Style // Foreground(ColorBorder)

	// Group preview styles
	styleGroupPreviewHeader lipgloss.Style // Foreground(ColorCyan).Bold(true)
	styleGroupPreviewCount  lipgloss.Style // Foreground(ColorText).Bold(true)
	styleGroupRepoBranch    lipgloss.Style // Foreground(ColorCyan).Bold(true)

	// Group preview session status summary
	styleGroupStatusRunning lipgloss.Style // Foreground(ColorGreen)
	styleGroupStatusWaiting lipgloss.Style // Foreground(ColorYellow)
	styleGroupStatusIdle    lipgloss.Style // Foreground(ColorText)
	styleGroupStatusError   lipgloss.Style // Foreground(ColorRed)

	// Group preview session list
	styleGroupSessionTool lipgloss.Style // Foreground(ColorPurple).Faint(true)

	// Group preview hints
	styleGroupHint lipgloss.Style // Foreground(ColorComment).Italic(true)
)

// ToolStyleCache provides pre-allocated styles for each tool type
// Avoids repeated lipgloss.NewStyle() calls in renderSessionItem()
var ToolStyleCache map[string]lipgloss.Style

// DefaultToolStyle is used when tool is not in cache
var DefaultToolStyle lipgloss.Style

// Menu Styles
var MenuStyle lipgloss.Style

// Additional Styles
var (
	SubtitleStyle lipgloss.Style
	ColorError    lipgloss.TerminalColor
	ColorSuccess  lipgloss.TerminalColor
	ColorWarning  lipgloss.TerminalColor
	ColorPrimary  lipgloss.TerminalColor
)

// Pre-compiled styles for RenderLogoCompact (called every render frame).
var (
	logoCompactBracketBase = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	logoCompactBorderBase  = lipgloss.NewStyle().Foreground(ColorBorder)
	logoCompactSpaceBase   = lipgloss.NewStyle()
	logoCompactIndicBase   = lipgloss.NewStyle().Bold(true)
)

// LogoBorderStyle for the grid lines
var LogoBorderStyle lipgloss.Style

// LogoFrames kept for backward compatibility (empty state default)
var LogoFrames = [][]string{
	{"●", "◐", "○"},
}

// initStyles initializes all style variables with current theme colors
// Called by InitTheme after color variables are set
func initStyles() {
	// Base Styles
	BaseStyle = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorBg)

	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent).
		Background(ColorSurface).
		Padding(0, 1)

	PanelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	HighlightStyle = lipgloss.NewStyle().
		Foreground(ColorInvertFg).
		Background(ColorAccent).
		Bold(true)

	DimStyle = lipgloss.NewStyle().
		Foreground(ColorComment)

	ErrorStyle = lipgloss.NewStyle().
		Foreground(ColorRed).
		Bold(true)

	SuccessStyle = lipgloss.NewStyle().
		Foreground(ColorGreen).
		Bold(true)

	WarningStyle = lipgloss.NewStyle().
		Foreground(ColorYellow).
		Bold(true)

	InfoStyle = lipgloss.NewStyle().
		Foreground(ColorCyan)

	// Status Indicator Styles
	RunningStyle = lipgloss.NewStyle().
		Foreground(ColorGreen).
		Bold(true)

	WaitingStyle = lipgloss.NewStyle().
		Foreground(ColorYellow).
		Bold(true)

	IdleStyle = lipgloss.NewStyle().
		Foreground(ColorComment)

	ErrorIndicatorStyle = lipgloss.NewStyle().
		Foreground(ColorRed).
		Bold(true)

	// Menu Bar Styles
	MenuBarStyle = lipgloss.NewStyle().
		Background(ColorSurface).
		Foreground(ColorText).
		Padding(0, 1)

	MenuKeyStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	MenuDescStyle = lipgloss.NewStyle().
		Foreground(ColorText)

	MenuSeparatorStyle = lipgloss.NewStyle().
		Foreground(ColorBorder)

	// Search Styles
	SearchBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorAccent).
		Padding(0, 1).
		Foreground(ColorText)

	SearchPromptStyle = lipgloss.NewStyle().
		Foreground(ColorPurple).
		Bold(true)

	SearchMatchStyle = lipgloss.NewStyle().
		Background(ColorYellow).
		Foreground(ColorInvertFg).
		Bold(true)

	// Dialog Styles
	DialogBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPurple).
		Padding(1, 2).
		Background(ColorSurface)

	DialogTitleStyle = lipgloss.NewStyle().
		Foreground(ColorPurple).
		Bold(true).
		Align(lipgloss.Center)

	DialogButtonStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Background(ColorBorder).
		Padding(0, 2).
		MarginRight(1)

	DialogButtonActiveStyle = lipgloss.NewStyle().
		Foreground(ColorInvertFg).
		Background(ColorAccent).
		Padding(0, 2).
		MarginRight(1).
		Bold(true)

	// Preview Pane Styles
	PreviewPanelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1)

	PreviewTitleStyle = lipgloss.NewStyle().
		Foreground(ColorCyan).
		Bold(true).
		Underline(true)

	PreviewHeaderStyle = lipgloss.NewStyle().
		Foreground(ColorPurple).
		Bold(true)

	PreviewContentStyle = lipgloss.NewStyle().
		Foreground(ColorText)

	PreviewMetaStyle = lipgloss.NewStyle().
		Foreground(ColorComment).
		Italic(true)

	// List Item Styles
	ListItemStyle = lipgloss.NewStyle().
		Foreground(ColorText).
		PaddingLeft(2)

	ListItemActiveStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true).
		PaddingLeft(2)

	// Tag Styles
	TagStyle = lipgloss.NewStyle().
		Foreground(ColorInvertFg).
		Background(ColorPurple).
		Padding(0, 1).
		MarginRight(1)

	TagActiveStyle = lipgloss.NewStyle().
		Foreground(ColorInvertFg).
		Background(ColorGreen).
		Padding(0, 1).
		MarginRight(1)

	TagErrorStyle = lipgloss.NewStyle().
		Foreground(ColorInvertFg).
		Background(ColorRed).
		Padding(0, 1).
		MarginRight(1)

	// Timestamp Style
	TimestampStyle = lipgloss.NewStyle().
		Foreground(ColorComment).
		Italic(true)

	// Folder Styles
	FolderStyle = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	FolderCollapsedStyle = lipgloss.NewStyle().
		Foreground(ColorComment)

	// Session Item Styles
	SessionItemStyle = lipgloss.NewStyle().
		Foreground(ColorText).
		PaddingLeft(2)

	SessionItemSelectedStyle = lipgloss.NewStyle().
		Foreground(ColorInvertFg).
		Background(ColorAccent).
		Bold(true).
		PaddingLeft(0)

	// Tree connector styles
	TreeConnectorStyle = lipgloss.NewStyle().Foreground(ColorText)
	TreeConnectorSelStyle = lipgloss.NewStyle().Foreground(ColorInvertFg).Background(ColorAccent)

	// Session status indicator styles
	SessionStatusRunning = lipgloss.NewStyle().Foreground(ColorGreen)
	SessionStatusWaiting = lipgloss.NewStyle().Foreground(ColorYellow)
	SessionStatusIdle = lipgloss.NewStyle().Foreground(ColorTextDim)
	SessionStatusError = lipgloss.NewStyle().Foreground(ColorRed)
	SessionStatusSelStyle = lipgloss.NewStyle().Foreground(ColorInvertFg).Background(ColorAccent)

	PRBadgeOpen = lipgloss.NewStyle().Foreground(ColorGreen)
	PRBadgeDraft = lipgloss.NewStyle().Foreground(ColorComment)
	PRBadgeMerged = lipgloss.NewStyle().Foreground(ColorPurple)
	PRBadgeClosed = lipgloss.NewStyle().Foreground(ColorRed)

	// Session title styles by state
	SessionTitleDefault = lipgloss.NewStyle().Foreground(ColorText)
	SessionTitleActive = lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	SessionTitleError = lipgloss.NewStyle().Foreground(ColorText).Underline(true)
	SessionTitleSelStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorInvertFg).Background(ColorAccent)

	// Selection indicator
	SessionSelectionPrefix = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	SessionCheckboxUnchecked = lipgloss.NewStyle().Foreground(ColorTextDim)

	// Group item styles
	GroupExpandStyle = lipgloss.NewStyle().Foreground(ColorText)
	GroupNameStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	GroupCountStyle = lipgloss.NewStyle().Foreground(ColorText)
	GroupHotkeyStyle = lipgloss.NewStyle().Foreground(ColorComment)
	GroupStatusRunning = lipgloss.NewStyle().Foreground(ColorGreen)
	GroupStatusWaiting = lipgloss.NewStyle().Foreground(ColorYellow)

	// Group selected styles
	GroupNameSelStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorInvertFg).Background(ColorAccent)
	GroupCountSelStyle = lipgloss.NewStyle().Foreground(ColorInvertFg).Background(ColorAccent)
	GroupExpandSelStyle = lipgloss.NewStyle().Foreground(ColorInvertFg).Background(ColorAccent)

	// ToolStyleCache - reinitialize with current theme colors
	ToolStyleCache = map[string]lipgloss.Style{
		"claude":   lipgloss.NewStyle().Foreground(ColorOrange),
		"gemini":   lipgloss.NewStyle().Foreground(ColorPurple),
		"codex":    lipgloss.NewStyle().Foreground(ColorCyan),
		"aider":    lipgloss.NewStyle().Foreground(ColorRed),
		"cursor":   lipgloss.NewStyle().Foreground(ColorAccent),
		"shell":    lipgloss.NewStyle().Foreground(ColorText),
		"opencode": lipgloss.NewStyle().Foreground(ColorText),
	}

	// DefaultToolStyle
	DefaultToolStyle = lipgloss.NewStyle().Foreground(ColorText)

	// Menu Styles
	MenuStyle = lipgloss.NewStyle().
		Background(ColorSurface).
		Foreground(ColorText).
		Padding(0, 1)

	// Additional Styles
	SubtitleStyle = lipgloss.NewStyle().
		Foreground(ColorComment).
		Italic(true)

	ColorError = ColorRed
	ColorSuccess = ColorGreen
	ColorWarning = ColorYellow
	ColorPrimary = ColorAccent

	// LogoBorderStyle
	LogoBorderStyle = lipgloss.NewStyle().Foreground(ColorBorder)

	// Section divider styles
	styleSectionDividerLine = lipgloss.NewStyle().Foreground(ColorBorder)
	styleSectionDividerLabel = lipgloss.NewStyle().Foreground(ColorText).Bold(true)

	// General preview text styles
	stylePreviewLabel = lipgloss.NewStyle().Foreground(ColorText)
	stylePreviewLabelDim = lipgloss.NewStyle().Foreground(ColorTextDim)
	stylePreviewDim = lipgloss.NewStyle().Foreground(ColorText).Italic(true)
	stylePreviewKey = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	stylePreviewAccent = lipgloss.NewStyle().Foreground(ColorAccent)
	stylePreviewBoldName = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)

	// Connection status styles
	stylePreviewConnected = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	stylePreviewDetecting = lipgloss.NewStyle().Foreground(ColorYellow)
	stylePreviewNotFound = lipgloss.NewStyle().Foreground(ColorText)
	stylePreviewComment = lipgloss.NewStyle().Foreground(ColorComment)
	stylePreviewPRNum = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Underline(true)
	stylePreviewTimeElapsed = lipgloss.NewStyle().Foreground(ColorYellow).Italic(true)

	// PR check badge styles
	stylePreviewChecksFailed = lipgloss.NewStyle().Foreground(ColorRed)
	stylePreviewChecksPending = lipgloss.NewStyle().Foreground(ColorYellow)
	stylePreviewChecksPassed = lipgloss.NewStyle().Foreground(ColorGreen)

	// Loading state animation styles
	styleSpinnerLaunch = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	styleTitleLaunch = lipgloss.NewStyle().Foreground(ColorPurple).Bold(true)
	styleSpinnerMCP = lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	styleTitleMCP = lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	styleSpinnerFork = lipgloss.NewStyle().Foreground(ColorPurple).Bold(true)
	styleDotsMCP = lipgloss.NewStyle().Foreground(ColorCyan)
	styleDotsFork = lipgloss.NewStyle().Foreground(ColorPurple)

	// Error state styles
	stylePreviewWarn = lipgloss.NewStyle().Foreground(ColorYellow)

	// Session info card styles
	styleInfoCardHeader = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)

	// Badge styles
	styleToolBadge = lipgloss.NewStyle().
		Foreground(ColorInvertFg).
		Background(ColorPurple).
		Padding(0, 1)
	styleGroupBadge = lipgloss.NewStyle().
		Foreground(ColorInvertFg).
		Background(ColorCyan).
		Padding(0, 1)

	// Tower session badge style
	TowerBadgeStyle = lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)

	// List renderer styles
	styleListEmptyBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder)
	styleYoloBadge = lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)

	// Layout renderer styles
	styleLayoutSeparator = lipgloss.NewStyle().Foreground(ColorBorder)

	// Group preview styles
	styleGroupPreviewHeader = lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)
	styleGroupPreviewCount = lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	styleGroupRepoBranch = lipgloss.NewStyle().Foreground(ColorCyan).Bold(true)

	// Group preview session status summary
	styleGroupStatusRunning = lipgloss.NewStyle().Foreground(ColorGreen)
	styleGroupStatusWaiting = lipgloss.NewStyle().Foreground(ColorYellow)
	styleGroupStatusIdle = lipgloss.NewStyle().Foreground(ColorText)
	styleGroupStatusError = lipgloss.NewStyle().Foreground(ColorRed)

	// Group preview session list and hints
	styleGroupSessionTool = lipgloss.NewStyle().Foreground(ColorPurple).Faint(true)
	styleGroupHint = lipgloss.NewStyle().Foreground(ColorComment).Italic(true)

	// PR detail overlay styles (must be here — pr_detail.go vars are package-level
	// and would capture empty Color* values before InitTheme runs).
	prDetailHeaderStyle      = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	prDetailActiveTabStyle   = lipgloss.NewStyle().Foreground(ColorInvertFg).Background(ColorAccent).Bold(true).Padding(0, 1)
	prDetailInactiveTabStyle = lipgloss.NewStyle().Foreground(ColorComment).Padding(0, 1)

	// Nav tab styles (Sessions | PRs | Todos tab row)
	navTabActiveStyle   = lipgloss.NewStyle().Foreground(ColorInvertFg).Background(ColorAccent).Bold(true).Padding(0, 1)
	navTabInactiveStyle = lipgloss.NewStyle().Foreground(ColorComment).Background(ColorSurface).Padding(0, 1)

	// Filter pill styles (Running / Waiting / Idle / Error pills)
	filterPillAllActiveStyle     = lipgloss.NewStyle().Foreground(ColorInvertFg).Background(ColorAccent).Bold(true).Padding(0, 1)
	filterPillRunningActiveStyle = lipgloss.NewStyle().Foreground(ColorInvertFg).Background(ColorGreen).Bold(true).Padding(0, 1)
	filterPillWaitingActiveStyle = lipgloss.NewStyle().Foreground(ColorInvertFg).Background(ColorYellow).Bold(true).Padding(0, 1)
	filterPillIdleActiveStyle    = lipgloss.NewStyle().Foreground(ColorInvertFg).Background(ColorTextDim).Bold(true).Padding(0, 1)
	filterPillErrorActiveStyle   = lipgloss.NewStyle().Foreground(ColorInvertFg).Background(ColorRed).Bold(true).Padding(0, 1)
	filterPillInactiveStyle      = lipgloss.NewStyle().Foreground(ColorComment).Padding(0, 1)
	filterPillDimStyle           = lipgloss.NewStyle().Foreground(ColorText).Faint(true).Padding(0, 1)

	prDetailSeparatorStyle   = lipgloss.NewStyle().Foreground(ColorBorder)
	prDetailDimStyle         = lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true)
	prDetailErrStyle         = lipgloss.NewStyle().Foreground(ColorRed)
	prDetailBgStyle          = lipgloss.NewStyle().Background(ColorBg)
	prDetailLabelStyle       = lipgloss.NewStyle().Foreground(ColorComment)
	prDetailValueStyle       = lipgloss.NewStyle().Foreground(ColorText)
	prDetailSectionHdrStyle  = lipgloss.NewStyle().Foreground(ColorComment).Bold(true)
	prDetailAuthorStyle      = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	prDetailTimeStyle        = lipgloss.NewStyle().Foreground(ColorComment)
	prDetailPathYellowStyle  = lipgloss.NewStyle().Foreground(ColorYellow)
	prDetailFocusPadStyle    = lipgloss.NewStyle().Background(lipgloss.Color("#1e2d3a"))
	prDetailGreenStyle       = lipgloss.NewStyle().Foreground(ColorGreen)
	prDetailRedStyle         = lipgloss.NewStyle().Foreground(ColorRed)
	prDetailPurpleStyle      = lipgloss.NewStyle().Foreground(ColorPurple)
	prDetailYellowStyle      = lipgloss.NewStyle().Foreground(ColorYellow)
	prDetailCommentStyle     = lipgloss.NewStyle().Foreground(ColorComment)

	// DiffView overlay + shared diff line styles
	diffViewHeaderStyle    = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	diffViewSeparatorStyle = lipgloss.NewStyle().Foreground(ColorBorder)
	diffViewFooterStyle    = lipgloss.NewStyle().Foreground(ColorTextDim).Italic(true)
	diffHunkHeaderStyle    = lipgloss.NewStyle().Foreground(ColorComment)
	diffLineAddStyle       = lipgloss.NewStyle().Foreground(ColorGreen)
	diffLineDelStyle       = lipgloss.NewStyle().Foreground(ColorRed)
	diffLineContextStyle   = lipgloss.NewStyle().Foreground(ColorTextDim)
	diffLineNoNewlineStyle = lipgloss.NewStyle().Foreground(ColorComment).Italic(true)
}

// Helper Functions

// MenuKey creates a formatted menu item with key and description
func MenuKey(key, description string) string {
	return fmt.Sprintf("%s %s %s",
		MenuKeyStyle.Render(key),
		MenuSeparatorStyle.Render("•"),
		MenuDescStyle.Render(description),
	)
}

// StatusIndicator returns a styled status indicator.
// Read-locked to protect against concurrent style access during live theme switches.
// Standard symbols: ● running, ◐ waiting, ○ idle, ✕ error, ⟳ starting
func StatusIndicator(status string) string {
	themeMu.RLock()
	defer themeMu.RUnlock()
	switch status {
	case "running":
		return RunningStyle.Render("●")
	case "waiting":
		return WaitingStyle.Render("◐")
	case "idle":
		return IdleStyle.Render("○")
	case "error":
		return ErrorIndicatorStyle.Render("✕")
	case "starting":
		return WaitingStyle.Render("⟳") // Use yellow color, spinning arrow symbol
	default:
		return IdleStyle.Render("○")
	}
}

// ToolIcon returns the icon for a given tool
// Checks user config for custom tools first, then falls back to built-ins
func ToolIcon(tool string) string {
	// Use session.GetToolIcon which handles custom + built-in
	// Import would be circular, so we duplicate the logic here
	// Custom icons are handled by the session layer's GetToolDef
	switch tool {
	case "claude":
		return IconClaude
	case "gemini":
		return IconGemini
	case "opencode":
		return IconOpenCode
	case "codex":
		return IconCodex
	case "cursor":
		return "📝"
	case "shell":
		return IconShell
	default:
		return IconShell
	}
}

// ToolColor returns the brand color for a given tool
// Claude=orange (Anthropic), Gemini=purple (Google AI), Codex=cyan, Aider=red
func ToolColor(tool string) lipgloss.TerminalColor {
	switch tool {
	case "claude":
		return ColorOrange // Anthropic's orange
	case "gemini":
		return ColorPurple // Google AI purple
	case "codex":
		return ColorCyan // Light blue for OpenAI
	case "aider":
		return ColorRed // Red for Aider
	case "cursor":
		return ColorAccent // Blue for Cursor
	default:
		return ColorTextDim // Default gray
	}
}

// GetToolStyle returns cached style for tool or default.
// Read-locked to protect against concurrent map access during live theme switches.
func GetToolStyle(tool string) lipgloss.Style {
	themeMu.RLock()
	defer themeMu.RUnlock()
	if style, ok := ToolStyleCache[tool]; ok {
		return style
	}
	return DefaultToolStyle
}

// RenderLogoIndicator renders a single indicator with appropriate color
func RenderLogoIndicator(indicator string) string {
	var color lipgloss.TerminalColor
	switch indicator {
	case "●":
		color = ColorGreen // Running
	case "◐":
		color = ColorYellow // Waiting
	case "○":
		color = ColorTextDim // Idle
	default:
		color = ColorTextDim
	}
	return lipgloss.NewStyle().Foreground(color).Bold(true).Render(indicator)
}

// getLogoIndicators returns 3 indicators based on actual session status counts
// Priority: Running > Waiting > Idle
// Shows up to 3 indicators reflecting the real state
func getLogoIndicators(running, waiting, idle int) []string {
	indicators := make([]string, 0, 3)

	// Add running indicators (green ●)
	for i := 0; i < running && len(indicators) < 3; i++ {
		indicators = append(indicators, "●")
	}

	// Add waiting indicators (yellow ◐)
	for i := 0; i < waiting && len(indicators) < 3; i++ {
		indicators = append(indicators, "◐")
	}

	// Fill remaining with idle (gray ○)
	for len(indicators) < 3 {
		indicators = append(indicators, "○")
	}

	return indicators
}

// RenderLogoCompact renders the compact inline logo for the header
// Shows REAL status: running=●, waiting=◐, idle=○
// Format: ⟨ ● │ ◐ │ ○ ⟩  (using angle brackets for modern look)
// bg must match the row's background color so that each segment
// explicitly declares it — preventing lipgloss \x1b[0m resets from
// exposing the terminal default background between segments.
func RenderLogoCompact(running, waiting, idle int, bg lipgloss.TerminalColor) string {
	indicators := getLogoIndicators(running, waiting, idle)
	bracketStyle := logoCompactBracketBase.Background(bg)
	borderStyle := logoCompactBorderBase.Background(bg)
	sp := logoCompactSpaceBase.Background(bg).Render(" ")
	indicator := func(ind string) string {
		var color lipgloss.TerminalColor
		switch ind {
		case "●":
			color = ColorGreen
		case "◐":
			color = ColorYellow
		default:
			color = ColorTextDim
		}
		return logoCompactIndicBase.Foreground(color).Background(bg).Render(ind)
	}
	return bracketStyle.Render("⟨") +
		sp + indicator(indicators[0]) +
		borderStyle.Render(" │ ") + indicator(indicators[1]) +
		borderStyle.Render(" │ ") + indicator(indicators[2]) +
		sp + bracketStyle.Render("⟩")
}

// RenderLogoLarge renders the large logo for empty state
// Shows REAL status: running=●, waiting=◐, idle=○
// Format:
//
//	┌──┬──┬──┐
//	│● │◐ │○ │
//	└──┴──┴──┘
func RenderLogoLarge(running, waiting, idle int) string {
	indicators := getLogoIndicators(running, waiting, idle)
	top := LogoBorderStyle.Render("┌──┬──┬──┐")
	mid := LogoBorderStyle.Render("│") + RenderLogoIndicator(indicators[0]) + LogoBorderStyle.Render(" │") +
		RenderLogoIndicator(indicators[1]) + LogoBorderStyle.Render(" │") +
		RenderLogoIndicator(indicators[2]) + LogoBorderStyle.Render(" │")
	bot := LogoBorderStyle.Render("└──┴──┴──┘")
	return top + "\n" + mid + "\n" + bot
}
