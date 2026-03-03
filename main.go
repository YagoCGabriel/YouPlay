package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/olekukonko/tablewriter"
)

// ========================
// PATHS
// ========================

var (
	baseDir    = execDir()
	dataDir    = filepath.Join(baseDir, "data")
	configFile = filepath.Join(dataDir, "config.json")
)

func execDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// ========================
// CONSTANTES
// ========================

var qualityOptions = []string{
	"🏆 Melhor disponível",
	"🖥  1080p",
	"📺 720p",
	"📱 480p",
	"🎵 Apenas áudio",
}

var qualityFormats = map[string]string{
	"🏆 Melhor disponível": "bestvideo+bestaudio/best",
	"🖥  1080p":            "bestvideo[height<=1080][vcodec^=avc][ext=mp4]+bestaudio[ext=m4a]/bestvideo[height<=1080][vcodec^=avc]+bestaudio/bestvideo[height<=1080]+bestaudio",
	"📺 720p":             "bestvideo[height<=720][vcodec^=avc][ext=mp4]+bestaudio[ext=m4a]/bestvideo[height<=720][vcodec^=avc]+bestaudio/bestvideo[height<=720]+bestaudio",
	"📱 480p":             "bestvideo[height<=480]+bestaudio/best[height<=480]",
	"🎵 Apenas áudio":     "bestaudio/best",
}

var playersNoMerge = map[string]bool{
	"vlc":         true,
	"kmplayer":    true,
	"potplayer":   true,
	"potplayer64": true,
}

var mpvRenderFlags = []string{
	// — Renderização —
	"--hwdec=auto-safe",
	"--vo=gpu",
	"--gpu-api=d3d11",
	"--profile=gpu-hq",
	"--scale=lanczos",
	"--video-sync=display-resample",
	// — Buffer / Cache —
	"--cache=yes",
	"--cache-secs=120",
	"--demuxer-max-bytes=150MiB",
	"--demuxer-max-back-bytes=75MiB",
	"--demuxer-readahead-secs=20",
	"--network-timeout=30",
}

// ========================
// ESTILOS
// ========================

var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("86")).
			Padding(0, 2).
			Bold(true).
			Foreground(lipgloss.Color("86"))

	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// ========================
// CONFIG
// ========================

type Config struct {
	Player      string `json:"player"`
	DefaultMode string `json:"default_mode"`
}

var defaultConfig = Config{
	Player:      "mpv",
	DefaultMode: "manual",
}

func loadConfig() Config {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return defaultConfig
	}

	f, err := os.Open(configFile)
	if err != nil {
		saveConfig(defaultConfig)
		return defaultConfig
	}
	defer f.Close()

	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return defaultConfig
	}

	if cfg.Player == "" {
		cfg.Player = defaultConfig.Player
	}
	if cfg.DefaultMode == "" {
		cfg.DefaultMode = defaultConfig.DefaultMode
	}

	return cfg
}

func saveConfig(cfg Config) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	f, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "    ")
	return enc.Encode(cfg)
}

// ========================
// UI UTIL
// ========================

func clearScreen() {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

func pause() {
	fmt.Println(dimStyle.Render("\nPressione ENTER para continuar..."))
	fmt.Scanln()
}

func loading(text string, fn func()) {
	done := make(chan struct{})
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0

	go func() {
		for {
			select {
			case <-done:
				fmt.Print("\r\033[K")
				return
			default:
				fmt.Printf("\r%s %s", successStyle.Render(frames[i%len(frames)]), text)
				time.Sleep(80 * time.Millisecond)
				i++
			}
		}
	}()

	fn()
	close(done)
	time.Sleep(90 * time.Millisecond)
}

func printPanel(title, subtitle string) {
	content := fmt.Sprintf("%s\n%s", title, subtitle)
	fmt.Println(panelStyle.Render(content))
}

// ========================
// PLAYER
// ========================

func isPlayerAvailable(player string) bool {
	_, err := exec.LookPath(player)
	return err == nil
}

type StreamInfo struct {
	URL              string         `json:"url"`
	RequestedFormats []StreamFormat `json:"requested_formats"`
}

type StreamFormat struct {
	URL string `json:"url"`
}

func getStreamURLs(videoURL, qualityFormat string) ([]string, error) {
	cmd := exec.Command("yt-dlp",
		"--dump-json",
		"--quiet",
		"-f", qualityFormat,
		videoURL,
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp erro: %w", err)
	}

	var info StreamInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("erro ao parsear JSON: %w", err)
	}

	if len(info.RequestedFormats) > 0 {
		urls := make([]string, 0, len(info.RequestedFormats))
		for _, f := range info.RequestedFormats {
			if f.URL != "" {
				urls = append(urls, f.URL)
			}
		}
		return urls, nil
	}

	if info.URL == "" {
		return nil, fmt.Errorf("URL não encontrada na resposta do yt-dlp")
	}

	return []string{info.URL}, nil
}

