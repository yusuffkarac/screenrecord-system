package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type ScreenData struct {
	Type      string `json:"type"`
	Image     string `json:"image"`
	Timestamp int64  `json:"timestamp"`
	ClientID  string `json:"clientId"`
}

type BlockedApp struct {
	Name           string   `json:"name"`
	Processes      []string `json:"processes"`
	WarningMessage string   `json:"warning_message"`
}

type BlockedWebsite struct {
	Name           string   `json:"name"`
	URLs           []string `json:"urls"`
	WarningMessage string   `json:"warning_message"`
}

type WebsiteBlockerConfig struct {
	BlockedWebsites []BlockedWebsite `json:"blocked_websites"`
	Settings        struct {
		WebsiteBlockerEnabled bool   `json:"website_blocker_enabled"`
		BlockingMethod        string `json:"blocking_method"`
		BackupHosts           bool   `json:"backup_hosts"`
		ShowWarnings          bool   `json:"show_warnings"`
		RedirectTo            string `json:"redirect_to"`
		CheckIntervalSeconds  int    `json:"check_interval_seconds"`
		CloseBrowserTabs      bool   `json:"close_browser_tabs"`
		ShowBlockingMessage   bool   `json:"show_blocking_message"`
	} `json:"settings"`
}

type AppBlockerConfig struct {
	BlockedApplications []BlockedApp `json:"blocked_applications"`
	Settings            struct {
		CheckIntervalSeconds int  `json:"check_interval_seconds"`
		AutoKill             bool `json:"auto_kill"`
		ShowWarnings         bool `json:"show_warnings"`
		MaxWarnings          int  `json:"max_warnings"`
		AppBlockerEnabled    bool `json:"app_blocker_enabled"`
	} `json:"settings"`
}

type Client struct {
	conn            *websocket.Conn
	serverURL       string
	clientID        string
	isConnected     bool
	useHTTP         bool // HTTP POST kullan (WebSocket yerine)
	httpClient      *http.Client
	appBlocker      *AppBlockerConfig
	websiteBlocker  *WebsiteBlockerConfig
	warningCounts   map[string]int
	hostsBackupPath string
}

func generateClientID() string {
	clientIDFile := "client_id.txt"

	// Mevcut ID'yi dosyadan oku
	if data, err := ioutil.ReadFile(clientIDFile); err == nil {
		existingID := strings.TrimSpace(string(data))
		if existingID != "" {
			log.Printf("📄 Kaydedilmiş Client ID kullanılıyor: %s", existingID)
			return existingID
		}
	}

	// Yeni ID oluştur
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Basit unique ID
	timestamp := time.Now().Unix()
	newID := fmt.Sprintf("client_%s_%d", hostname, timestamp)

	// ID'yi dosyaya kaydet
	err = ioutil.WriteFile(clientIDFile, []byte(newID), 0644)
	if err != nil {
		log.Printf("⚠️ Client ID kaydedilemedi: %v", err)
	} else {
		log.Printf("💾 Yeni Client ID kaydedildi: %s", newID)
	}

	return newID
}

func NewClient(serverURL string) *Client {
	client := &Client{
		serverURL:       serverURL,
		clientID:        generateClientID(), // Kalıcı ID
		useHTTP:         true,               // HTTP POST kullan
		httpClient:      &http.Client{Timeout: 10 * time.Second},
		warningCounts:   make(map[string]int),
		hostsBackupPath: "/etc/hosts.backup",
	}

	// Konfigürasyonları yükle
	client.loadAppBlockerConfig()
	client.loadWebsiteBlockerConfig()

	return client
}

func (c *Client) loadAppBlockerConfig() {
	configFile := "blocked_apps.json"
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("⚠️ Uygulama engelleme config dosyası bulunamadı: %v", err)
		return
	}

	err = json.Unmarshal(data, &c.appBlocker)
	if err != nil {
		log.Printf("❌ Config dosyası parse edilemedi: %v", err)
		return
	}

	log.Printf("✅ %d uygulama engelleme listesine eklendi", len(c.appBlocker.BlockedApplications))
}

