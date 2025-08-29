# 📱 Screen Recorder Client

Bilgisayarınızın ekranını yakalar ve remote server'a gönderir.

## 🚀 Hızlı Başlangıç

```bash
# Bağımlılıkları yükle
go mod tidy

# Varsayılan server ile çalıştır
go run main.go

# Özel server URL ile çalıştır
go run main.go https://your-server.herokuapp.com

# Environment variable ile
export SERVER_URL=https://your-server.herokuapp.com
go run main.go
```

## ⚙️ Platform Gereksinimleri

### macOS
- `screencapture` komutu (built-in)
- Ekran Kaydı izni gerekli

### Linux
- `gnome-screenshot` veya `scrot`
```bash
sudo apt install gnome-screenshot
# veya
sudo apt install scrot
```

### Windows
- PowerShell (built-in)
- .NET Framework

## 🔧 Ayarlar

### FPS Değiştirme
`main.go` dosyasında:
```go
ticker := time.NewTicker(100 * time.Millisecond) // 10 FPS
// Değiştir:
ticker := time.NewTicker(50 * time.Millisecond)  // 20 FPS
```

### Kalite Ayarı
```go
jpeg.Options{Quality: 70} // 1-100 arası
```

## 📊 Performance Tips

- **Yüksek FPS**: Daha fazla CPU ve bandwidth kullanır
- **Yüksek Kalite**: Daha büyük dosya boyutu
- **Optimal**: 60-80 kalite, 10-15 FPS

## 🛑 Durdurma

`Ctrl+C` ile güvenli şekilde kapatın.
