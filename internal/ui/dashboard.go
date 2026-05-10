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
	colSwapWarn = lipgloss.Color("#ff7b72")

	// Themed accents
	colScenario = lipgloss.Color("#bd93f9") // Purple
	colLabelKey = lipgloss.Color("#8be9fd") // Vibrant Cyan for keys
	colSimTime  = lipgloss.Color("#f1fa8c") // Soft Yellow
)

// Demo farm boundary (Kericho farm)
var farmBoundary = [][2]float64{
	{35.296318, -0.364199},
	{35.306318, -0.364199},
	{35.306318, -0.354199},
	{35.296318, -0.354199},
}

// Fine-grained speed increments
var speedSteps = []int{1, 2, 5, 10, 15, 20, 25, 30, 45, 60, 120, 300, 600, 1800, 3600, 7200}

// Point‑in‑polygon (ray‑casting) for the TUI
func pointInPolygon(lat, lon float64, polygon [][2]float64) bool {
	inside := false
	n := len(polygon)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		pi, pj := polygon[i], polygon[j]
		if (pi[1] > lat) != (pj[1] > lat) &&
			lon < (pj[0]-pi[0])*(lat-pi[1])/(pj[1]-pi[1])+pi[0] {
			inside = !inside
		}
	}
	return inside
}

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

	styleLabelKey = lipgloss.NewStyle().
			Foreground(colLabelKey)

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

	styleSimSwap = lipgloss.NewStyle().
			Foreground(colSwapWarn).
			Bold(true)

	styleLink = lipgloss.NewStyle().
			Foreground(colAccent).
			Underline(true)
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
	snap            engine.TagSnapshot
	tempHistory     [40]float64
	histIdx         int
	histFull        bool
	tickCount       int
	lastLogRealTime time.Time
	outside         bool // true if cow left farm boundary
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
	deviceKeys []string
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

		// Restart current scenario
		case "r":
			d.simEngine.RestartScenario()
			d.addAlert("SYSTEM", "Scenario restarted from Hour 0", "info")

		// Live Speed Controls
		case "+", "=":
			for _, s := range speedSteps {
				if d.speed < s {
					d.speed = s
					break
				}
			}
			d.simEngine.SetSpeed(d.speed)
			d.addAlert("SYSTEM", fmt.Sprintf("Time-warp speed increased to %dx", d.speed), "info")

		case "-", "_":
			for i := len(speedSteps) - 1; i >= 0; i-- {
				if d.speed > speedSteps[i] {
					d.speed = speedSteps[i]
					break
				}
			}
			d.simEngine.SetSpeed(d.speed)
			d.addAlert("SYSTEM", fmt.Sprintf("Time-warp speed decreased to %dx", d.speed), "info")

		// Menu controls
		case "s", "m":
			d.showMenu = !d.showMenu
		case "up", "k":
			if d.showMenu && d.menuCursor > 0 {
				d.menuCursor--
			}
		case "down", "j":
			if d.showMenu && d.menuCursor < len(d.scenarioFiles)-1 {
				d.menuCursor++
			}
		case "escape":
			d.showMenu = false
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
				dev = &deviceState{lastLogRealTime: time.Now()}
				d.devices[snap.DeviceID] = dev
				d.deviceKeys = append(d.deviceKeys, snap.DeviceID)
			}
			
			prevSnap := dev.snap
			prevOutside := dev.outside

			dev.snap = snap
			dev.pushTemp(snap.Temp)

			// Track whether cow is outside farm boundary
			dev.outside = !pointInPolygon(snap.Lat, snap.Lon, farmBoundary)
			if dev.outside && !prevOutside {
				d.addAlert(snap.DeviceID, "Cow left farm boundary", "warn")
			}

			if time.Since(dev.lastLogRealTime) > 500*time.Millisecond {
				d.addAlert(snap.DeviceID, fmt.Sprintf("Telemetry packet sent | Seq: %d | Temp: %.1f°C | Accel: %dg", snap.Seq, snap.Temp, snap.Accel), "info")
				dev.lastLogRealTime = time.Now()
			}

			if snap.SimSwap && !prevSnap.SimSwap {
				d.addAlert(snap.DeviceID, "⚠  SIM SWAP DETECTED", "alert")
			}
			if snap.Temp > 39.7 {
				d.addAlert(snap.DeviceID, fmt.Sprintf("HIGH TEMP %.1f°C", snap.Temp), "alert")
			} else if snap.Temp > 39.5 {
				d.addAlert(snap.DeviceID, fmt.Sprintf("ELEVATED TEMP %.1f°C", snap.Temp), "warn")
			}
			if snap.BatteryPct < 20 && prevSnap.BatteryPct >= 20 {
				d.addAlert(snap.DeviceID, fmt.Sprintf("BATTERY LOW %d%%", snap.BatteryPct), "warn")
			}

		default:
			return
		}
	}
}

