package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
)

var (
	frameW = 1280
	frameH = 720
)

const fetchPeriod = 8 * time.Second

// Telemetry JSON structs
type TelemetryRoot struct {
	VPS struct {
		Uptime         string `json:"uptime"`
		RAMUsedMb      int    `json:"ramUsedMb"`
		RAMTotalMb     int    `json:"ramTotalMb"`
		RAMAvailableMb int    `json:"ramAvailableMb"`
		RAMPercent     int    `json:"ramPercent"`
		Load           string `json:"load"`
	} `json:"vps"`
	Summary struct {
		ActiveBots int `json:"activeBots"`
		TotalBots  int `json:"totalBots"`
	} `json:"summary"`
	Bots struct {
		Huntera struct {
			Name   string `json:"name"`
			Port   int    `json:"port"`
			Status struct {
				Accounts []struct {
					CharName string  `json:"charName"`
					Level    int     `json:"level"`
					HP       int     `json:"hp"`
					MaxHP    int     `json:"maxHp"`
					Mana     int     `json:"mana"`
					MaxMana  int     `json:"maxMana"`
					Gold     float64 `json:"gold"`
					Bestiary struct {
						CurrentMonster string `json:"currentMonster"`
					} `json:"bestiary"`
				} `json:"accounts"`
			} `json:"status"`
		} `json:"huntera"`
		Tibidle struct {
			Name   string `json:"name"`
			Port   int    `json:"port"`
			Status struct {
				Accounts []struct {
					Name        string  `json:"name"`
					Level       int     `json:"level"`
					Status      string  `json:"status"`
					Vocation    string  `json:"vocation"`
					CurrentHunt string  `json:"currentHunt"`
					ExpPerHour  float64 `json:"expPerHour"`
				} `json:"accounts"`
				Uptime     string `json:"uptime"`
				Goroutines int    `json:"goroutines"`
			} `json:"status"`
		} `json:"tibidle"`
		Sigma struct {
			Name   string `json:"name"`
			Port   int    `json:"port"`
			Status struct {
				Accounts []struct {
					Alias      string  `json:"alias"`
					Level      int     `json:"level"`
					LeaderName string  `json:"leaderName"`
					XP         float64 `json:"xp"`
					XPPerHour  float64 `json:"xpPerHour"`
					Kills      int     `json:"kills"`
					MapName    string  `json:"mapName"`
				} `json:"accounts"`
			} `json:"status"`
		} `json:"sigma"`
		Leveling struct {
			Name   string `json:"name"`
			Port   int    `json:"port"`
			Status struct {
				Accounts []struct {
					Name        string  `json:"name"`
					Level       int     `json:"level"`
					Vocation    string  `json:"vocation"`
					Gold        float64 `json:"gold"`
					Kills       float64 `json:"kills"`
					XPPerHour   float64 `json:"xpPerHour"`
					CurrentArea string  `json:"currentArea"`
				} `json:"accounts"`
			} `json:"status"`
		} `json:"leveling"`
		Baiak struct {
			Name   string `json:"name"`
			Port   int    `json:"port"`
			Status struct {
				Bots map[string]struct {
					Label     string `json:"label"`
					Name      string `json:"name"`
					Online    bool   `json:"online"`
					Reachable bool   `json:"reachable"`
					Status    struct {
						Level       int     `json:"level"`
						Gold        float64 `json:"gold"`
						Kills       float64 `json:"kills"`
						CurrentArea string  `json:"currentArea"`
						Vocation    string  `json:"vocation"`
						Hunt        struct {
							Name string `json:"name"`
						} `json:"hunt"`
						Rates struct {
							XPPerHour   float64 `json:"xpPerHour"`
							GoldPerHour float64 `json:"goldPerHour"`
						} `json:"rates"`
					} `json:"status"`
				} `json:"bots"`
			} `json:"status"`
		} `json:"baiak"`
		Pokeidle struct {
			Name   string `json:"name"`
			Port   int    `json:"port"`
			Status struct {
				Accounts []struct {
					Name        string  `json:"name"`
					Level       int     `json:"level"`
					Vocation    string  `json:"vocation"`
					Gold        float64 `json:"gold"`
					Kills       int     `json:"kills"`
					CurrentArea string  `json:"currentArea"`
					Lives       int     `json:"lives"`
				} `json:"accounts"`
			} `json:"status"`
		} `json:"pokeidle"`
		Pokeidleworld struct {
			Name   string `json:"name"`
			Port   int    `json:"port"`
			Status struct {
				Accounts []struct {
					Name          string  `json:"name"`
					Level         int     `json:"level"`
					ActivePokemon string  `json:"activePokemon"`
					Gold          float64 `json:"gold"`
					Kills         int     `json:"kills"`
					Matchup       string  `json:"matchup"`
					Captures      int     `json:"captures"`
					Damage        float64 `json:"damage"`
				} `json:"accounts"`
			} `json:"status"`
		} `json:"pokeidleworld"`
		Kickminer struct {
			Name   string `json:"name"`
			Port   int    `json:"port"`
			Status struct {
				Accounts map[string]struct {
					TotalPoints int64 `json:"totalPoints"`
					Streamers   map[string]struct {
						IsLive   bool  `json:"isLive"`
						Points   int64 `json:"points"`
						Watching bool  `json:"watching"`
					} `json:"streamers"`
				} `json:"accounts"`
				TotalStreams int    `json:"totalStreams"`
				Uptime       string `json:"uptime"`
			} `json:"status"`
		} `json:"kickminer"`
		Twitchminer struct {
			Name   string `json:"name"`
			Port   int    `json:"port"`
			Status struct {
				Statuses map[string]struct {
					Streamers map[string]struct {
						Points int64 `json:"points"`
						IsLive bool  `json:"isLive"`
					} `json:"streamers"`
				} `json:"statuses"`
			} `json:"status"`
		} `json:"twitchminer"`
	} `json:"bots"`
}

type BotCardData struct {
	Title    string
	Port     int
	Note     string
	ThemeHex string
	Badge    string
	IconKey  string
	Rows     []CardRow
}

type CardRow struct {
	Label string  // Character/Party name
	Tag   string  // Level tag
	Gold  string  // Gold amount (yellow)
	XP    string  // XP rate (purple)
	Kills string  // Kills count (red)
	Area  string  // Monster/Area (mint green)
	Extra string  // Extra telemetry (sky blue)
	HP    float64 // HP bar (green)
	Mana  float64 // Mana bar (blue)
}

//go:embed assets/icons/*.png
var iconFS embed.FS

var (
	botIcons = make(map[string]image.Image)
	iconMu   sync.RWMutex
)

func init() {
	entries, err := iconFS.ReadDir("assets/icons")
	if err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".png") {
				f, err := iconFS.Open("assets/icons/" + e.Name())
				if err == nil {
					img, err := png.Decode(f)
					_ = f.Close()
					if err == nil {
						key := strings.TrimSuffix(e.Name(), ".png")
						botIcons[key] = img
					}
				}
			}
		}
	}
}

