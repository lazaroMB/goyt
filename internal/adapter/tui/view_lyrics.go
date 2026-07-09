package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type lyricsLoadedMsg struct {
	plainLyrics  string
	syncedLyrics []SyncedLine
	err          error
	trackID      string
}

type lyricsPreloadedMsg struct {
	plainLyrics  string
	syncedLyrics []SyncedLine
	err          error
	trackID      string
}

func stripAccents(s string) string {
	r := strings.NewReplacer(
		"à", "a", "á", "a", "â", "a", "ã", "a", "ä", "a", "å", "a", "æ", "ae",
		"ç", "c",
		"è", "e", "é", "e", "ê", "e", "ë", "e",
		"ì", "i", "í", "i", "î", "i", "ï", "i",
		"ñ", "n",
		"ò", "o", "ó", "o", "ô", "o", "õ", "o", "ö", "o", "ø", "o",
		"ù", "u", "ú", "u", "û", "u", "ü", "u",
		"ý", "y", "ÿ", "y",
		"š", "s", "ž", "z",
		"ß", "ss",
		"À", "a", "Á", "a", "Â", "a", "Ã", "a", "Ä", "a", "Å", "a",
		"Ç", "c",
		"È", "e", "É", "e", "Ê", "e", "Ë", "e",
		"Ì", "i", "Í", "i", "Î", "i", "Ï", "i",
		"Ñ", "n",
		"Ò", "o", "Ó", "o", "Ô", "o", "Õ", "o", "Ö", "o", "Ø", "o",
		"Ù", "u", "Ú", "u", "Û", "u", "Ü", "u",
	)
	return r.Replace(s)
}

func cleanSearchQuery(q string) string {
	q = strings.ReplaceAll(q, "&", " ")
	q = strings.ReplaceAll(q, "(", " ")
	q = strings.ReplaceAll(q, ")", " ")
	q = strings.ReplaceAll(q, "[", " ")
	q = strings.ReplaceAll(q, "]", " ")
	q = strings.ReplaceAll(q, "-", " ")
	q = strings.ReplaceAll(q, "'", "") // strip quote
	q = strings.ReplaceAll(q, "’", "") // curly quote
	q = strings.ReplaceAll(q, "\"", " ")
	
	words := strings.Fields(q)
	return strings.Join(words, " ")
}