func (c *Client) loadWebsiteBlockerConfig() {
	configFile := "blocked_websites.json"
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("⚠️ Website engelleme config dosyası bulunamadı: %v", err)
		return
	}

	err = json.Unmarshal(data, &c.websiteBlocker)
	if err != nil {
		log.Printf("❌ Website config dosyası parse edilemedi: %v", err)
		return
	}

	log.Printf("✅ %d website engelleme listesine eklendi", len(c.websiteBlocker.BlockedWebsites))
}

func (c *Client) Connect() error {
	if c.useHTTP {
		// HTTP bağlantısı testi
		log.Printf("Sunucu bağlantısı test ediliyor: %s", c.serverURL)

		resp, err := c.httpClient.Get(c.serverURL + "/api/stats")
		if err != nil {
			return fmt.Errorf("sunucu erişilemez: %v", err)
		}
		resp.Body.Close()

		c.isConnected = true
		log.Println("✅ HTTP sunucuya başarıyla bağlandı")
		return nil
	}

	// WebSocket bağlantısı (fallback)
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return err
	}

	wsURL := fmt.Sprintf("ws://%s:8765", u.Hostname())
	if u.Scheme == "https" {
		wsURL = fmt.Sprintf("wss://%s:8765", u.Hostname())
	}

	log.Printf("WebSocket bağlantısı: %s", wsURL)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}

	c.conn = conn
	c.isConnected = true

	// Client kaydını gönder
	registerMsg := map[string]interface{}{
		"type":      "client_register",
		"client_id": c.clientID,
	}

	if err := c.conn.WriteJSON(registerMsg); err != nil {
		log.Printf("⚠️ Client kayıt hatası: %v", err)
	}

	log.Println("✅ WebSocket sunucuya başarıyla bağlandı")
	return nil
}

func (c *Client) Disconnect() {
	if c.conn != nil {
		c.conn.Close()
	}
	c.isConnected = false
}

func (c *Client) StartScreenCapture() {
	ticker := time.NewTicker(33 * time.Millisecond) // 30 FPS
	defer ticker.Stop()

	// Frame skipping için
	lastFrameTime := time.Now()
	frameSkipThreshold := 50 * time.Millisecond

	log.Println("🎥 Ekran yakalama başlatıldı...")

	for range ticker.C {
		// Frame skipping - eğer işlem çok uzun sürüyorsa atla
		if time.Since(lastFrameTime) < frameSkipThreshold {
			continue
		}
		lastFrameTime = time.Now()

		if !c.isConnected {
			log.Println("⚠️ Bağlantı kesildi, yeniden bağlanmaya çalışılıyor...")
			if err := c.Connect(); err != nil {
				log.Printf("❌ Yeniden bağlanma hatası: %v", err)
				time.Sleep(3 * time.Second)
				continue
			}
		}

		// Ekran görüntüsü al
		img, err := c.takeScreenshot()
		if err != nil {
			log.Printf("⚠️ Ekran yakalama hatası: %v", err)
			continue
		}

		// JPEG'e çevir ve base64 encode et
		imageData, err := c.imageToBase64(img)
		if err != nil {
			log.Printf("⚠️ Image encode hatası: %v", err)
			continue
		}

		// Sunucuya gönder
		screenData := ScreenData{
			Type:      "screen_update",
			Image:     imageData,
			Timestamp: time.Now().Unix(),
			ClientID:  c.clientID,
		}

		if c.useHTTP {
			// HTTP POST ile gönder
			if err := c.sendScreenHTTP(screenData); err != nil {
				log.Printf("⚠️ HTTP veri gönderme hatası: %v", err)
				c.isConnected = false
				continue
			}
		} else {
			// WebSocket ile gönder
			if err := c.conn.WriteJSON(screenData); err != nil {
				log.Printf("⚠️ WebSocket veri gönderme hatası: %v", err)
				c.isConnected = false
				continue
			}
		}
	}
}

func (c *Client) takeScreenshot() (image.Image, error) {
	switch runtime.GOOS {
	case "darwin": // macOS
		return c.takeScreenshotMacOS()
	case "linux":
		return c.takeScreenshotLinux()
	case "windows":
		return c.takeScreenshotWindows()
	default:
		return nil, fmt.Errorf("desteklenmeyen platform: %s", runtime.GOOS)
	}
}