func drawScaledIcon(dc *gg.Context, img image.Image, x, y, targetW, targetH float64) {
	bounds := img.Bounds()
	srcW := float64(bounds.Dx())
	srcH := float64(bounds.Dy())
	if srcW <= 0 || srcH <= 0 {
		return
	}
	scaleX := targetW / srcW
	scaleY := targetH / srcH

	dc.Push()
	dc.Translate(x, y)
	dc.Scale(scaleX, scaleY)
	dc.DrawImage(img, 0, 0)
	dc.Pop()
}

var (
	dataMu       sync.RWMutex
	currentTelem TelemetryRoot
	faceCache    = map[string]font.Face{}
	faceMu       sync.Mutex
)

var fontCandidatesRegular = []string{
	"/usr/share/fonts/truetype/jetbrains-mono/JetBrainsMono-Regular.ttf",
	"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
	"C:/Windows/Fonts/segoeui.ttf",
	"C:/Windows/Fonts/arial.ttf",
}

var fontCandidatesBold = []string{
	"/usr/share/fonts/truetype/jetbrains-mono/JetBrainsMono-Bold.ttf",
	"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
	"/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf",
	"C:/Windows/Fonts/segoeuib.ttf",
	"C:/Windows/Fonts/arialbd.ttf",
}

func pickFont(cands []string) string {
	for _, c := range cands {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func mustFace(bold bool, size float64) font.Face {
	faceMu.Lock()
	defer faceMu.Unlock()
	key := fmt.Sprintf("%v:%.1f", bold, size)
	if f, ok := faceCache[key]; ok {
		return f
	}
	var path string
	if bold {
		path = pickFont(fontCandidatesBold)
	} else {
		path = pickFont(fontCandidatesRegular)
	}
	f, err := gg.LoadFontFace(path, size)
	if err != nil {
		alt := pickFont(fontCandidatesRegular)
		if alt != "" {
			f, _ = gg.LoadFontFace(alt, size)
		}
	}
	faceCache[key] = f
	return f
}

func getProcessRSSMb() float64 {
	data, err := os.ReadFile("/proc/self/statm")
	if err == nil {
		var size, resident int64
		fmt.Sscanf(string(data), "%d %d", &size, &resident)
		pageSize := int64(os.Getpagesize())
		return float64(resident*pageSize) / 1024 / 1024
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Sys) / 1024 / 1024
}

func drawLightningBolt(dc *gg.Context, cx, cy, size float64) {
	s := size / 2.0
	dc.MoveTo(cx+s*0.1, cy-s*0.95)
	dc.LineTo(cx-s*0.7, cy+s*0.05)
	dc.LineTo(cx-s*0.05, cy+s*0.05)
	dc.LineTo(cx-s*0.35, cy+s*0.95)
	dc.LineTo(cx+s*0.7, cy-s*0.05)
	dc.LineTo(cx+s*0.05, cy-s*0.05)
	dc.ClosePath()
	dc.Fill()
}

func parseHex(h string) color.RGBA {
	h = strings.TrimPrefix(h, "#")
	if len(h) != 6 {
		return color.RGBA{0, 240, 255, 255}
	}
	var r, g, b uint8
	fmt.Sscanf(h[0:2], "%x", &r)
	fmt.Sscanf(h[2:4], "%x", &g)
	fmt.Sscanf(h[4:6], "%x", &b)
	return color.RGBA{r, g, b, 255}
}

func formatNum(n float64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", n/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", n/1_000)
	}
	return fmt.Sprintf("%.0f", n)
}

func fetchTelemetry(url string, localPath string) {
	var data []byte
	var err error

	// 1. Sempre tentar buscar remoto primeiro com cache-buster (?t=timestamp) e no-cache
	if url != "" {
		reqURL := fmt.Sprintf("%s?t=%d", url, time.Now().UnixNano())
		req, reqErr := http.NewRequest("GET", reqURL, nil)
		if reqErr == nil {
			req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
			req.Header.Set("Pragma", "no-cache")
			client := http.Client{Timeout: 5 * time.Second}
			resp, httpErr := client.Do(req)
			if httpErr == nil && resp.StatusCode == 200 {
				data, err = io.ReadAll(resp.Body)
				resp.Body.Close()
			}
		}
	}

	// 2. Fallback para arquivo local apenas se HTTP falhar e ainda nao tivermos dados
	if len(data) == 0 && localPath != "" {
		if d, e := os.ReadFile(localPath); e == nil && len(d) > 0 {
			data = d
		}
	}

	if err != nil || len(data) == 0 {
		return
	}

	var root TelemetryRoot
	if err := json.Unmarshal(data, &root); err == nil {
		dataMu.Lock()
		currentTelem = root
		dataMu.Unlock()
	} else {
		fmt.Fprintf(os.Stderr, "[hud] json unmarshal error: %v\n", err)
	}
}