func (d *Dashboard) addAlert(device, msg, level string) {
	entry := alertEntry{ts: time.Now(), device: device, message: msg, level: level}
	d.alerts = append(d.alerts, entry)
	if len(d.alerts) > 50 {
		d.alerts = d.alerts[len(d.alerts)-50:]
	}
}

// ─────────────────────────────────────────────
// View Output Construction
// ─────────────────────────────────────────────
func (d *Dashboard) View() string {
	if d.width == 0 || d.height == 0 {
		return "Initialising…"
	}

	// 1. High Visibility Title (Hardcoded colors so it works on all terminal profiles)
	titleBlock := lipgloss.NewStyle().
		// Background(lipgloss.Color("#ff79c6")).
		Foreground(lipgloss.Color("#00ff00")).
		Bold(true).
		Padding(0, 1).
		Render("Uwatu Simulator v0.1.1")

	// 2. Info Row
	scenName := lipgloss.NewStyle().Foreground(colScenario).Bold(true).Render(d.scenario)
	speedStr := lipgloss.NewStyle().Foreground(colWhite).Bold(true).Render(fmt.Sprintf("%dx", d.speed))
	uptime := time.Since(d.startedAt).Round(time.Second).String()

	infoRow := styleDim.Render("SCENARIO: ") + scenName +
		styleDim.Render("   │   SPEED: ") + speedStr +
		styleDim.Render("   │   UPTIME: ") + styleDim.Render(uptime) + " " + d.spinner.View()

	// 3. Device panels
	var panelsView string
	if len(d.deviceKeys) > 0 {
		panelWidth := (d.width - 4) / len(d.deviceKeys)
		var panels []string
		for _, key := range d.deviceKeys {
			dev := d.devices[key]
			panels = append(panels, d.renderDevicePanel(dev, panelWidth))
		}
		panelsView = lipgloss.JoinHorizontal(lipgloss.Top, panels...)
	} else {
		panelsView = styleDim.Render("  Waiting for first telemetry…")
	}

	// 4. Sim time strip & Web UI link
	var simTimeView string
	if len(d.deviceKeys) > 0 {
		firstDev := d.devices[d.deviceKeys[0]]
		simT := firstDev.snap.SimTime.Format("Mon 02 Jan 2006  15:04:05")
		isNight := firstDev.snap.SimTime.Hour() < 6 || firstDev.snap.SimTime.Hour() >= 18
		nightMark := "  ☀  DAY"
		if isNight {
			nightMark = "  🌙 NIGHT"
		}
		
		webLink := styleLink.Render("http://localhost:3000/dashboard")

		simTimeView = styleDim.Render("  SIM TIME  ") + 
			lipgloss.NewStyle().Foreground(colSimTime).Bold(true).Render(simT) + 
			styleDim.Render(nightMark) +
			styleDim.Render(fmt.Sprintf("   total ticks: %d   │  : ", d.totalTicks)) + 
			webLink
	}

	// 5. Assemble Top Section
	// We add an empty string "" at the absolute top so the title isn't glued to the roof of the terminal
	topSection := lipgloss.JoinVertical(lipgloss.Left,
		"", // Extra top padding to prevent top-edge clipping
		titleBlock,
		"", // spacing
		infoRow,
		"", // spacing
		panelsView,
		"", // spacing
		simTimeView,
		"", // spacing
	)

	// 6. Footer
	footerStr := fmt.Sprintf("  broker: %s   │   [-] slower  [+] faster   │   [r] restart   │   [s] menu  │  [q] quit", d.broker)
	footerView := styleFooter.Width(d.width).Render(footerStr)

	// 7. Precise Height Calculations
	// Subtracting 3 forces the UI to leave empty lines at the bottom of the screen.
	// This physically guarantees the terminal will not scroll and the footer won't hug VSCode.
	safeHeight := d.height - 3
	usedHeight := lipgloss.Height(topSection) + lipgloss.Height(footerView)

	middleHeight := safeHeight - usedHeight
	if middleHeight < 5 {
		middleHeight = 5 // Absolute minimum to avoid crashes
	}

	// 8. Middle Content (Menu or Logs)
	var middleView string
	if d.showMenu {
		middleView = d.renderMenu(middleHeight)
	} else {
		middleView = d.renderAlertLog(middleHeight)
	}

	// Final Assemble
	return lipgloss.JoinVertical(lipgloss.Left,
		topSection,
		middleView,
		footerView,
	)
}