func (c *Client) takeScreenshotMacOS() (image.Image, error) {
	tmpFile := "/tmp/screenshot_" + fmt.Sprintf("%d", time.Now().UnixNano()) + ".png"

	cmd := exec.Command("screencapture", "-x", "-t", "png", tmpFile)
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	data, err := exec.Command("cat", tmpFile).Output()
	if err != nil {
		return nil, err
	}

	exec.Command("rm", tmpFile).Run()

	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

func (c *Client) takeScreenshotLinux() (image.Image, error) {
	cmd := exec.Command("gnome-screenshot", "-f", "/dev/stdout")
	output, err := cmd.Output()
	if err != nil {
		cmd = exec.Command("scrot", "-o", "/dev/stdout")
		output, err = cmd.Output()
		if err != nil {
			return nil, err
		}
	}

	img, _, err := image.Decode(bytes.NewReader(output))
	return img, err
}

func (c *Client) takeScreenshotWindows() (image.Image, error) {
	powershellScript := `
	Add-Type -AssemblyName System.Windows.Forms
	Add-Type -AssemblyName System.Drawing
	$Screen = [System.Windows.Forms.SystemInformation]::VirtualScreen
	$bitmap = New-Object System.Drawing.Bitmap $Screen.Width, $Screen.Height
	$graphics = [System.Drawing.Graphics]::FromImage($bitmap)
	$graphics.CopyFromScreen($Screen.Left, $Screen.Top, 0, 0, $bitmap.Size)
	$stream = New-Object System.IO.MemoryStream
	$bitmap.Save($stream, [System.Drawing.Imaging.ImageFormat]::Png)
	[Convert]::ToBase64String($stream.ToArray())
	`

	cmd := exec.Command("powershell", "-Command", powershellScript)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	data, err := base64.StdEncoding.DecodeString(string(output))
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

func (c *Client) imageToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer

	// Görüntüyü küçült (performans için)
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Eğer çok büyükse %50 küçült
	if width > 1920 || height > 1080 {
		width = width / 2
		height = height / 2
		// Basit downsampling
		smallImg := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				smallImg.Set(x, y, img.At(x*2, y*2))
			}
		}
		img = smallImg
	}

	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 50})
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "data:image/jpeg;base64," + encoded, nil
}