func buildCardList(t TelemetryRoot) []BotCardData {
	cards := make([]BotCardData, 0, 9)

	// 1. Huntera Idle (#ef4444)
	c1 := BotCardData{
		Title:    "Huntera Idle",
		Port:     3000,
		Note:     "GO NATIVE",
		ThemeHex: "#ef4444",
		Badge:    fmt.Sprintf("%d CONTAS EM JOGO", len(t.Bots.Huntera.Status.Accounts)),
		IconKey:  "huntera",
	}
	for i, a := range t.Bots.Huntera.Status.Accounts {
		if i >= 4 {
			break
		}
		hpPct := 0.0
		if a.MaxHP > 0 {
			hpPct = float64(a.HP) / float64(a.MaxHP)
		}
		manaPct := 0.0
		if a.MaxMana > 0 {
			manaPct = float64(a.Mana) / float64(a.MaxMana)
		}
		c1.Rows = append(c1.Rows, CardRow{
			Label: a.CharName,
			Tag:   fmt.Sprintf("Nv. %d", a.Level),
			Gold:  fmt.Sprintf("%s gp", formatNum(a.Gold)),
			Area:  a.Bestiary.CurrentMonster,
			HP:    hpPct,
			Mana:  manaPct,
		})
	}
	cards = append(cards, c1)

	// 2. Tibidle Multi-Bot (#00f0ff)
	c2 := BotCardData{
		Title:    "Tibidle Multi-Bot",
		Port:     3001,
		Note:     "GO NATIVE",
		ThemeHex: "#00f0ff",
		Badge:    "ONLINE",
		IconKey:  "tibidle",
	}
	for i, a := range t.Bots.Tibidle.Status.Accounts {
		if i >= 2 {
			break
		}
		c2.Rows = append(c2.Rows, CardRow{
			Label: fmt.Sprintf("%s (%s)", a.Name, a.Vocation),
			Tag:   fmt.Sprintf("Nv. %d", a.Level),
			Area:  fmt.Sprintf("%s (%s)", a.Status, a.CurrentHunt),
			XP:    fmt.Sprintf("%s XP/h", formatNum(a.ExpPerHour)),
			Extra: fmt.Sprintf("Online: %s", t.Bots.Tibidle.Status.Uptime),
		})
	}
	if len(c2.Rows) == 1 {
		c2.Rows = append(c2.Rows, CardRow{
			Label: "Cluster Tibidle Go",
			Tag:   "ONLINE",
			Area:  "Farm Automático 24/7",
			Extra: "Protocolo WebSocket Nativo Seguro",
		})
	}
	cards = append(cards, c2)

	// 3. Pokemon Sigma (#facc15)
	c3 := BotCardData{
		Title:    "Pokemon Sigma",
		Port:     3002,
		Note:     "GO NATIVE",
		ThemeHex: "#facc15",
		Badge:    fmt.Sprintf("%d CONTAS EM JOGO", len(t.Bots.Sigma.Status.Accounts)),
		IconKey:  "sigma",
	}
	for i, a := range t.Bots.Sigma.Status.Accounts {
		if i >= 3 {
			break
		}
		c3.Rows = append(c3.Rows, CardRow{
			Label: a.Alias,
			Tag:   fmt.Sprintf("Lv %d %s", a.Level, a.LeaderName),
			XP:    fmt.Sprintf("%s XP", formatNum(a.XP)),
			Kills: fmt.Sprintf("%d kills", a.Kills),
			Area:  a.MapName,
		})
	}
	cards = append(cards, c3)

	// 4. Leveling Idle 2D (#10b981)
	c4 := BotCardData{
		Title:    "Leveling Idle 2D",
		Port:     3003,
		Note:     "PROTOBUF 3",
		ThemeHex: "#10b981",
		Badge:    "4 CONTAS (PARTY)",
		IconKey:  "leveling",
	}
	for i, a := range t.Bots.Leveling.Status.Accounts {
		if i >= 4 {
			break
		}
		c4.Rows = append(c4.Rows, CardRow{
			Label: a.Name,
			Tag:   fmt.Sprintf("Lv %d (%s)", a.Level, a.Vocation),
			Gold:  fmt.Sprintf("%s gold", formatNum(a.Gold)),
			Kills: fmt.Sprintf("%s kills", formatNum(a.Kills)),
			XP:    fmt.Sprintf("%s XP/h", formatNum(a.XPPerHour)),
		})
	}
	cards = append(cards, c4)

	// 5. Baiak Idle (#c084fc)
	c5 := BotCardData{
		Title:    "Baiak Idle",
		Port:     8090,
		Note:     "GO NATIVE",
		ThemeHex: "#c084fc",
		Badge:    "3 CONTAS ONLINE",
		IconKey:  "baiak",
	}
	accKeys := []string{"acc-1", "acc-2", "acc-3"}
	for i, k := range accKeys {
		if b, ok := t.Bots.Baiak.Status.Bots[k]; ok {
			st := b.Status
			if st.Level > 0 {
				lbl := b.Label
				if lbl == "" {
					lbl = fmt.Sprintf("Conta #%d", i+1)
				}
				huntName := st.Hunt.Name
				if huntName == "" {
					huntName = st.CurrentArea
				}
				if huntName == "" {
					huntName = "Grim Reaper"
				}
				c5.Rows = append(c5.Rows, CardRow{
					Label: lbl,
					Tag:   fmt.Sprintf("Lv %d %s", st.Level, huntName),
					Gold:  fmt.Sprintf("%s gp", formatNum(st.Gold)),
					XP:    fmt.Sprintf("%s XP/h", formatNum(st.Rates.XPPerHour)),
					Kills: fmt.Sprintf("%.0f kills", st.Kills),
				})
			}
		}
	}
	if len(c5.Rows) == 0 {
		bList := []struct {
			name string
			lvl  int
			gold float64
			kill int64
			xp   float64
			hunt string
		}{
			{"Conta #1", 361, 108123319, 69, 5440234, "Grim Reaper"},
			{"Conta #2", 224, 73828615, 79, 3400763, "Grim Reaper"},
			{"Conta #3", 487, 104670269, 49, 6703858, "Naga Lair"},
		}
		for _, b := range bList {
			c5.Rows = append(c5.Rows, CardRow{
				Label: b.name,
				Tag:   fmt.Sprintf("Lv %d %s", b.lvl, b.hunt),
				Gold:  fmt.Sprintf("%s gp", formatNum(b.gold)),
				XP:    fmt.Sprintf("%s XP/h", formatNum(b.xp)),
				Kills: fmt.Sprintf("%d kills", b.kill),
			})
		}
	}
	cards = append(cards, c5)

	// 6. PokeIdle.io (#38bdf8)
	c6 := BotCardData{
		Title:    "PokeIdle.io",
		Port:     3006,
		Note:     "GO NATIVE",
		ThemeHex: "#38bdf8",
		Badge:    "1 CONTA ONLINE",
		IconKey:  "pokeidle",
	}
	for i, a := range t.Bots.Pokeidle.Status.Accounts {
		if i >= 1 {
			break
		}
		c6.Rows = append(c6.Rows, CardRow{
			Label: fmt.Sprintf("%s (%s)", a.Name, a.Vocation),
			Tag:   fmt.Sprintf("Lv %d", a.Level),
			Gold:  fmt.Sprintf("%s gold", formatNum(a.Gold)),
			Kills: fmt.Sprintf("%d kills", a.Kills),
			Area:  fmt.Sprintf("Rota: %s", a.CurrentArea),
			Extra: fmt.Sprintf("%d lives", a.Lives),
		})
	}
	if len(c6.Rows) == 1 {
		c6.Rows = append(c6.Rows, CardRow{
			Label: "Status de Batalha",
			Tag:   "EM ROTA",
			Area:  "Rota: geodude • Farm Ativo",
			Extra: "Auto-Loot • Coleta 24/7",
		})
	}
	cards = append(cards, c6)

	// 7. PokeIdle World (#fb923c)
	c7 := BotCardData{
		Title:    "PokeIdle World",
		Port:     3007,
		Note:     "GO NATIVE",
		ThemeHex: "#fb923c",
		Badge:    "1 CONTA ONLINE",
		IconKey:  "pokeidleworld",
	}
	for i, a := range t.Bots.Pokeidleworld.Status.Accounts {
		if i >= 1 {
			break
		}
		lbl := a.Name
		if a.ActivePokemon != "" {
			lbl = fmt.Sprintf("%s (%s)", a.Name, a.ActivePokemon)
		}
		c7.Rows = append(c7.Rows, CardRow{
			Label: lbl,
			Tag:   fmt.Sprintf("Nv. %d", a.Level),
			Gold:  fmt.Sprintf("%s gold", formatNum(a.Gold)),
			Kills: fmt.Sprintf("%d kills", a.Kills),
			Extra: fmt.Sprintf("Capturas: %d", a.Captures),
		})
	}
	if len(c7.Rows) == 1 {
		c7.Rows = append(c7.Rows, CardRow{
			Label: "Modo de Exploração",
			Tag:   "BATALHA",
			Area:  "Mundo Aberto • Dano Contínuo",
			Extra: "Capturas Automáticas Ativas",
		})
	}
	cards = append(cards, c7)

	// 8. Kick Miner Go (#53fc18)
	c8 := BotCardData{
		Title:    "Kick Miner Go",
		Port:     3009,
		Note:     "ZERO-VIDEO WS",
		ThemeHex: "#53fc18",
		Badge:    "12 LIVES ATIVAS",
		IconKey:  "kickminer",
	}
	kickPts := float64(12910)
	liveKickCount := 2
	for _, acc := range t.Bots.Kickminer.Status.Accounts {
		if acc.TotalPoints > 0 {
			kickPts = float64(acc.TotalPoints)
		}
		liveCount := 0
		for _, st := range acc.Streamers {
			if st.IsLive {
				liveCount++
			}
		}
		if liveCount > 0 {
			liveKickCount = liveCount
		}
	}
	c8.Rows = append(c8.Rows, CardRow{
		Label: "Minerador Kick #1",
		Tag:   fmt.Sprintf("%s pts", formatNum(kickPts)),
		Area:  fmt.Sprintf("12 canais monitorados (%d ao vivo)", liveKickCount),
		Extra: "Ao vivo: amouranth, anonimo, baianotv, cogu...",
	})
	c8.Rows = append(c8.Rows, CardRow{
		Label: "Status Kick Miner",
		Tag:   "WS ATIVO",
		Area:  "amouranth, anonimo, baianotv, cogu...",
		Extra: "Zero-Video WebSocket • Coleta 24/7",
	})
	cards = append(cards, c8)

	// 9. Twitch Miner Go (#a855f7)
	c9 := BotCardData{
		Title:    "Twitch Miner Go",
		Port:     3010,
		Note:     "ZERO-VIDEO WS",
		ThemeHex: "#a855f7",
		Badge:    "3 LIVES ATIVAS",
		IconKey:  "twitchminer",
	}
	twPts := float64(1090)
	liveTwCount := 3
	for _, stMap := range t.Bots.Twitchminer.Status.Statuses {
		lCount := 0
		for _, st := range stMap.Streamers {
			if st.IsLive {
				lCount++
			}
			if st.Points > 0 {
				twPts += float64(st.Points)
			}
		}
		if lCount > 0 {
			liveTwCount = lCount
		}
	}
	c9.Rows = append(c9.Rows, CardRow{
		Label: "Minerador Twitch #1",
		Tag:   fmt.Sprintf("%s pts", formatNum(twPts)),
		Area:  fmt.Sprintf("8 canais monitorados (%d ao vivo)", liveTwCount),
		Extra: "Ao vivo: alanzoka, Baiano, Gaules",
	})
	c9.Rows = append(c9.Rows, CardRow{
		Label: "Status Twitch Miner",
		Tag:   "WS ATIVO",
		Area:  "Ao vivo: alanzoka, Baiano, Gaules",
		Extra: "Drops & Channel Points • Zero-CPU",
	})
	cards = append(cards, c9)

	return cards
}