func (d *Dashboard) renderMenu(height int) string {
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

	sb.WriteString("\n" + styleDim.Render("    [↑/↓] Navigate   [Enter] Select   [ESC] Cancel"))
	return stylePanelBase.Width(d.width - 2).Height(height).Render(sb.String())
}

func (d *Dashboard) renderDevicePanel(dev *deviceState, width int) string {
	snap := dev.snap
	if snap.DeviceID == "" {
		return stylePanelBase.Width(width - 2).Render(styleDim.Render("no data"))
	}

	// Determine panel colour based on device status
	titleColor := colGreen
	statusLabel := "HEALTHY"
	if dev.outside {
		if snap.SimSwap {
			titleColor = colRed
			statusLabel = "⚠ SWAPPED"
		} else {
			titleColor = colOrange
			statusLabel = "⚠ LEFT FARM"
		}
	}

	header := lipgloss.NewStyle().Foreground(titleColor).Bold(true).Render(snap.DeviceID) +
		"  " + styleDim.Render(snap.AnimalID) +
		"  " + lipgloss.NewStyle().Foreground(titleColor).Render(statusLabel)

	tempColor := colGreen
	tempLabel := "NORMAL"
	if snap.Temp >= 39.7 {
		tempColor = colRed
		tempLabel = "HIGH !"
	} else if snap.Temp >= 39.5 {
		tempColor = colOrange
		tempLabel = "ELEVATED"
	} else if snap.Temp < 37.8 {
		tempColor = colYellow
		tempLabel = "LOW"
	}

	tempStyle := lipgloss.NewStyle().Foreground(tempColor).Bold(true)
	
	tempLine := styleLabelKey.Render("TEMPERATURE    ") +
		tempStyle.Render(fmt.Sprintf("%.2f°C", snap.Temp)) +
		"  " + tempStyle.Render(tempLabel)

	sparkLine := styleLabel.Render("               ") +
		lipgloss.NewStyle().Foreground(tempColor).Render(dev.sparkline())

	accelBar := accelMiniBar(snap.Accel, 100)
	accelLine := styleLabelKey.Render("ACCELEROMETER  ") +
		styleValue.Render(fmt.Sprintf("%3d g", snap.Accel)) +
		"  " + accelBar

	battColor := colGreen
	if snap.BatteryPct < 20 {
		battColor = colRed
	} else if snap.BatteryPct < 40 {
		battColor = colYellow
	}
	battStyle := lipgloss.NewStyle().Foreground(battColor).Bold(true)
	battLine := styleLabelKey.Render("BATTERY        ") +
		battStyle.Render(fmt.Sprintf("%3d%%", snap.BatteryPct)) +
		styleDim.Render(fmt.Sprintf("  %dmV", snap.BatteryMv))

	locLine := styleLabelKey.Render("LOCATION       ") + styleValue.Render(fmt.Sprintf("%.4f, %.4f", snap.Lat, snap.Lon))

	// SIM status line (only show when swapped or to indicate OK)
	simLine := styleLabelKey.Render("SIM STATUS     ")
	if snap.SimSwap {
		simLine += styleSimSwap.Render("⚠  SWAP DETECTED")
	} else {
		simLine += styleOk.Render("✓ OK")
	}

	metaLine := styleDim.Render(fmt.Sprintf("uptime %s   seq #%d", formatDuration(snap.UptimeS), snap.Seq))

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
	title := styleHeader.Render("  SYSTEM & TELEMETRY LOG")

	if len(d.alerts) == 0 {
		body := title + "\n" + styleOk.Render("  ✓ No events yet...")
		return stylePanelBase.Width(d.width - 2).Height(height).Render(body)
	}

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
			levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#56d364"))
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
	return stylePanelBase.Width(d.width - 2).Height(height).Render(body)
}

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