func openPlayer(player, url, qualityFormat string) {
	clearScreen()

	if !isPlayerAvailable(player) {
		fmt.Println(errorStyle.Render(fmt.Sprintf("❌ %s não está instalado ou não está no PATH.", player)))
		pause()
		return
	}

	fmt.Println(successStyle.Render(fmt.Sprintf("▶ Abrindo no %s...", player)))

	var cmd *exec.Cmd

	switch {
	case player == "mpv":
		args := []string{
			url,
			fmt.Sprintf("--ytdl-format=%s", qualityFormat),
			"--ytdl-raw-options=concurrent-fragments=4,retries=10,fragment-retries=10",
		}
		args = append(args, mpvRenderFlags...)
		cmd = exec.Command(player, args...)

	case playersNoMerge[player]:
		var streamURLs []string
		var resolveErr error

		loading("Resolvendo stream...", func() {
			streamURLs, resolveErr = getStreamURLs(url, qualityFormat)
		})

		if resolveErr != nil {
			fmt.Println(errorStyle.Render(fmt.Sprintf("Erro ao resolver stream: %v", resolveErr)))
			pause()
			return
		}

		if len(streamURLs) == 2 {
			fmt.Println(warnStyle.Render("⚠ Este player não suporta merge de streams. Usando stream único (mp4)."))
			loading("Resolvendo stream mp4...", func() {
				streamURLs, resolveErr = getStreamURLs(url, "best[ext=mp4]/best")
			})
			if resolveErr != nil {
				fmt.Println(errorStyle.Render(fmt.Sprintf("Erro ao resolver stream mp4: %v", resolveErr)))
				pause()
				return
			}
		}

		cmd = exec.Command(player, streamURLs...)

	default:
		// Player desconhecido — resolve e tenta abrir
		var streamURLs []string
		var resolveErr error

		loading("Resolvendo stream...", func() {
			streamURLs, resolveErr = getStreamURLs(url, qualityFormat)
		})

		if resolveErr != nil {
			fmt.Println(errorStyle.Render(fmt.Sprintf("Erro ao resolver stream: %v", resolveErr)))
			pause()
			return
		}

		cmd = exec.Command(player, streamURLs...)
	}

	if err := cmd.Start(); err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("Erro ao abrir player: %v", err)))
		pause()
	}
}

// ========================
// SEARCH
// ========================

type VideoEntry struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Duration float64 `json:"duration"`
	URL      string  `json:"url"`
}

func (v VideoEntry) FullURL() string {
	if v.ID != "" {
		return fmt.Sprintf("https://www.youtube.com/watch?v=%s", v.ID)
	}
	if strings.HasPrefix(v.URL, "http") {
		return v.URL
	}
	return ""
}

func (v VideoEntry) DurationStr() string {
	if v.Duration == 0 {
		return "--:--"
	}
	mins := int(v.Duration) / 60
	secs := int(v.Duration) % 60
	return fmt.Sprintf("%d:%02d", mins, secs)
}

func searchVideos(query string) ([]VideoEntry, error) {
	cmd := exec.Command("yt-dlp",
		"--flat-playlist",
		"--print-json",
		"--quiet",
		fmt.Sprintf("ytsearch10:%s", query),
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp erro: %w", err)
	}

	var entries []VideoEntry
	scanner := bufio.NewScanner(strings.NewReader(string(out)))

	buf := make([]byte, 0, 512*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry VideoEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func chooseVideo(videos []VideoEntry) string {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"#", "Título", "Duração"})
	table.SetBorder(true)
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.FgCyanColor, tablewriter.Bold},
		tablewriter.Colors{tablewriter.FgWhiteColor, tablewriter.Bold},
		tablewriter.Colors{tablewriter.FgMagentaColor, tablewriter.Bold},
	)
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.FgCyanColor},
		tablewriter.Colors{tablewriter.FgWhiteColor},
		tablewriter.Colors{tablewriter.FgMagentaColor},
	)
	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_CENTER,
		tablewriter.ALIGN_LEFT,
		tablewriter.ALIGN_CENTER,
	})

	for i, v := range videos {
		title := v.Title
		if len(title) > 60 {
			title = title[:57] + "..."
		}
		table.Append([]string{
			strconv.Itoa(i + 1),
			title,
			v.DurationStr(),
		})
	}

	table.Render()

	var choice string
	prompt := &survey.Input{Message: "Escolha o número do vídeo:"}
	if err := survey.AskOne(prompt, &choice); err != nil || choice == "" {
		return ""
	}

	index, err := strconv.Atoi(strings.TrimSpace(choice))
	if err != nil || index < 1 || index > len(videos) {
		fmt.Println(errorStyle.Render("Número inválido."))
		return ""
	}

	return videos[index-1].FullURL()
}

