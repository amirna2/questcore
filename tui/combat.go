package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nathoo/questcore/engine/state"
)

// healthBar produces an ASCII bar like ████████░░░░ using block characters.
func healthBar(current, max, width int) string {
	if max <= 0 {
		return strings.Repeat("░", width)
	}
	if current < 0 {
		current = 0
	}
	if current > max {
		current = max
	}
	filled := (current * width) / max
	// Ensure at least 1 filled block if current > 0.
	if current > 0 && filled == 0 {
		filled = 1
	}
	empty := width - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

// padRight pads a string to the given display width with spaces.
// Unlike fmt's %-*s, this accounts for multi-byte runes correctly.
func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// renderCombatStatus produces the bordered combat HUD.
func (m Model) renderCombatStatus() string {
	s := m.engine.State
	if !state.InCombat(s) {
		return ""
	}

	enemyID := s.Combat.EnemyID
	enemyHP, _ := state.GetStat(s, m.defs, enemyID, "hp")
	enemyMaxHP, _ := state.GetStat(s, m.defs, enemyID, "max_hp")
	playerHP, _ := state.GetStat(s, m.defs, "player", "hp")
	playerMaxHP, _ := state.GetStat(s, m.defs, "player", "max_hp")

	enemyName := enemyID
	if n, ok := state.GetEntityProp(s, m.defs, enemyID, "name"); ok {
		if ns, ok := n.(string); ok {
			enemyName = ns
		}
	}

	const barWidth = 12
	const nameWidth = 20
	hpEnemy := fmt.Sprintf("%d/%d HP", enemyHP, enemyMaxHP)
	hpPlayer := fmt.Sprintf("%d/%d HP", playerHP, playerMaxHP)
	// Right-align HP values to the same width.
	hpWidth := len(hpEnemy)
	if len(hpPlayer) > hpWidth {
		hpWidth = len(hpPlayer)
	}

	line1 := fmt.Sprintf("%s %s  %*s", padRight(enemyName, nameWidth), healthBar(enemyHP, enemyMaxHP, barWidth), hpWidth, hpEnemy)
	line2 := fmt.Sprintf("%s %s  %*s", padRight("You", nameWidth), healthBar(playerHP, playerMaxHP, barWidth), hpWidth, hpPlayer)
	content := line1 + "\n" + line2

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("208")).
		Foreground(lipgloss.Color("208")).
		Padding(0, 1).
		Render(content)

	box = injectBorderTitle(box, " COMBAT ")

	return box
}

// renderVictory produces a bordered box showing the final combat result after
// the enemy is defeated. Called with the enemy ID captured before Step() cleared
// the combat state.
func (m Model) renderVictory(enemyID string) string {
	s := m.engine.State
	enemyMaxHP, _ := state.GetStat(s, m.defs, enemyID, "max_hp")
	playerHP, _ := state.GetStat(s, m.defs, "player", "hp")
	playerMaxHP, _ := state.GetStat(s, m.defs, "player", "max_hp")

	enemyName := enemyID
	if n, ok := state.GetEntityProp(s, m.defs, enemyID, "name"); ok {
		if ns, ok := n.(string); ok {
			enemyName = ns
		}
	}

	const barWidth = 12
	const nameWidth = 20
	hpEnemy := fmt.Sprintf("0/%d HP", enemyMaxHP)
	hpPlayer := fmt.Sprintf("%d/%d HP", playerHP, playerMaxHP)
	hpWidth := len(hpEnemy)
	if len(hpPlayer) > hpWidth {
		hpWidth = len(hpPlayer)
	}

	line1 := fmt.Sprintf("%s %s  %*s", padRight(enemyName, nameWidth), healthBar(0, enemyMaxHP, barWidth), hpWidth, hpEnemy)
	line2 := fmt.Sprintf("%s %s  %*s", padRight("You", nameWidth), healthBar(playerHP, playerMaxHP, barWidth), hpWidth, hpPlayer)
	content := line1 + "\n" + line2

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("34")).
		Foreground(lipgloss.Color("34")).
		Padding(0, 1).
		Render(content)

	box = injectBorderTitle(box, " VICTORY ")

	return box
}

