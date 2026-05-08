package ui

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"uwatu-simulator/internal/director"
	"uwatu-simulator/internal/engine"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ─────────────────────────────────────────────
// Colour palette
// ─────────────────────────────────────────────
var (
	colBg       = lipgloss.Color("#0d1117")
	colPanel    = lipgloss.Color("#161b22")
	colBorder   = lipgloss.Color("#30363d")
	colAccent   = lipgloss.Color("#58a6ff")
	colGreen    = lipgloss.Color("#3fb950")
	colYellow   = lipgloss.Color("#d29922")
	colOrange   = lipgloss.Color("#f0883e")
	colRed      = lipgloss.Color("#f85149")
	colMuted    = lipgloss.Color("#8b949e")
	colWhite    = lipgloss.Color("#e6edf3")
	colCyan     = lipgloss.Color("#39d5c4")
	colPurple   = lipgloss.Color("#bc8cff")
	colSwapWarn = lipgloss.Color("#ff7b72")
)

// ─────────────────────────────────────────────
// Styles
// ─────────────────────────────────────────────
var (
	stylePanelBase = lipgloss.NewStyle().
			Background(colPanel).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colBorder).
			Padding(0, 1)

	styleHeader = lipgloss.NewStyle().
			Foreground(colAccent).
			Bold(true)

	styleLabel = lipgloss.NewStyle().
			Foreground(colMuted)

	styleValue = lipgloss.NewStyle().
			Foreground(colWhite).
			Bold(true)

	styleAlert = lipgloss.NewStyle().
			Foreground(colRed).
			Bold(true)

	styleOk = lipgloss.NewStyle().
		Foreground(colGreen)

	styleWarn = lipgloss.NewStyle().
			Foreground(colYellow)

	styleDim = lipgloss.NewStyle().
			Foreground(colMuted)

	styleFooter = lipgloss.NewStyle().
			Foreground(colMuted).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colBorder).
			PaddingTop(0)

	styleTitleBar = lipgloss.NewStyle().
			Background(colAccent).
			Foreground(colBg).
			Bold(true).
			Padding(0, 2)

	styleScenarioTag = lipgloss.NewStyle().
				Foreground(colPurple).
				Bold(true)

	styleSimSwap = lipgloss.NewStyle().
			Foreground(colSwapWarn).
			Bold(true)
)

// ─────────────────────────────────────────────
// Alert log entry
// ─────────────────────────────────────────────
type alertEntry struct {
	ts      time.Time
	device  string
	message string
	level   string // "warn" | "alert" | "info"
}

// ─────────────────────────────────────────────
// Per-device state
// ─────────────────────────────────────────────
type deviceState struct {
	snap        engine.TagSnapshot
	tempHistory [40]float64
	histIdx     int
	histFull    bool
	tickCount   int
}

func (d *deviceState) pushTemp(t float64) {
	d.tempHistory[d.histIdx] = t
	d.histIdx = (d.histIdx + 1) % 40
	if d.histIdx == 0 {
		d.histFull = true
	}
	d.tickCount++
}

func (d *deviceState) sparkline() string {
	bars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	count := 40
	if !d.histFull {
		count = d.histIdx
	}
	if count < 2 {
		return strings.Repeat("▁", 20)
	}

	// find min/max over rolling window
	minT, maxT := d.tempHistory[0], d.tempHistory[0]
	for i := 0; i < count; i++ {
		if d.tempHistory[i] < minT {
			minT = d.tempHistory[i]
		}
		if d.tempHistory[i] > maxT {
			maxT = d.tempHistory[i]
		}
	}
	rang := maxT - minT
	if rang < 0.01 {
		rang = 0.01
	}

	var sb strings.Builder
	start := 0
	if count > 20 {
		start = count - 20
	}
	for i := start; i < count; i++ {
		idx := int(math.Round((d.tempHistory[i]-minT)/rang*7))
		if idx < 0 {
			idx = 0
		}
		if idx > 7 {
			idx = 7
		}
		sb.WriteString(bars[idx])
	}
	return sb.String()
}

