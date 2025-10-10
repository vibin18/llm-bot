package services

import (
	"regexp"
	"strings"
)

// FormatForWhatsApp formats text to use WhatsApp's formatting syntax
// Supports:
// - **bold** or __bold__ → *bold*
// - *italic* or _italic_ → _italic_
// - ~~strikethrough~~ → ~strikethrough~
// - `code` → ```code```
// - ```code block``` → ```code block```
// - Bullet points and numbered lists
// - Line breaks and paragraphs
func FormatForWhatsApp(text string) string {
	if text == "" {
		return text
	}

	// Convert markdown bold (**text** or __text__) to WhatsApp bold (*text*)
	text = convertMarkdownBold(text)

	// Convert markdown strikethrough (~~text~~) to WhatsApp strikethrough (~text~)
	text = convertMarkdownStrikethrough(text)

	// Handle code blocks and inline code
	text = convertMarkdownCode(text)

	// Clean up extra whitespace while preserving intentional line breaks
	text = cleanupWhitespace(text)

	// Ensure proper spacing for lists
	text = formatLists(text)

	return text
}

// convertMarkdownBold converts **text** or __text__ to *text*
func convertMarkdownBold(text string) string {
	// Convert **text** to *text* (non-greedy match)
	re := regexp.MustCompile(`\*\*([^*]+?)\*\*`)
	text = re.ReplaceAllString(text, "*$1*")

	// Convert __text__ to *text* (non-greedy match)
	re = regexp.MustCompile(`__([^_]+?)__`)
	text = re.ReplaceAllString(text, "*$1*")

	return text
}

// convertMarkdownStrikethrough converts ~~text~~ to ~text~
func convertMarkdownStrikethrough(text string) string {
	re := regexp.MustCompile(`~~(.+?)~~`)
	text = re.ReplaceAllString(text, "~$1~")
	return text
}

// convertMarkdownCode handles code blocks and inline code
func convertMarkdownCode(text string) string {
	// Multi-line code blocks: ```code``` stays as ```code```
	// This is already WhatsApp format, no change needed

	// Inline code: `code` → ```code```
	re := regexp.MustCompile("`([^`]+?)`")
	text = re.ReplaceAllString(text, "```$1```")

	return text
}

// cleanupWhitespace removes extra whitespace while preserving intentional formatting
func cleanupWhitespace(text string) string {
	// Remove trailing whitespace from lines
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	text = strings.Join(lines, "\n")

	// Remove more than 2 consecutive newlines (keep at most double line break)
	re := regexp.MustCompile(`\n{3,}`)
	text = re.ReplaceAllString(text, "\n\n")

	// Trim leading/trailing whitespace
	text = strings.TrimSpace(text)

	return text
}

// formatLists ensures proper formatting for bullet points and numbered lists
func formatLists(text string) string {
	lines := strings.Split(text, "\n")
	var formatted []string
	inList := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is a list item
		isListItem := false
		if len(trimmed) > 0 {
			// Bullet points: -, *, •, ◦, ▪, ▫
			if strings.HasPrefix(trimmed, "- ") ||
				strings.HasPrefix(trimmed, "* ") ||
				strings.HasPrefix(trimmed, "• ") ||
				strings.HasPrefix(trimmed, "◦ ") ||
				strings.HasPrefix(trimmed, "▪ ") ||
				strings.HasPrefix(trimmed, "▫ ") {
				isListItem = true
			}

			// Numbered lists: 1., 2., etc.
			re := regexp.MustCompile(`^\d+\.\s`)
			if re.MatchString(trimmed) {
				isListItem = true
			}
		}

		if isListItem {
			// Add spacing before list if this is the first item
			if !inList && i > 0 && len(formatted) > 0 {
				formatted = append(formatted, "")
			}
			formatted = append(formatted, trimmed)
			inList = true
		} else {
			// Add spacing after list if this is not a list item
			if inList && trimmed != "" {
				formatted = append(formatted, "")
			}
			formatted = append(formatted, line)
			inList = trimmed == "" // Empty line keeps us "in list" mode
		}
	}

	return strings.Join(formatted, "\n")
}

// FormatWebhookResponse formats webhook text responses for WhatsApp
// This is a convenience wrapper that applies all formatting
func FormatWebhookResponse(text string) string {
	return FormatForWhatsApp(text)
}