func (c *Client) sendScreenHTTP(screenData ScreenData) error {
	// JSON payload hazırla
	jsonData, err := json.Marshal(screenData)
	if err != nil {
		return err
	}

	// HTTP POST isteği gönder
	resp, err := c.httpClient.Post(
		c.serverURL+"/api/screen-update",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) StartAppBlocker() {
	if c.appBlocker == nil {
		return
	}

	interval := time.Duration(c.appBlocker.Settings.CheckIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Println("🚫 Uygulama engelleyici başlatıldı...")

	for range ticker.C {
		c.checkAndBlockApps()
	}
}

func (c *Client) checkAndBlockApps() {
	runningProcesses, err := c.getRunningProcesses()
	if err != nil {
		log.Printf("⚠️ Process listesi alınamadı: %v", err)
		return
	}

	for _, blockedApp := range c.appBlocker.BlockedApplications {
		for _, process := range blockedApp.Processes {
			if c.isProcessRunning(process, runningProcesses) {
				log.Printf("🚫 Yasaklı uygulama tespit edildi: %s (%s)", blockedApp.Name, process)
				c.handleBlockedApp(blockedApp, process)
				break // Aynı app için sadece bir kez uyarı göster
			}
		}
	}
}

func (c *Client) getRunningProcesses() ([]string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("ps", "aux")
	case "linux":
		cmd = exec.Command("ps", "aux")
	case "windows":
		cmd = exec.Command("tasklist", "/fo", "csv")
	default:
		return nil, fmt.Errorf("desteklenmeyen platform: %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	return lines, nil
}

func (c *Client) isProcessRunning(processName string, runningProcesses []string) bool {
	processLower := strings.ToLower(processName)

	for _, line := range runningProcesses {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, processLower) {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *Client) handleBlockedApp(app BlockedApp, process string) {
	// Uyarı sayısını artır
	c.warningCounts[app.Name]++

	if c.appBlocker.Settings.ShowWarnings {
		log.Printf("🚫 %s", app.WarningMessage)
		log.Printf("📊 Uyarı: %d/%d", c.warningCounts[app.Name], c.appBlocker.Settings.MaxWarnings)
	}

	// Maksimum uyarı sayısına ulaşıldıysa veya otomatik kapatma aktifse
	if c.appBlocker.Settings.AutoKill || c.warningCounts[app.Name] >= c.appBlocker.Settings.MaxWarnings {
		c.killProcess(process, app.Name)
		c.warningCounts[app.Name] = 0 // Sayacı sıfırla
	}
}

func (c *Client) killProcess(processName, appName string) {
	log.Printf("🔧 %s kapatılıyor...", appName)

	switch runtime.GOOS {
	case "darwin":
		// macOS için özel app kapatma
		success := c.killMacOSApp(processName, appName)
		if !success {
			// Fallback: Force kill
			log.Printf("🔄 %s force kill deneniyor...", appName)
			cmd := exec.Command("pkill", "-f", processName)
			err := cmd.Run()
			if err != nil {
				log.Printf("⚠️ %s kapatılamadı: %v", appName, err)
			} else {
				log.Printf("✅ %s force kill ile kapatıldı", appName)
			}
		}
	case "linux":
		cmd := exec.Command("pkill", "-f", processName)
		err := cmd.Run()
		if err != nil {
			log.Printf("⚠️ %s kapatılamadı: %v", appName, err)
		} else {
			log.Printf("✅ %s başarıyla kapatıldı", appName)
		}
	case "windows":
		cmd := exec.Command("taskkill", "/f", "/im", processName)
		err := cmd.Run()
		if err != nil {
			log.Printf("⚠️ %s kapatılamadı: %v", appName, err)
		} else {
			log.Printf("✅ %s başarıyla kapatıldı", appName)
		}
	default:
		log.Printf("❌ Process kapatma desteklenmiyor: %s", runtime.GOOS)
	}
}

func (c *Client) killMacOSApp(processName, appName string) bool {
	// Farklı yöntemler dene
	var cmds []*exec.Cmd

	// 1. App name ile AppleScript
	if appName == "Steam" {
		cmds = append(cmds, exec.Command("osascript", "-e", "quit app \"Steam\""))
	} else if appName == "Chrome" {
		cmds = append(cmds, exec.Command("osascript", "-e", "quit app \"Google Chrome\""))
	} else if appName == "Firefox" {
		cmds = append(cmds, exec.Command("osascript", "-e", "quit app \"Firefox\""))
	}

	// 2. killall komutu
	if strings.Contains(strings.ToLower(processName), "chrome") {
		cmds = append(cmds, exec.Command("killall", "Google Chrome"))
	} else if strings.Contains(strings.ToLower(processName), "firefox") {
		cmds = append(cmds, exec.Command("killall", "firefox"))
	} else if strings.Contains(strings.ToLower(processName), "steam") {
		cmds = append(cmds, exec.Command("killall", "steam"))
	}

	// 3. pkill
	cmds = append(cmds, exec.Command("pkill", "-i", processName))

	// Sırayla dene
	for _, cmd := range cmds {
		err := cmd.Run()
		if err == nil {
			log.Printf("✅ %s başarıyla kapatıldı", appName)
			return true
		}
	}

	return false
}

func (c *Client) StartWebsiteBlocker() {
	if c.websiteBlocker == nil {
		return
	}

	log.Println("🚫 Website engelleyici başlatıldı...")

	if c.websiteBlocker.Settings.BlockingMethod == "hosts" {
		// Hosts dosyası yöntemi (sudo gerektirir)
		if c.websiteBlocker.Settings.BackupHosts {
			c.backupHostsFile()
		}
		c.blockWebsites()
	}

	// Periyodik kontrol
	interval := time.Duration(c.websiteBlocker.Settings.CheckIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if c.websiteBlocker.Settings.BlockingMethod == "browser_check" {
			c.checkAndCloseBrowserTabs()
		} else {
			c.ensureWebsitesBlocked()
		}
	}
}

func (c *Client) backupHostsFile() {
	if runtime.GOOS == "windows" {
		return // Windows için şimdilik skip
	}

	// Hosts dosyasını backup'la
	cmd := exec.Command("sudo", "cp", "/etc/hosts", c.hostsBackupPath)
	err := cmd.Run()
	if err != nil {
		log.Printf("⚠️ Hosts dosyası backup edilemedi: %v", err)
	} else {
		log.Printf("✅ Hosts dosyası backup edildi: %s", c.hostsBackupPath)
	}
}

func (c *Client) blockWebsites() {
	if c.websiteBlocker == nil || !c.websiteBlocker.Settings.WebsiteBlockerEnabled {
		return
	}

	log.Printf("🚫 %d website engelleniyor...", len(c.websiteBlocker.BlockedWebsites))

	// Hosts dosyasına yasaklı siteleri ekle
	var hostsEntries []string
	hostsEntries = append(hostsEntries, "\n# Blocked websites by screen recorder client")

	for _, website := range c.websiteBlocker.BlockedWebsites {
		for _, url := range website.URLs {
			entry := fmt.Sprintf("%s %s", c.websiteBlocker.Settings.RedirectTo, url)
			hostsEntries = append(hostsEntries, entry)
		}
	}

	hostsEntries = append(hostsEntries, "# End blocked websites\n")

	// Hosts dosyasına append et
	c.appendToHostsFile(strings.Join(hostsEntries, "\n"))
}

func (c *Client) appendToHostsFile(content string) {
	var hostsPath string
	if runtime.GOOS == "windows" {
		hostsPath = "C:\\Windows\\System32\\drivers\\etc\\hosts"
	} else {
		hostsPath = "/etc/hosts"
	}

	// sudo ile append et
	cmd := exec.Command("sudo", "sh", "-c", fmt.Sprintf("echo '%s' >> %s", content, hostsPath))
	err := cmd.Run()
	if err != nil {
		log.Printf("⚠️ Hosts dosyasına yazılamadı: %v", err)
	} else {
		log.Printf("✅ %d website hosts dosyasına eklendi", len(c.websiteBlocker.BlockedWebsites))
	}
}

func (c *Client) ensureWebsitesBlocked() {
	// Hosts dosyasını kontrol et, eğer değiştirilmişse tekrar ekle
	var hostsPath string
	if runtime.GOOS == "windows" {
		hostsPath = "C:\\Windows\\System32\\drivers\\etc\\hosts"
	} else {
		hostsPath = "/etc/hosts"
	}

	data, err := ioutil.ReadFile(hostsPath)
	if err != nil {
		log.Printf("⚠️ Hosts dosyası okunamadı: %v", err)
		return
	}

	hostsContent := string(data)

	// Kontrol et: yasaklı siteler hala var mı?
	for _, website := range c.websiteBlocker.BlockedWebsites {
		for _, url := range website.URLs {
			if !strings.Contains(hostsContent, url) {
				log.Printf("🔄 %s tekrar engelleniyor...", url)
				c.blockWebsites()
				return
			}
		}
	}
}

func (c *Client) checkAndCloseBrowserTabs() {
	// Browser process'lerini kontrol et
	browsers := []string{"Google Chrome", "Firefox", "Safari", "Microsoft Edge"}

	for _, browser := range browsers {
		switch runtime.GOOS {
		case "darwin":
			c.checkMacOSBrowser(browser)
		case "linux":
			c.checkLinuxBrowser(browser)
		case "windows":
			c.checkWindowsBrowser(browser)
		}
	}
}

func (c *Client) checkMacOSBrowser(browserName string) {
	// AppleScript ile browser tab'larını kontrol et
	for _, website := range c.websiteBlocker.BlockedWebsites {
		for _, url := range website.URLs {
			// Chrome için
			if browserName == "Google Chrome" {
				script := fmt.Sprintf(`
					tell application "Google Chrome"
						set tabsToClose to {}
						repeat with w in windows
							repeat with t in tabs of w
								if URL of t contains "%s" then
									set end of tabsToClose to t
								end if
							end repeat
						end repeat
						repeat with t in tabsToClose
							close t
						end repeat
					end tell
				`, url)
				c.runAppleScript(script, website.WarningMessage)
			}

			// Firefox için
			if browserName == "Firefox" {
				// Firefox için daha basit yaklaşım - process'i kapat
				cmd := exec.Command("pgrep", "-f", "firefox")
				output, err := cmd.Output()
				if err == nil && len(output) > 0 {
					if c.websiteBlocker.Settings.ShowWarnings {
						log.Printf("🚫 %s", website.WarningMessage)
					}
					if c.websiteBlocker.Settings.CloseBrowserTabs {
						log.Printf("🔧 Firefox yasaklı site nedeniyle kapatılıyor...")
						exec.Command("osascript", "-e", "quit app \"Firefox\"").Run()
					}
				}
			}

			// Safari için
			if browserName == "Safari" {
				script := fmt.Sprintf(`
					tell application "Safari"
						repeat with w in windows
							repeat with t in tabs of w
								if URL of t contains "%s" then
									close t
								end if
							end repeat
						end repeat
					end tell
				`, url)
				c.runAppleScript(script, website.WarningMessage)
			}
		}
	}
}

func (c *Client) runAppleScript(script, warningMessage string) {
	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()
	if err == nil {
		if c.websiteBlocker.Settings.ShowWarnings {
			log.Printf("🚫 %s", warningMessage)
		}
		log.Printf("✅ Yasaklı tab kapatıldı")
	}
}

func (c *Client) checkLinuxBrowser(browserName string) {
	// Linux için basit process kill
	processNames := map[string]string{
		"Google Chrome": "chrome",
		"Firefox":       "firefox",
	}

	if processName, exists := processNames[browserName]; exists {
		cmd := exec.Command("pgrep", "-f", processName)
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			log.Printf("🚫 %s tespit edildi, kapatılıyor...", browserName)
			exec.Command("pkill", "-f", processName).Run()
		}
	}
}

func (c *Client) checkWindowsBrowser(browserName string) {
	// Windows için tasklist kullan
	processNames := map[string]string{
		"Google Chrome":  "chrome.exe",
		"Firefox":        "firefox.exe",
		"Microsoft Edge": "msedge.exe",
	}

	if processName, exists := processNames[browserName]; exists {
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", processName))
		output, err := cmd.Output()
		if err == nil && strings.Contains(string(output), processName) {
			log.Printf("🚫 %s tespit edildi, kapatılıyor...", browserName)
			exec.Command("taskkill", "/F", "/IM", processName).Run()
		}
	}
}

func (c *Client) unblockWebsites() {
	if runtime.GOOS == "windows" {
		return // Windows için şimdilik skip
	}

	// Backup'tan restore et
	if c.websiteBlocker.Settings.BackupHosts {
		cmd := exec.Command("sudo", "cp", c.hostsBackupPath, "/etc/hosts")
		err := cmd.Run()
		if err != nil {
			log.Printf("⚠️ Hosts dosyası restore edilemedi: %v", err)
		} else {
			log.Printf("✅ Hosts dosyası restore edildi")
		}
	}
}

func main() {
	// Server URL'ini al (argument veya environment variable)
	// Default olarak local server'ı kullan
	serverURL := "http://127.0.0.1:5000"
	if len(os.Args) > 1 {
		serverURL = os.Args[1]
	}
	if envURL := os.Getenv("SERVER_URL"); envURL != "" {
		serverURL = envURL
	}

	fmt.Printf("🚀 Screen Recorder Client\n")
	fmt.Printf("📡 Server: %s\n", serverURL)
	fmt.Printf("💻 Platform: %s\n", runtime.GOOS)

	// macOS izin uyarısı
	if runtime.GOOS == "darwin" {
		fmt.Println("⚠️  macOS'te 'Ekran Kaydı' izni gerekebilir")
		fmt.Println("   Sistem Ayarları > Gizlilik ve Güvenlik > Ekran Kaydı")
	}

	// Client oluştur
	client := NewClient(serverURL)

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\n🛑 Kapatılıyor...")
		client.Disconnect()
		os.Exit(0)
	}()

	// Sunucuya bağlan
	for {
		err := client.Connect()
		if err != nil {
			log.Printf("❌ Bağlantı hatası: %v", err)
			log.Println("🔄 3 saniye sonra tekrar denenecek...")
			time.Sleep(3 * time.Second)
			continue
		}
		break
	}

	// Uygulama engelleyiciyi başlat
	if client.appBlocker != nil && client.appBlocker.Settings.AppBlockerEnabled {
		go client.StartAppBlocker()
	}

	// Website engelleyiciyi başlat
	if client.websiteBlocker != nil && client.websiteBlocker.Settings.WebsiteBlockerEnabled {
		go client.StartWebsiteBlocker()
	}

	// Ekran yakalamayı başlat
	client.StartScreenCapture()
}