// ─────────────────────────────────────────────
// Dashboard model
// ─────────────────────────────────────────────
type Dashboard struct {
	devices    map[string]*deviceState
	deviceKeys []string // insertion-ordered
	alerts     []alertEntry
	scenario   string
	speed      int
	broker     string
	snapshots  <-chan engine.TagSnapshot
	spinner    spinner.Model
	width      int
	height     int
	totalTicks int
	startedAt  time.Time
	lastUpdate time.Time

	// Menu state
	simEngine     *engine.Engine
	showMenu      bool
	scenarioFiles []string
	scenarioNames []string
	menuCursor    int
}

func NewDashboard(snapshots <-chan engine.TagSnapshot, scenario string, speed int, broker string, eng *engine.Engine) *Dashboard {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colCyan)

	// Preload available scenarios from disk
	files, _ := filepath.Glob("config/scenarios/*.json")
	names := []string{"Baseline (Default Healthy)"}
	paths := []string{""}

	for _, f := range files {
		paths = append(paths, f)
		base := filepath.Base(f)
		names = append(names, strings.TrimSuffix(base, filepath.Ext(base)))
	}

	if scenario == "" {
		scenario = "Baseline (Default Healthy)"
	} else {
		scenario = filepath.Base(scenario)
	}

	return &Dashboard{
		devices:       make(map[string]*deviceState),
		snapshots:     snapshots,
		scenario:      scenario,
		speed:         speed,
		broker:        broker,
		spinner:       sp,
		startedAt:     time.Now(),
		simEngine:     eng,
		scenarioFiles: paths,
		scenarioNames: names,
	}
}

// ─────────────────────────────────────────────
// Bubbletea interface
// ─────────────────────────────────────────────
type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (d *Dashboard) Init() tea.Cmd {
	return tea.Batch(tickCmd(), d.spinner.Tick)
}

func (d *Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return d, tea.Quit

		// Menu controls
		case "s":
			d.showMenu = !d.showMenu
		case "up", "k":
			if d.showMenu && d.menuCursor > 0 {
				d.menuCursor--
			}
		case "down", "j":
			if d.showMenu && d.menuCursor < len(d.scenarioFiles)-1 {
				d.menuCursor++
			}
		case "enter":
			if d.showMenu {
				path := d.scenarioFiles[d.menuCursor]
				name := d.scenarioNames[d.menuCursor]

				var scn director.Scenario
				if path != "" {
					var err error
					scn, err = director.LoadScenario(path)
					if err != nil {
						d.addAlert("SYSTEM", fmt.Sprintf("Failed to load scenario: %v", err), "alert")
					}
				}

				// Apply to engine safely
				d.simEngine.SetScenario(scn)
				d.scenario = name
				d.showMenu = false
				d.addAlert("SYSTEM", fmt.Sprintf("Switched to scenario: %s", name), "info")
			}
		}

	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height

	case spinner.TickMsg:
		var cmd tea.Cmd
		d.spinner, cmd = d.spinner.Update(msg)
		return d, cmd

	case tickMsg:
		d.drainSnapshots()
		return d, tickCmd()
	}

	return d, nil
}

func (d *Dashboard) drainSnapshots() {
	for {
		select {
		case snap := <-d.snapshots:
			d.totalTicks++
			d.lastUpdate = time.Now()

			dev, ok := d.devices[snap.DeviceID]
			if !ok {
				dev = &deviceState{}
				d.devices[snap.DeviceID] = dev
				d.deviceKeys = append(d.deviceKeys, snap.DeviceID)
			}
			prev := dev.snap
			dev.snap = snap
			dev.pushTemp(snap.Temp)

			// Generate alerts
			if snap.SimSwap && !prev.SimSwap {
				d.addAlert(snap.DeviceID, "⚠  SIM SWAP DETECTED", "alert")
			}
			if snap.Temp > 39.2 {
				d.addAlert(snap.DeviceID, fmt.Sprintf("🌡  HIGH TEMP %.1f°C", snap.Temp), "alert")
			} else if snap.Temp > 39.0 {
				d.addAlert(snap.DeviceID, fmt.Sprintf("🌡  ELEVATED TEMP %.1f°C", snap.Temp), "warn")
			}
			if snap.BatteryPct < 20 && prev.BatteryPct >= 20 {
				d.addAlert(snap.DeviceID, fmt.Sprintf("🔋 BATTERY LOW %d%%", snap.BatteryPct), "warn")
			}

		default:
			return
		}
	}
}