func chooseQuality() string {
	var choice string
	prompt := &survey.Select{
		Message: "Escolha a qualidade:",
		Options: qualityOptions,
	}
	if err := survey.AskOne(prompt, &choice); err != nil || choice == "" {
		return qualityFormats["🏆 Melhor disponível"]
	}
	return qualityFormats[choice]
}

// ========================
// MODOS
// ========================

func manualMode(cfg Config) {
	clearScreen()

	var query string
	prompt := &survey.Input{Message: "Digite o nome do vídeo:"}
	if err := survey.AskOne(prompt, &query); err != nil || query == "" {
		return
	}

	var videos []VideoEntry
	var searchErr error

	clearScreen()
	loading("Buscando vídeos...", func() {
		videos, searchErr = searchVideos(query)
	})

	if searchErr != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("Erro na busca: %v", searchErr)))
		pause()
		return
	}

	if len(videos) == 0 {
		fmt.Println(errorStyle.Render("Nenhum resultado encontrado."))
		pause()
		return
	}

	fmt.Println()
	url := chooseVideo(videos)
	if url == "" {
		return
	}

	quality := chooseQuality()
	openPlayer(cfg.Player, url, quality)
}

func autoMode(cfg Config) {
	clearScreen()

	var query string
	prompt := &survey.Input{Message: "Digite o nome do vídeo:"}
	if err := survey.AskOne(prompt, &query); err != nil || query == "" {
		return
	}

	var videos []VideoEntry
	var searchErr error

	clearScreen()
	loading("Buscando melhor resultado...", func() {
		videos, searchErr = searchVideos(query)
	})

	if searchErr != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("Erro na busca: %v", searchErr)))
		pause()
		return
	}

	if len(videos) == 0 {
		fmt.Println(errorStyle.Render("Nenhum resultado encontrado."))
		pause()
		return
	}

	first := videos[0]
	url := first.FullURL()

	if url == "" {
		fmt.Println(errorStyle.Render("Não foi possível obter a URL do vídeo."))
		pause()
		return
	}

	quality := chooseQuality()
	fmt.Println(successStyle.Render(fmt.Sprintf("▶ Abrindo: %s", first.Title)))
	openPlayer(cfg.Player, url, quality)
}

// ========================
// CONFIG MENU
// ========================

func configMenu(cfg *Config) {
	for {
		clearScreen()

		var choice string
		prompt := &survey.Select{
			Message: "Configurações",
			Options: []string{"Selecionar Player", "Voltar"},
		}
		if err := survey.AskOne(prompt, &choice); err != nil {
			return
		}

		switch choice {
		case "Selecionar Player":
			var player string
			playerPrompt := &survey.Select{
				Message: "Escolha o player:",
				Options: []string{
					"mpv", "vlc", "kmplayer",
					"potplayer", "potplayer64", "outro",
				},
			}
			if err := survey.AskOne(playerPrompt, &player); err != nil {
				continue
			}

			if player == "outro" {
				customPrompt := &survey.Input{
					Message: "Digite o nome ou caminho completo do executável:",
				}
				if err := survey.AskOne(customPrompt, &player); err != nil || player == "" {
					continue
				}
			}

			cfg.Player = player
			if err := saveConfig(*cfg); err != nil {
				fmt.Println(errorStyle.Render(fmt.Sprintf("Erro ao salvar config: %v", err)))
			} else {
				fmt.Println(successStyle.Render(fmt.Sprintf("Player definido como: %s", player)))
			}
			pause()

		case "Voltar":
			return
		}
	}
}

// ========================
// MAIN
// ========================

func main() {
	cfg := loadConfig()

	for {
		clearScreen()

		printPanel(
			"YouPlay v2.0",
			fmt.Sprintf("Player: %s", cfg.Player),
		)

		fmt.Println()

		var choice string
		prompt := &survey.Select{
			Message: "Menu Principal",
			Options: []string{
				"🔎 Pesquisa (Manual)",
				"⚡ Modo AUTO",
				"⚙ Configurações",
				"❌ Sair",
			},
		}
		if err := survey.AskOne(prompt, &choice); err != nil {
			break
		}

		switch {
		case strings.HasPrefix(choice, "🔎"):
			manualMode(cfg)
		case strings.HasPrefix(choice, "⚡"):
			autoMode(cfg)
		case strings.HasPrefix(choice, "⚙"):
			configMenu(&cfg)
		case strings.HasPrefix(choice, "❌"):
			clearScreen()
			return
		}
	}
}