// renderDefeat produces a bordered box showing the final combat result when the
// player is killed. Shows player at 0 HP and the enemy's remaining HP.
func (m Model) renderDefeat(enemyID string) string {
	s := m.engine.State
	enemyHP, _ := state.GetStat(s, m.defs, enemyID, "hp")
	enemyMaxHP, _ := state.GetStat(s, m.defs, enemyID, "max_hp")
	playerMaxHP, _ := state.GetStat(s, m.defs, "player", "max_hp")

	enemyName := enemyID
	if n, ok := state.GetEntityProp(s, m.defs, enemyID, "name"); ok {
		if ns, ok := n.(string); ok {
			enemyName = ns
		}
	}

	const barWidth = 12
	const nameWidth = 20
	hpEnemy := fmt.Sprintf("%d/%d HP", enemyHP, enemyMaxHP)
	hpPlayer := fmt.Sprintf("0/%d HP", playerMaxHP)
	hpWidth := len(hpEnemy)
	if len(hpPlayer) > hpWidth {
		hpWidth = len(hpPlayer)
	}

	line1 := fmt.Sprintf("%s %s  %*s", padRight(enemyName, nameWidth), healthBar(enemyHP, enemyMaxHP, barWidth), hpWidth, hpEnemy)
	line2 := fmt.Sprintf("%s %s  %*s", padRight("You", nameWidth), healthBar(0, playerMaxHP, barWidth), hpWidth, hpPlayer)
	content := line1 + "\n" + line2

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Foreground(lipgloss.Color("196")).
		Padding(0, 1).
		Render(content)

	box = injectBorderTitle(box, " DEFEAT ")

	return box
}

// renderGameOver produces the bordered game over screen.
func (m Model) renderGameOver(enemyID string) string {
	s := m.engine.State

	enemyName := "unknown"
	if enemyID != "" {
		enemyName = enemyID
		if n, ok := state.GetEntityProp(s, m.defs, enemyID, "name"); ok {
			if ns, ok := n.(string); ok {
				enemyName = ns
			}
		}
	}

	content := fmt.Sprintf("You were slain by the %s.\n\n/load to restore a save\n/quit to exit", enemyName)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Padding(0, 1).
		Render(content)

	box = injectBorderTitle(box, " GAME OVER ")

	return box
}

// injectBorderTitle replaces part of the top border line with a title string.
// Handles ANSI escape codes in the rendered border by working on the visible
// characters only.
func injectBorderTitle(rendered string, title string) string {
	lines := strings.SplitN(rendered, "\n", 2)
	if len(lines) < 2 {
		return rendered
	}

	top := lines[0]
	titleStr := "─" + title + "─"

	// Find the position after the first visible character (corner ╭) by
	// scanning past any leading ANSI escape sequences + the corner char.
	// Then replace the next N visible characters with the title.
	var result strings.Builder
	inserted := false
	visibleIdx := 0
	i := 0
	runes := []rune(top)

	for i < len(runes) {
		// Skip ANSI escape sequences.
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			start := i
			i += 2
			for i < len(runes) && runes[i] != 'm' {
				i++
			}
			if i < len(runes) {
				i++ // skip 'm'
			}
			result.WriteString(string(runes[start:i]))
			continue
		}

		visibleIdx++

		if visibleIdx == 2 && !inserted {
			// We're at the second visible char (first ─ after ╭).
			// Insert the title, skipping the corresponding original chars.
			result.WriteString(titleStr)
			inserted = true
			// Skip len(titleStr) visible characters from the original.
			titleRunes := []rune(titleStr)
			skip := len(titleRunes)
			for skip > 0 && i < len(runes) {
				if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
					start := i
					i += 2
					for i < len(runes) && runes[i] != 'm' {
						i++
					}
					if i < len(runes) {
						i++
					}
					result.WriteString(string(runes[start:i]))
					continue
				}
				skip--
				i++
			}
			continue
		}

		result.WriteRune(runes[i])
		i++
	}

	return result.String() + "\n" + lines[1]
}