func (d *Dashboard) addAlert(device, msg, level string) {
	entry := alertEntry{ts: time.Now(), device: device, message: msg, level: level}
	d.alerts = append(d.alerts, entry)
	// Keep last 50
	if len(d.alerts) > 50 {
		d.alerts = d.alerts[len(d.alerts)-50:]
	}
}

// ─────────────────────────────────────────────
// View
// ─────────────────────────────────────────────
func (d *Dashboard) View() string {
	if d.width == 0 {
		return "Initialising…"
	}

	var sections []string

	// ── Title bar ──────────────────────────────
	uptime := time.Since(d.startedAt).Round(time.Second)
	titleLeft := fmt.Sprintf("  🐄 UWATU DIGITAL TWIN  v1.0.0  │  %s%dx%s  │  scenario: %s",
		styleLabel.Render("speed "),
		d.speed,
		styleLabel.Render(""),
		styleScenarioTag.Render(d.scenario),
	)
	titleRight := fmt.Sprintf("uptime %s  %s  ", uptime, d.spinner.View())
	titlePad := d.width - lipgloss.Width(titleLeft) - lipgloss.Width(titleRight)
	if titlePad < 0 {
		titlePad = 0
	}
	titleBar := styleTitleBar.Width(d.width).Render(titleLeft + strings.Repeat(" ", titlePad) + titleRight)
	sections = append(sections, titleBar)

	// ── Swap Middle Content (Menu OR Device Panels) ──
	if d.showMenu {
		sections = append(sections, d.renderMenu())
	} else {
		// Device panels
		if len(d.deviceKeys) > 0 {
			panelWidth := (d.width - 4) / len(d.deviceKeys)
			var panels []string
			for _, key := range d.deviceKeys {
				dev := d.devices[key]
				panels = append(panels, d.renderDevicePanel(dev, panelWidth))
			}
			sections = append(sections, lipgloss.JoinHorizontal(lipgloss.Top, panels...))
		} else {
			sections = append(sections, styleDim.Render("  Waiting for first telemetry…"))
		}
	}

	// ── Sim time strip ─────────────────────────
	if len(d.deviceKeys) > 0 {
		firstDev := d.devices[d.deviceKeys[0]]
		simT := firstDev.snap.SimTime.Format("Mon 02 Jan 2006  15:04:05")
		isNight := firstDev.snap.SimTime.Hour() < 6 || firstDev.snap.SimTime.Hour() >= 18
		nightMark := ""
		if isNight {
			nightMark = "  🌙 NIGHT"
		} else {
			nightMark = "  ☀  DAY"
		}
		sections = append(sections,
			styleDim.Render("  SIM TIME  ")+styleValue.Render(simT)+styleDim.Render(nightMark)+
				styleDim.Render(fmt.Sprintf("   total ticks: %d", d.totalTicks)),
		)
	}

	// ── Alert log ──────────────────────────────
	alertHeight := d.height - lipgloss.Height(strings.Join(sections, "\n")) - 4
	if alertHeight < 3 {
		alertHeight = 3
	}
	sections = append(sections, d.renderAlertLog(alertHeight))

	// ── Footer ─────────────────────────────────
	footer := styleFooter.Width(d.width).Render(
		fmt.Sprintf("  broker: %s   │   [s] switch scenario  │  [q] quit", d.broker),
	)
	sections = append(sections, footer)

	return strings.Join(sections, "\n")
}

func (d *Dashboard) renderMenu() string {
	var sb strings.Builder
	sb.WriteString(styleHeader.Render("  SELECT SCENARIO TO INJECT") + "\n\n")

	for i, name := range d.scenarioNames {
		cursor := "    "
		style := styleDim
		if i == d.menuCursor {
			cursor = "  > "
			style = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
		}
		sb.WriteString(cursor + style.Render(name) + "\n")
	}

	sb.WriteString("\n" + styleDim.Render("    [↑/↓] Navigate   [Enter] Select   [s] Cancel"))

	return stylePanelBase.Width(d.width - 2).Render(sb.String())
}