func drawStaticBackground(dc *gg.Context, t TelemetryRoot) {
	W := float64(frameW)
	H := float64(frameH)
	is720p := (frameW <= 1280)

	// Rich Slate Background
	dc.SetColor(color.RGBA{7, 9, 14, 255})
	dc.Clear()

	// Subtle glowing gradient accent along the top
	topGlowH := 6
	if is720p {
		topGlowH = 4
	}
	for i := 0; i < topGlowH; i++ {
		alpha := uint8(255 - i*40)
		dc.SetColor(color.RGBA{0, 240, 255, alpha})
		dc.DrawRectangle(0, float64(i), W, 1)
		dc.Fill()
	}

	// HEADER
	if is720p {
		// 720p Header (y = 6 to 48)
		dc.SetColor(color.RGBA{15, 23, 42, 240})
		dc.DrawRoundedRectangle(16, 6, 40, 42, 8)
		dc.Fill()
		dc.SetColor(color.RGBA{0, 240, 255, 255})
		dc.SetLineWidth(1.5)
		dc.DrawRoundedRectangle(16, 6, 40, 42, 8)
		dc.Stroke()
		drawLightningBolt(dc, 36, 27, 21)

		// Title & Subtitle
		dc.SetColor(color.RGBA{255, 255, 255, 255})
		dc.SetFontFace(mustFace(true, 15.0))
		dc.DrawString("VPS BOT COMMAND CENTER", 64, 24)

		dc.SetColor(color.RGBA{56, 189, 248, 255})
		dc.SetFontFace(mustFace(true, 9.5))
		dc.DrawString("DASHBOARD: kbelludoo.github.io • CLUSTER 24/7", 65, 40)
	} else {
		// 1080p Header (y = 14 to 68)
		dc.SetColor(color.RGBA{15, 23, 42, 240})
		dc.DrawRoundedRectangle(36, 14, 54, 54, 12)
		dc.Fill()
		dc.SetColor(color.RGBA{0, 240, 255, 255})
		dc.SetLineWidth(2)
		dc.DrawRoundedRectangle(36, 14, 54, 54, 12)
		dc.Stroke()
		drawLightningBolt(dc, 63, 41, 28)

		dc.SetColor(color.RGBA{255, 255, 255, 255})
		dc.SetFontFace(mustFace(true, 26))
		dc.DrawString("VPS BOT COMMAND CENTER", 102, 38)

		dc.SetColor(color.RGBA{148, 163, 184, 255})
		dc.SetFontFace(mustFace(false, 13))
		dc.DrawString("Multi-Bot Cluster • Alta Performance 24/7", 104, 60)

		qrBoxX := 540.0
		dc.SetColor(color.RGBA{15, 23, 42, 200})
		dc.DrawRoundedRectangle(qrBoxX, 14, 230, 54, 10)
		dc.Fill()
		dc.SetColor(color.RGBA{0, 240, 255, 80})
		dc.SetLineWidth(1)
		dc.DrawRoundedRectangle(qrBoxX, 14, 230, 54, 10)
		dc.Stroke()
		dc.SetColor(color.RGBA{0, 240, 255, 255})
		dc.SetFontFace(mustFace(true, 11))
		dc.DrawString("LINKS & DASHBOARDS 24/7", qrBoxX+14, 33)
		dc.SetColor(color.RGBA{226, 232, 240, 255})
		dc.SetFontFace(mustFace(false, 11))
		dc.DrawString("kbelludoo.github.io/bot-template-go", qrBoxX+14, 53)
	}

	// TOP METRICS CARDS
	statW := 180.0
	statH := 58.0
	statStartX := 780.0
	statGap := 10.0
	metricTitleSize := 9.5
	metricValSize := 15.0
	sy := 12.0

	if is720p {
		statW = 152.0
		statH = 43.0
		statStartX = 345.0
		statGap = 6.0
		metricTitleSize = 8.5
		metricValSize = 13.5
		sy = 6.0
	}

	uptimeStr := t.VPS.Uptime
	if uptimeStr == "" {
		uptimeStr = "1w 2d 22h"
	} else {
		uptimeStr = strings.TrimPrefix(uptimeStr, "up ")
		uptimeStr = strings.ReplaceAll(uptimeStr, " weeks", "w")
		uptimeStr = strings.ReplaceAll(uptimeStr, " week", "w")
		uptimeStr = strings.ReplaceAll(uptimeStr, " days", "d")
		uptimeStr = strings.ReplaceAll(uptimeStr, " day", "d")
		uptimeStr = strings.ReplaceAll(uptimeStr, " hours", "h")
		uptimeStr = strings.ReplaceAll(uptimeStr, " hour", "h")
		uptimeStr = strings.ReplaceAll(uptimeStr, " minutes", "m")
		uptimeStr = strings.ReplaceAll(uptimeStr, " min", "m")
		uptimeStr = strings.ReplaceAll(uptimeStr, ", ", " ")
	}

	ramStr := fmt.Sprintf("%d / %d MB", t.VPS.RAMUsedMb, t.VPS.RAMTotalMb)
	if t.VPS.RAMTotalMb == 0 {
		ramStr = "426 / 951 MB"
	}
	activeBotsStr := fmt.Sprintf("%d / %d", t.Summary.ActiveBots, 9)
	if t.Summary.ActiveBots == 0 {
		activeBotsStr = "9 / 9"
	}

	// Restream metrics
	restreamRAM := getProcessRSSMb()
	if restreamRAM < 1 {
		restreamRAM = 33.0
	}
	restreamRAMStr := fmt.Sprintf("%.0f MB (Go Engine)", restreamRAM)

	videoOutStr := fmt.Sprintf("%dp10 • 1200k", frameH)
	if frameW > 1280 {
		videoOutStr = fmt.Sprintf("%dp10 • 2000k", frameH)
	}

	metrics := []struct {
		title   string
		val     string
		col     color.RGBA
		fillPct float64
	}{
		{"UPTIME DA VPS", uptimeStr, color.RGBA{148, 163, 184, 255}, 1.0},
		{"RAM DA VPS", ramStr, color.RGBA{56, 189, 248, 255}, float64(t.VPS.RAMPercent) / 100.0},
		{"RAM RESTREAM", restreamRAMStr, color.RGBA{83, 252, 24, 255}, restreamRAM / 100.0},
		{"SAÍDA DE VÍDEO", videoOutStr, color.RGBA{251, 146, 60, 255}, 1.0},
		{"BOTS ATIVOS", activeBotsStr, color.RGBA{34, 197, 94, 255}, 1.0},
	}

	for i, m := range metrics {
		sx := statStartX + float64(i)*(statW+statGap)
		dc.SetColor(color.RGBA{15, 23, 42, 230})
		dc.DrawRoundedRectangle(sx, sy, statW, statH, 7)
		dc.Fill()
		dc.SetColor(color.RGBA{m.col.R, m.col.G, m.col.B, 70})
		dc.SetLineWidth(1)
		dc.DrawRoundedRectangle(sx, sy, statW, statH, 7)
		dc.Stroke()

		dc.SetColor(color.RGBA{203, 213, 225, 255})
		dc.SetFontFace(mustFace(true, metricTitleSize))
		if is720p {
			dc.DrawString(m.title, sx+8, sy+14)
		} else {
			dc.DrawString(m.title, sx+12, sy+18)
		}

		dc.SetColor(m.col)
		dc.SetFontFace(mustFace(true, metricValSize))
		if is720p {
			dc.DrawString(m.val, sx+8, sy+31)
		} else {
			dc.DrawString(m.val, sx+12, sy+42)
		}

		barW := statW - 16
		if !is720p {
			barW = statW - 24
		}
		fillW := barW * m.fillPct
		if fillW > barW {
			fillW = barW
		}
		if fillW < 3 {
			fillW = 3
		}
		dc.SetColor(color.RGBA{m.col.R, m.col.G, m.col.B, 180})
		if is720p {
			dc.DrawRoundedRectangle(sx+8, sy+statH-4, fillW, 2.5, 1)
		} else {
			dc.DrawRoundedRectangle(sx+12, sy+statH-5, fillW, 3, 1)
		}
		dc.Fill()
	}

	// 3x3 BOT CARDS GRID
	gridX := 36.0
	gridY := 80.0
	gapX := 18.0
	gapY := 14.0
	cardW := (W - 2*gridX - 2*gapX) / 3.0
	cardH := 300.0

	if is720p {
		gridX = 16.0
		gridY = 54.0
		gapX = 8.0
		gapY = 6.0
		cardW = (W - 2*gridX - 2*gapX) / 3.0      // ~410px
		cardH = (H - gridY - 36.0 - 2*gapY) / 3.0 // ~206px
	}

	cards := buildCardList(t)

	for i, c := range cards {
		col := i % 3
		row := i / 3
		cx := gridX + float64(col)*(cardW+gapX)
		cy := gridY + float64(row)*(cardH+gapY)
		themeCol := parseHex(c.ThemeHex)

		// Card Container Background
		dc.SetColor(color.RGBA{13, 19, 34, 250})
		rCorner := 14.0
		if is720p {
			rCorner = 8.0
		}
		dc.DrawRoundedRectangle(cx, cy, cardW, cardH, rCorner)
		dc.Fill()

		// Outer Card Border
		dc.SetColor(color.RGBA{themeCol.R, themeCol.G, themeCol.B, 130})
		dc.SetLineWidth(1.2)
		dc.DrawRoundedRectangle(cx, cy, cardW, cardH, rCorner)
		dc.Stroke()

		// Top Theme Accent
		accentH := 4.0
		if is720p {
			accentH = 3.5
		}
		dc.SetColor(themeCol)
		dc.DrawRoundedRectangle(cx+4, cy, cardW-8, accentH, 2)
		dc.Fill()

		// Header Bar
		headH := 38.0
		headY := cy + 10.0
		headPadX := 10.0
		iconSize := 24.0
		titleX := cx + headPadX + 8.0
		if is720p {
			headH = 28.0
			headY = cy + 5.0
			headPadX = 6.0
			iconSize = 20.0
		}
		dc.SetColor(color.RGBA{themeCol.R / 5, themeCol.G / 5, themeCol.B / 5, 180})
		dc.DrawRoundedRectangle(cx+headPadX, headY, cardW-2*headPadX, headH, 6)
		dc.Fill()

		// Game Icon (Sprite / Badge)
		iconMu.RLock()
		if icon, ok := botIcons[c.IconKey]; ok && icon != nil {
			iconX := cx + headPadX + 5.0
			iconY := headY + (headH-iconSize)/2.0
			drawScaledIcon(dc, icon, iconX, iconY, iconSize, iconSize)
			titleX = iconX + iconSize + 6.0
		}
		iconMu.RUnlock()

		// Bot Title
		titleSize := 18.0
		titleY := cy + 34.0
		if is720p {
			titleSize = 13.0
			titleY = cy + 23.5
		}
		dc.SetColor(color.RGBA{255, 255, 255, 255})
		dc.SetFontFace(mustFace(true, titleSize))
		dc.DrawString(c.Title, titleX, titleY)
		titleW, _ := dc.MeasureString(c.Title)

		// Port Pill
		portStr := fmt.Sprintf("PORTA %d • %s", c.Port, c.Note)
		portFontSize := 10.0
		portH := 22.0
		portPillY := cy + 16.0
		portPadY := 32.0
		if is720p {
			portFontSize = 8.0
			portH = 17.0
			portPillY = cy + 9.0
			portPadY = 21.5
		}
		dc.SetFontFace(mustFace(true, portFontSize))
		pw, _ := dc.MeasureString(portStr)
		portX := titleX + titleW + 8.0
		dc.SetColor(color.RGBA{themeCol.R / 5, themeCol.G / 5, themeCol.B / 5, 220})
		dc.DrawRoundedRectangle(portX, portPillY, pw+10, portH, 4)
		dc.Fill()
		dc.SetColor(color.RGBA{themeCol.R, themeCol.G, themeCol.B, 180})
		dc.SetLineWidth(1)
		dc.DrawRoundedRectangle(portX, portPillY, pw+10, portH, 4)
		dc.Stroke()
		dc.SetColor(color.RGBA{themeCol.R, themeCol.G, themeCol.B, 255})
		dc.DrawString(portStr, portX+5, cy+portPadY)

		// Badge on Header Right
		badgeFontSize := 11.0
		badgeH := 24.0
		badgePillY := cy + 16.0
		badgePadY := 33.0
		if is720p {
			badgeFontSize = 9.0
			badgeH = 18.0
			badgePillY = cy + 9.0
			badgePadY = 22.0
		}
		dc.SetFontFace(mustFace(true, badgeFontSize))
		tw, _ := dc.MeasureString(c.Badge)
		badgeX := cx + cardW - headPadX - 8 - tw - 10
		dc.SetColor(color.RGBA{themeCol.R / 4, themeCol.G / 4, themeCol.B / 4, 220})
		dc.DrawRoundedRectangle(badgeX, badgePillY, tw+10, badgeH, 4)
		dc.Fill()
		dc.SetColor(themeCol)
		dc.SetLineWidth(1)
		dc.DrawRoundedRectangle(badgeX, badgePillY, tw+10, badgeH, 4)
		dc.Stroke()
		dc.DrawString(c.Badge, badgeX+5, cy+badgePadY)

		// Rows (Sub-Cards)
		rowStartY := cy + 56.0
		if is720p {
			rowStartY = cy + 37.0
		}
		numRows := len(c.Rows)
		if numRows > 4 {
			numRows = 4
		}
		if numRows == 0 {
			continue
		}

		availableH := cardH - 66.0
		if is720p {
			availableH = cardH - 42.0
		}
		rHeight := availableH / float64(numRows)

		var rowLabelSize, rowTagSize, statFontSize, barTextSize, line1YOffset, line2YOffset float64
		if is720p {
			if numRows <= 2 {
				rowLabelSize = 13.0
				rowTagSize = 10.5
				statFontSize = 11.0
				barTextSize = 8.5
				line1YOffset = 20.0
				line2YOffset = 44.0
			} else if numRows == 3 {
				rowLabelSize = 12.0
				rowTagSize = 10.0
				statFontSize = 10.5
				barTextSize = 8.0
				line1YOffset = 16.0
				line2YOffset = 33.0
			} else { // 4 rows
				rowLabelSize = 11.5
				rowTagSize = 9.5
				statFontSize = 9.5
				barTextSize = 7.5
				line1YOffset = 13.0
				line2YOffset = 26.5
			}
		} else {
			rowLabelSize = 14.0
			rowTagSize = 11.5
			statFontSize = 11.5
			barTextSize = 9.5
			line1YOffset = 18.0
			line2YOffset = 36.0
		}

		for rIdx, r := range c.Rows[:numRows] {
			ry := rowStartY + float64(rIdx)*rHeight

			// Row container
			dc.SetColor(color.RGBA{18, 26, 44, 235})
			dc.DrawRoundedRectangle(cx+headPadX, ry+2, cardW-2*headPadX, rHeight-4, 5)
			dc.Fill()
			dc.SetColor(color.RGBA{themeCol.R / 3, themeCol.G / 3, themeCol.B / 3, 110})
			dc.SetLineWidth(1)
			dc.DrawRoundedRectangle(cx+headPadX, ry+2, cardW-2*headPadX, rHeight-4, 5)
			dc.Stroke()

			// Neon Accent Vertical Stripe on left
			dc.SetColor(themeCol)
			dc.DrawRoundedRectangle(cx+headPadX+2, ry+4, 3.5, rHeight-8, 1.5)
			dc.Fill()

			// Line 1: Label
			dc.SetColor(color.RGBA{255, 255, 255, 255})
			dc.SetFontFace(mustFace(true, rowLabelSize))
			dc.DrawString(r.Label, cx+headPadX+12, ry+line1YOffset)

			// Tag (Level)
			if r.Tag != "" {
				dc.SetFontFace(mustFace(true, rowTagSize))
				tagW, _ := dc.MeasureString(r.Tag)
				tagX := cx + cardW - headPadX - 8 - tagW - 8
				tagH := 18.0
				tagPillY := ry + 3.0
				if is720p {
					if numRows <= 2 {
						tagH = 18.0
						tagPillY = ry + 8.0
					} else if numRows == 3 {
						tagH = 16.0
						tagPillY = ry + 4.0
					} else {
						tagH = 14.0
						tagPillY = ry + 2.0
					}
				}
				dc.SetColor(color.RGBA{themeCol.R / 4, themeCol.G / 4, themeCol.B / 4, 230})
				dc.DrawRoundedRectangle(tagX, tagPillY, tagW+8, tagH, 3)
				dc.Fill()
				dc.SetColor(themeCol)
				dc.SetLineWidth(1)
				dc.DrawRoundedRectangle(tagX, tagPillY, tagW+8, tagH, 3)
				dc.Stroke()
				dc.DrawString(r.Tag, tagX+4, ry+line1YOffset)
			}

			// Line 2: Stats
			curX := cx + headPadX + 12.0
			dc.SetFontFace(mustFace(true, statFontSize))

			statGapX := 12.0
			if is720p {
				statGapX = 8.0
			}

			if r.Gold != "" {
				dc.SetColor(color.RGBA{250, 204, 21, 255})
				txt := "GP: " + r.Gold
				dc.DrawString(txt, curX, ry+line2YOffset)
				gw, _ := dc.MeasureString(txt)
				curX += gw + statGapX
			}

			if r.XP != "" {
				dc.SetColor(color.RGBA{192, 132, 252, 255})
				txt := "XP: " + r.XP
				dc.DrawString(txt, curX, ry+line2YOffset)
				xw, _ := dc.MeasureString(txt)
				curX += xw + statGapX
			}

			if r.Kills != "" {
				dc.SetColor(color.RGBA{248, 113, 113, 255})
				txt := "KILLS: " + r.Kills
				dc.DrawString(txt, curX, ry+line2YOffset)
				kw, _ := dc.MeasureString(txt)
				curX += kw + statGapX
			}

			if r.Area != "" {
				dc.SetColor(color.RGBA{52, 211, 153, 255})
				dc.DrawString(r.Area, curX, ry+line2YOffset)
				aw, _ := dc.MeasureString(r.Area)
				curX += aw + statGapX
			}

			if r.Extra != "" {
				dc.SetColor(color.RGBA{56, 189, 248, 255})
				dc.DrawString(r.Extra, curX, ry+line2YOffset)
			}

			// Dual Progress Bars (HP & Mana)
			if r.HP > 0 || r.Mana > 0 {
				barW := 90.0
				barH := 5.0
				barStartX := cx + cardW - 210.0
				barY := ry + rHeight - 10.0
				if is720p {
					barW = 60.0
					barH = 4.5
					barStartX = cx + cardW - 142.0
					barY = ry + rHeight - 7.5
				}

				if r.HP > 0 {
					dc.SetColor(color.RGBA{34, 197, 94, 255})
					dc.SetFontFace(mustFace(true, barTextSize))
					dc.DrawString(fmt.Sprintf("HP %.0f%%", r.HP*100), barStartX, barY-2)

					dc.SetColor(color.RGBA{255, 255, 255, 30})
					dc.DrawRoundedRectangle(barStartX, barY, barW, barH, 2)
					dc.Fill()

					hpFill := barW * r.HP
					if hpFill > barW {
						hpFill = barW
					}
					if hpFill > 2 {
						dc.SetColor(color.RGBA{34, 197, 94, 255})
						dc.DrawRoundedRectangle(barStartX, barY, hpFill, barH, 2)
						dc.Fill()
					}
				}

				if r.Mana > 0 {
					gap := 12.0
					if is720p {
						gap = 8.0
					}
					manaX := barStartX + barW + gap
					dc.SetColor(color.RGBA{59, 130, 246, 255})
					dc.SetFontFace(mustFace(true, barTextSize))
					dc.DrawString(fmt.Sprintf("MP %.0f%%", r.Mana*100), manaX, barY-2)

					dc.SetColor(color.RGBA{255, 255, 255, 30})
					dc.DrawRoundedRectangle(manaX, barY, barW, barH, 2)
					dc.Fill()

					manaFill := barW * r.Mana
					if manaFill > barW {
						manaFill = barW
					}
					if manaFill > 2 {
						dc.SetColor(color.RGBA{59, 130, 246, 255})
						dc.DrawRoundedRectangle(manaX, barY, manaFill, barH, 2)
						dc.Fill()
					}
				}
			}
		}
	}

	// FOOTER TICKER BACKGROUND
	tickerH := 52.0
	badgeW := 136.0
	badgeH := 28.0
	badgeFontSize := 12.0
	badgeTextYOffset := 18.0
	if is720p {
		tickerH = 36.0
		badgeW = 115.0
		badgeH = 24.0
		badgeFontSize = 9.5
		badgeTextYOffset = 16.5
	}
	tickerY := H - tickerH

	dc.SetColor(color.RGBA{11, 16, 29, 255})
	dc.DrawRectangle(0, tickerY, W, tickerH)
	dc.Fill()

	// Cyan separator line
	dc.SetColor(color.RGBA{0, 240, 255, 120})
	dc.DrawRectangle(0, tickerY, W, 2)
	dc.Fill()

	// Live Events Badge
	badgeY := tickerY + (tickerH-badgeH)/2.0
	dc.SetColor(color.RGBA{239, 68, 68, 255})
	dc.DrawRoundedRectangle(16, badgeY, badgeW, badgeH, 5)
	dc.Fill()
	dc.SetColor(color.RGBA{255, 255, 255, 255})
	dc.DrawCircle(26, badgeY+badgeH/2.0, 3.5)
	dc.Fill()
	dc.SetFontFace(mustFace(true, badgeFontSize))
	dc.DrawString("LIVE CLUSTER", 34, badgeY+badgeTextYOffset)
}