func (m *Model) fetchLyricsCmd(artist, title, trackID string) tea.Cmd {
	return func() tea.Msg {
		cleanArtist := cleanSearchTerm(artist)
		cleanTitle := cleanSearchTerm(title)
		normArtist := stripAccents(cleanArtist)
		normTitle := stripAccents(cleanTitle)

		qOrig := cleanSearchQuery(fmt.Sprintf("%s %s", cleanArtist, cleanTitle))
		qNorm := stripAccents(qOrig)

		type lyricTask struct {
			name     string
			isSearch bool
			param1   string // artist for get, query for search
			param2   string // track_name for get, empty for search
		}

		var tasks []lyricTask
		// Task 1: exact get with original metadata
		tasks = append(tasks, lyricTask{name: "exact_orig", isSearch: false, param1: cleanArtist, param2: cleanTitle})

		// Task 2: exact get with accents stripped (if different)
		if normArtist != cleanArtist || normTitle != cleanTitle {
			tasks = append(tasks, lyricTask{name: "exact_norm", isSearch: false, param1: normArtist, param2: normTitle})
		}

		// Task 3: search query with original
		tasks = append(tasks, lyricTask{name: "search_orig", isSearch: true, param1: qOrig})

		// Task 4: search query with accents stripped (if different)
		if qNorm != qOrig {
			tasks = append(tasks, lyricTask{name: "search_norm", isSearch: true, param1: qNorm})
		}

		// Increased total timeout to 12s to account for slow networks
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()

		type taskResult struct {
			plainLyrics  string
			syncedLyrics []SyncedLine
		}

		resChan := make(chan taskResult, len(tasks))
		var wg sync.WaitGroup

		// doneChan is closed when all launched goroutines complete
		doneChan := make(chan struct{})
		
		// To track actively running tasks and close doneChan safely
		var launchedCount int32
		var completedCount int32

		for i, t := range tasks {
			// Check if we already got a result before starting the next task (staggered delay)
			if i > 0 {
				select {
				case res := <-resChan:
					return lyricsLoadedMsg{
						plainLyrics:  res.plainLyrics,
						syncedLyrics: res.syncedLyrics,
						trackID:      trackID,
					}
				case <-ctx.Done():
					return lyricsLoadedMsg{err: fmt.Errorf("no lyrics found (request timed out)"), trackID: trackID}
				case <-time.After(1200 * time.Millisecond):
					// Proceed to launch next task
				}
			}

			atomic.AddInt32(&launchedCount, 1)
			wg.Add(1)
			go func(task lyricTask) {
				defer wg.Done()
				defer func() {
					if atomic.AddInt32(&completedCount, 1) == atomic.LoadInt32(&launchedCount) {
						// Only close doneChan if we've completed all currently launched tasks
						// Note: since doneChan is checked in the select loop, we can just use a helper goroutine
					}
				}()

				var urlStr string
				if task.isSearch {
					urlStr = fmt.Sprintf("https://lrclib.net/api/search?q=%s", url.QueryEscape(task.param1))
				} else {
					urlStr = fmt.Sprintf("https://lrclib.net/api/get?artist=%s&track_name=%s",
						url.QueryEscape(task.param1), url.QueryEscape(task.param2))
				}

				req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
				if err != nil {
					return
				}

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return
				}

				if task.isSearch {
					var searchResults []struct {
						PlainLyrics  string `json:"plainLyrics"`
						SyncedLyrics string `json:"syncedLyrics"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&searchResults); err == nil && len(searchResults) > 0 {
						resChan <- taskResult{
							plainLyrics:  searchResults[0].PlainLyrics,
							syncedLyrics: parseLRC(searchResults[0].SyncedLyrics),
						}
						cancel() // Cancel other requests immediately!
					}
				} else {
					var res struct {
						PlainLyrics  string `json:"plainLyrics"`
						SyncedLyrics string `json:"syncedLyrics"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
						resChan <- taskResult{
							plainLyrics:  res.PlainLyrics,
							syncedLyrics: parseLRC(res.SyncedLyrics),
						}
						cancel() // Cancel other requests immediately!
					}
				}
			}(t)
		}

		// Background waiter to notify when ALL workers have completed
		go func() {
			wg.Wait()
			close(doneChan)
		}()

		select {
		case res := <-resChan:
			return lyricsLoadedMsg{
				plainLyrics:  res.plainLyrics,
				syncedLyrics: res.syncedLyrics,
				trackID:      trackID,
			}
		case <-doneChan:
			select {
			case res := <-resChan:
				return lyricsLoadedMsg{
					plainLyrics:  res.plainLyrics,
					syncedLyrics: res.syncedLyrics,
					trackID:      trackID,
				}
			default:
				return lyricsLoadedMsg{err: fmt.Errorf("no lyrics found for this track"), trackID: trackID}
			}
		case <-ctx.Done():
			select {
			case res := <-resChan:
				return lyricsLoadedMsg{
					plainLyrics:  res.plainLyrics,
					syncedLyrics: res.syncedLyrics,
					trackID:      trackID,
				}
			default:
				return lyricsLoadedMsg{err: fmt.Errorf("no lyrics found (request timed out)"), trackID: trackID}
			}
		}
	}
}

func cleanSearchTerm(s string) string {
	// Remove common suffixes like "(Official Video)", "[Official Music Video]", etc.
	s = strings.ToLower(s)
	suffixes := []string{
		"official video", "official music video", "lyric video", "lyrics video",
		"official audio", "high quality", "hq", "remastered", "remaster",
	}
	for _, suffix := range suffixes {
		s = strings.ReplaceAll(s, "("+suffix+")", "")
		s = strings.ReplaceAll(s, "["+suffix+"]", "")
		s = strings.ReplaceAll(s, suffix, "")
	}
	return strings.TrimSpace(s)
}

func parseLRC(lrc string) []SyncedLine {
	var lines []SyncedLine
	rawLines := strings.Split(lrc, "\n")
	for _, line := range rawLines {
		line = strings.TrimSpace(line)
		if len(line) < 10 || !strings.HasPrefix(line, "[") {
			continue
		}
		endIdx := strings.Index(line, "]")
		if endIdx == -1 {
			continue
		}
		timeStr := line[1:endIdx]
		text := line[endIdx+1:]

		parts := strings.Split(timeStr, ":")
		if len(parts) < 2 {
			continue
		}
		minVal, _ := strconv.Atoi(parts[0])
		secStr := parts[1]
		var secVal int
		var milliVal int
		if dotIdx := strings.Index(secStr, "."); dotIdx != -1 {
			secVal, _ = strconv.Atoi(secStr[:dotIdx])
			milliStr := secStr[dotIdx+1:]
			if len(milliStr) == 2 {
				milliVal, _ = strconv.Atoi(milliStr)
				milliVal *= 10
			} else if len(milliStr) == 3 {
				milliVal, _ = strconv.Atoi(milliStr)
			} else if len(milliStr) == 1 {
				milliVal, _ = strconv.Atoi(milliStr)
				milliVal *= 100
			}
		} else {
			secVal, _ = strconv.Atoi(secStr)
		}

		totalSeconds := float64(minVal)*60.0 + float64(secVal) + float64(milliVal)/1000.0
		lines = append(lines, SyncedLine{
			Time: totalSeconds,
			Text: strings.TrimSpace(text),
		})
	}
	return lines
}