func (d *Dashboard) renderDevicePanel(dev *deviceState, width int) string {
	snap := dev.snap
	if snap.DeviceID == "" {
		return stylePanelBase.Width(width - 2).Render(styleDim.Render("no data"))
	}

	// ── Header ──
	header := styleHeader.Render(snap.DeviceID) + "  " + styleDim.Render(snap.AnimalID)

	// ── Temperature ──
	tempColor := colGreen
	tempLabel := "NORMAL"
	if snap.Temp >= 39.2 {
		tempColor = colRed
		tempLabel = "HIGH !"
	} else if snap.Temp >= 39.0 {
		tempColor = colOrange
		tempLabel = "ELEVATED"
	} else if snap.Temp < 37.8 {
		tempColor = colYellow
		tempLabel = "LOW"
	}
	tempStyle := lipgloss.NewStyle().Foreground(tempColor).Bold(true)
	tempLine := styleLabel.Render("TEMP  ") +
		tempStyle.Render(fmt.Sprintf("%.2f°C", snap.Temp)) +
		"  " + tempStyle.Render(tempLabel)

	// Spark
	sparkLine := styleLabel.Render("      ") +
		lipgloss.NewStyle().Foreground(tempColor).Render(dev.sparkline())

	// ── Accelerometer ──
	accelBar := accelMiniBar(snap.Accel, 100)
	accelLine := styleLabel.Render("ACCEL ") +
		styleValue.Render(fmt.Sprintf("%3d g", snap.Accel)) +
		"  " + accelBar

	// ── Battery ──
	battColor := colGreen
	if snap.BatteryPct < 20 {
		battColor = colRed
	} else if snap.BatteryPct < 40 {
		battColor = colYellow
	}
	battStyle := lipgloss.NewStyle().Foreground(battColor).Bold(true)
	battLine := styleLabel.Render("BATT  ") +
		battStyle.Render(fmt.Sprintf("%3d%%", snap.BatteryPct)) +
		styleDim.Render(fmt.Sprintf("  %dmV", snap.BatteryMv))

	// ── Location ──
	locLine := styleLabel.Render("LOC   ")
	if snap.DemoLat != 0 || snap.DemoLon != 0 {
		locLine += styleValue.Render(fmt.Sprintf("%.4f, %.4f", snap.DemoLat, snap.DemoLon))
	} else {
		locLine += styleDim.Render("—")
	}

	// ── SIM swap ──
	simLine := styleLabel.Render("SIM   ")
	if snap.SimSwap {
		simLine += styleSimSwap.Render("⚠  SWAP DETECTED")
	} else {
		simLine += styleOk.Render("✓ OK")
	}

	// ── Uptime / seq ──
	metaLine := styleDim.Render(fmt.Sprintf("uptime %s   seq #%d",
		formatDuration(snap.UptimeS), snap.Seq))

	body := strings.Join([]string{
		header,
		strings.Repeat("─", width-4),
		tempLine,
		sparkLine,
		accelLine,
		battLine,
		locLine,
		simLine,
		metaLine,
	}, "\n")

	return stylePanelBase.Width(width - 2).Render(body)
}

func (d *Dashboard) renderAlertLog(height int) string {
	title := styleHeader.Render("  ALERT LOG")

	if len(d.alerts) == 0 {
		body := title + "\n" + styleOk.Render("  ✓ No alerts")
		return stylePanelBase.Width(d.width - 2).Height(height).Render(body)
	}

	// Show most-recent entries that fit
	var lines []string
	start := len(d.alerts) - (height - 2)
	if start < 0 {
		start = 0
	}
	for _, a := range d.alerts[start:] {
		ts := a.ts.Format("15:04:05")
		var levelStyle lipgloss.Style
		switch a.level {
		case "alert":
			levelStyle = styleAlert
		case "warn":
			levelStyle = styleWarn
		case "info":
			levelStyle = lipgloss.NewStyle().Foreground(colAccent)
		default:
			levelStyle = styleOk
		}
		lines = append(lines,
			styleDim.Render(ts)+"  "+
				lipgloss.NewStyle().Foreground(colCyan).Render(a.device)+"  "+
				levelStyle.Render(a.message),
		)
	}

	body := title + "\n" + strings.Join(lines, "\n")
	return stylePanelBase.Width(d.width - 2).Render(body)
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

func accelMiniBar(val, max int) string {
	filled := val * 10 / max
	if filled > 10 {
		filled = 10
	}
	if filled < 0 {
		filled = 0
	}

	color := colGreen
	if val > 60 {
		color = colOrange
	} else if val > 80 {
		color = colRed
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
	return lipgloss.NewStyle().Foreground(color).Render(bar)
}

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	return fmt.Sprintf("%dm%02ds", m, s)
}