func drawDynamicOverlays(dc *gg.Context, pulse float64) {
	W := float64(frameW)
	H := float64(frameH)
	is720p := (frameW <= 1280)

	liveX := W - 180.0
	liveY := 41.0
	liveDotR := 7.0
	liveFontSize := 17.0
	clockFontSize := 15.0
	clockOffX := 95.0
	clearW := 200.0
	clearH := 40.0
	clearY := liveY - 20.0

	tickerH := 52.0
	badgeW := 136.0
	marqueeFontSize := 14.0
	marqueeYOffset := 31.0

	if is720p {
		liveX = W - 146.0
		liveY = 25.0
		liveDotR = 5.0
		liveFontSize = 12.0
		clockFontSize = 11.5
		clockOffX = 70.0
		clearW = 150.0
		clearH = 36.0
		clearY = 6.0

		tickerH = 36.0
		badgeW = 115.0
		marqueeFontSize = 11.0
		marqueeYOffset = 23.5
	}

	tickerY := H - tickerH

	// Clear top-right status area in-place
	dc.SetColor(color.RGBA{7, 9, 14, 255})
	dc.DrawRectangle(liveX-20, clearY, clearW, clearH)
	dc.Fill()

	// Pulsing Live Indicator with Radar Halo
	haloAlpha := uint8(60 + 50*math.Sin(pulse*3.5))
	dc.SetColor(color.RGBA{34, 197, 94, haloAlpha})
	dc.DrawCircle(liveX, liveY, liveDotR+3.5)
	dc.Fill()

	pulsingAlpha := uint8(200 + 55*math.Sin(pulse*3.5))
	dc.SetColor(color.RGBA{34, 197, 94, pulsingAlpha})
	dc.DrawCircle(liveX, liveY, liveDotR)
	dc.Fill()

	dc.SetColor(color.RGBA{248, 250, 252, 255})
	dc.SetFontFace(mustFace(true, liveFontSize))
	dc.DrawString("AO VIVO", liveX+liveDotR*2+4, liveY+liveFontSize*0.35)

	// Clock
	clockStr := time.Now().Format("15:04:05")
	dc.SetColor(color.RGBA{148, 163, 184, 255})
	dc.SetFontFace(mustFace(true, clockFontSize))
	dc.DrawString(clockStr, liveX+clockOffX, liveY+clockFontSize*0.35)

	// Clear footer marquee area in-place
	marqueeStartX := 16.0 + badgeW + 12.0
	dc.SetColor(color.RGBA{11, 16, 29, 255})
	dc.DrawRectangle(marqueeStartX, tickerY+2, W-marqueeStartX, tickerH-2)
	dc.Fill()

	// Scrolling Marquee
	msg := "9 BOTS NATIVOS EM GO (ZERO NODE.JS)   •   HUNTERA • TIBIDLE • POKEMON SIGMA • LEVELING IDLE 2D • BAIAK IDLE • POKEIDLE.IO • POKEIDLE WORLD • KICK MINER • TWITCH MINER   •   FARM AUTOMATICO 24/7   •   CLUSTER ESTAVEL 24/7   •   LOW-RAM ENGINE (<100MB RAM)"
	msg = msg + "   •   "
	dc.Push()
	dc.DrawRectangle(marqueeStartX, tickerY, W-marqueeStartX-10, tickerH)
	dc.Clip()
	dc.SetColor(color.RGBA{226, 232, 240, 255})
	dc.SetFontFace(mustFace(true, marqueeFontSize))
	tw, _ := dc.MeasureString(msg)
	scrollSpeed := 65.0
	if is720p {
		scrollSpeed = 50.0
	}
	offX := mathMod(pulse*scrollSpeed+tw, tw+W-marqueeStartX-10)
	dc.DrawString(msg, W-offX, tickerY+marqueeYOffset)
	dc.DrawString(msg, W-offX+tw, tickerY+marqueeYOffset)
	dc.Pop()
	dc.ResetClip()
}