func (m *Model) renderLyrics() string {
	var sb strings.Builder
	title := "Lyrics"
	if m.trackLoaded {
		title = fmt.Sprintf("Lyrics: %s - %s", m.currentTrack.Artist, m.currentTrack.Title)
	}
	sb.WriteString(fmt.Sprintf("  %s\n\n", title))

	if m.lyricsLoading {
		sb.WriteString("  Loading lyrics from LRCLib...\n")
		return sb.String()
	}

	if m.lyricsError != nil {
		sb.WriteString(fmt.Sprintf("  Error loading lyrics: %v\n\n", m.lyricsError))
		sb.WriteString("  Press 'r' to retry fetching lyrics.\n")
		return sb.String()
	}

	mainHeight := m.height - 11
	h := mainHeight - 5
	if h < 5 {
		h = 5
	}

	// 1. Synced lyrics rendering
	if len(m.syncedLyrics) > 0 {
		activeIdx := -1
		for i, line := range m.syncedLyrics {
			if m.timePos >= line.Time {
				activeIdx = i
			} else {
				break
			}
		}

		start := activeIdx - h/2
		if start < 0 {
			start = 0
		}
		end := start + h
		if end > len(m.syncedLyrics) {
			end = len(m.syncedLyrics)
			start = end - h
			if start < 0 {
				start = 0
			}
		}

		for i := start; i < end; i++ {
			line := m.syncedLyrics[i]
			prefix := "  "
			lineStyle := lipgloss.NewStyle()
			
			if i == activeIdx {
				prefix = "▶ "
				lineStyle = lineStyle.
					Foreground(lipgloss.Color(m.theme.PrimaryHighlight)).
					Bold(true)
			} else {
				lineStyle = lineStyle.Foreground(lipgloss.Color(m.theme.TextSecondary))
			}
			
			sb.WriteString(lineStyle.Render(prefix+line.Text) + "\n")
		}
		return sb.String()
	}

	// 2. Plain lyrics rendering
	if m.plainLyrics != "" {
		lines := strings.Split(m.plainLyrics, "\n")
		
		// Clamp scroll offset
		if m.lyricsScrollOffset < 0 {
			m.lyricsScrollOffset = 0
		}
		if m.lyricsScrollOffset > len(lines)-h {
			m.lyricsScrollOffset = len(lines) - h
		}
		if m.lyricsScrollOffset < 0 {
			m.lyricsScrollOffset = 0
		}

		end := m.lyricsScrollOffset + h
		if end > len(lines) {
			end = len(lines)
		}

		scrollIndicator := ""
		if m.lyricsScrollOffset > 0 && end < len(lines) {
			scrollIndicator = " ▲ ▼"
		} else if m.lyricsScrollOffset > 0 {
			scrollIndicator = " ▲"
		} else if end < len(lines) {
			scrollIndicator = " ▼"
		}

		sb.WriteString(fmt.Sprintf("  Plain Lyrics (scroll with Up/Down)%s:\n\n", scrollIndicator))

		for i := m.lyricsScrollOffset; i < end; i++ {
			line := lines[i]
			lineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.TextPrimary))
			sb.WriteString(lineStyle.Render("  "+line) + "\n")
		}
		return sb.String()
	}

	sb.WriteString("  No lyrics available for this track.\n")
	return sb.String()
}

func (m *Model) preloadLyricsCmd(artist, title, trackID string) tea.Cmd {
	return func() tea.Msg {
		msg := m.fetchLyricsCmd(artist, title, trackID)()
		if lMsg, ok := msg.(lyricsLoadedMsg); ok {
			return lyricsPreloadedMsg{
				plainLyrics:  lMsg.plainLyrics,
				syncedLyrics: lMsg.syncedLyrics,
				err:          lMsg.err,
				trackID:      lMsg.trackID,
			}
		}
		return nil
	}
}