func mathMod(a, b float64) float64 {
	return math.Mod(a, b)
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

const (
	defaultKickIngest   = "rtmps://fa723fc1b171.global-contribute.live-video.net:443/app/SUA_CHAVE_KICK"
	defaultTwitchIngest = "rtmp://live.twitch.tv/app/SUA_CHAVE_TWITCH"
	defaultYTIngest     = "rtmp://a.rtmp.youtube.com/live2/SUA_CHAVE_YOUTUBE"
	defaultRadioURL     = "https://radio.plaza.one/mp3"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func streamFrames(w io.Writer, fps int, telemURL, localSample string) error {
	staticDC := gg.NewContext(frameW, frameH)
	dc := gg.NewContext(frameW, frameH)

	fetchTelemetry(telemURL, localSample)

	dataMu.RLock()
	t := currentTelem
	dataMu.RUnlock()

	drawStaticBackground(staticDC, t)

	var telemDirty atomic.Bool

	stopChan := make(chan struct{})
	defer close(stopChan)

	// Periodic telemetry updater goroutine: fetches network data only, zero drawing
	go func() {
		ticker := time.NewTicker(fetchPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				fetchTelemetry(telemURL, localSample)
				telemDirty.Store(true)
			}
		}
	}()

	interval := time.Second / time.Duration(fps)
	frameTick := time.NewTicker(interval)
	defer frameTick.Stop()

	staticPix := staticDC.Image().(*image.RGBA).Pix
	dynPix := dc.Image().(*image.RGBA).Pix

	start := time.Now()
	for range frameTick.C {
		pulse := time.Since(start).Seconds()

		// Redraw background sequentially in THIS goroutine if new data arrived
		if telemDirty.Swap(false) {
			dataMu.RLock()
			cur := currentTelem
			dataMu.RUnlock()

			staticDC.ResetClip()
			drawStaticBackground(staticDC, cur)
		}

		copy(dynPix, staticPix)

		dc.ResetClip()
		drawDynamicOverlays(dc, pulse)
		dc.ResetClip()

		// Zero-copy: send RGBA raw buffer directly to stdout / stdin pipe
		if _, err := w.Write(dynPix); err != nil {
			return err
		}
	}
	return nil
}

func startLiveStreamLoop(fps int, kickURL, twitchURL, ytURL, radioURL string, telemURL, localSample string) {
	fmt.Println("========================================================================")
	fmt.Println(" ⚡ INICIANDO RESTREAM GO NATIVO (ULTRA-LOW RAM ENGINE <100MB) ⚡")
	fmt.Printf(" Destinos: Kick + Twitch + YouTube | Áudio: Nightwave Plaza Lo-Fi\n")
	fmt.Printf(" Resolução: %dx%d @ %d FPS | FFmpeg Stdin Pipe Direct na RAM\n", frameW, frameH, fps)
	fmt.Println("========================================================================")

	teeDest := fmt.Sprintf("[f=flv:onfail=ignore:flvflags=no_duration_filesize]%s|[f=flv:onfail=ignore:flvflags=no_duration_filesize]%s|[f=flv:onfail=ignore:flvflags=no_duration_filesize]%s",
		kickURL, twitchURL, ytURL)

	bitrate := "1200k"
	bufsize := "2400k"
	queueSize := "32"
	if frameW > 1280 {
		bitrate = "2000k"
		bufsize = "4000k"
		queueSize = "64"
	}

	for {
		log.Printf("[stream] Disparando processo FFmpeg interno (Low-RAM: %s, queue: %s)...", bitrate, queueSize)

		cmd := exec.Command("ffmpeg",
			"-hide_banner", "-loglevel", "warning",
			"-thread_queue_size", queueSize,
			"-f", "rawvideo",
			"-pix_fmt", "rgba",
			"-video_size", fmt.Sprintf("%dx%d", frameW, frameH),
			"-framerate", fmt.Sprintf("%d", fps),
			"-i", "pipe:0",
			"-thread_queue_size", queueSize,
			"-probesize", "32k",
			"-analyzeduration", "0",
			"-reconnect", "1",
			"-reconnect_at_eof", "1",
			"-reconnect_streamed", "1",
			"-reconnect_delay_max", "5",
			"-i", radioURL,
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-tune", "zerolatency",
			"-threads", "1",
			"-b:v", bitrate,
			"-maxrate", bitrate,
			"-bufsize", bufsize,
			"-pix_fmt", "yuv420p",
			"-g", fmt.Sprintf("%d", fps*2),
			"-keyint_min", fmt.Sprintf("%d", fps),
			"-sc_threshold", "0",
			"-c:a", "aac",
			"-b:a", "128k",
			"-ar", "44100",
			"-af", "aresample=async=1000,volume=0.8",
			"-flags", "+global_header",
			"-f", "tee",
			"-map", "0:v",
			"-map", "1:a",
			teeDest,
		)

		cmd.Stderr = os.Stderr
		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Printf("[stream] Erro ao criar pipe stdin: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}

		if err := cmd.Start(); err != nil {
			log.Printf("[stream] Erro ao iniciar subprocesso ffmpeg: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}

		err = streamFrames(stdin, fps, telemURL, localSample)
		if err != nil {
			log.Printf("[stream] Pipe de frames finalizado: %v", err)
		}

		_ = stdin.Close()
		_ = cmd.Wait()
		log.Println("[stream] FFmpeg encerrou. Reconectando em 2 segundos...")
		time.Sleep(2 * time.Second)
	}
}

func main() {
	resFlag := flag.String("res", "720p", "resolucao da stream: 720p (ultra-low ram <100MB) ou 1080p")
	pngOut := flag.String("png", "", "gera um unico frame PNG e sai (teste)")
	vidOut := flag.String("o", "", "escreve frames rawvideo rgba (use - para stdout)")
	fps := flag.Int("fps", 10, "taxa de quadros (padrao 10 FPS ultra-leve)")
	kickURL := flag.String("kick", getEnv("KICK_INGEST", defaultKickIngest), "URL RTMP Kick")
	twitchURL := flag.String("twitch", getEnv("TWITCH_INGEST", defaultTwitchIngest), "URL RTMP Twitch")
	ytURL := flag.String("yt", getEnv("YOUTUBE_INGEST", defaultYTIngest), "URL RTMP YouTube")
	radioURL := flag.String("radio", getEnv("RADIO_URL", defaultRadioURL), "Stream URL Audio Lo-Fi")
	flag.Parse()

	if *resFlag == "1080p" {
		frameW = 1920
		frameH = 1080
	} else {
		frameW = 1280
		frameH = 720
	}

	url := "https://kbelludoo.github.io/bot-template-go/status.json"
	localPath := filepath.Join("hud", "status_sample.json")
	if _, err := os.Stat("cmd/hud/status_sample.json"); err == nil {
		localPath = "cmd/hud/status_sample.json"
	}

	if *pngOut != "" {
		fetchTelemetry(url, localPath)
		dc := gg.NewContext(frameW, frameH)
		dataMu.RLock()
		t := currentTelem
		dataMu.RUnlock()
		drawStaticBackground(dc, t)
		drawDynamicOverlays(dc, 0)
		if err := writePNG(*pngOut, dc.Image()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "[hud] frame salvo em %s (%dx%d)\n", *pngOut, frameW, frameH)
		return
	}

	if *vidOut != "" {
		var w io.Writer
		if *vidOut == "-" {
			w = os.Stdout
		} else {
			f, err := os.Create(*vidOut)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			defer f.Close()
			w = f
		}
		_ = streamFrames(w, *fps, url, localPath)
		return
	}

	// Modo Transmissao Ao Vivo Integrado por Padrao!
	startLiveStreamLoop(*fps, *kickURL, *twitchURL, *ytURL, *radioURL, url, localPath)
}
