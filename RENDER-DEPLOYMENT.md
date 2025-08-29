# 🚀 Render.com Ücretsiz Deployment Rehberi

Bu rehber, ekran yakalama sisteminizi Render.com'da ücretsiz olarak nasıl yayınlayacağınızı gösterir.

## 🎯 Özellikler

✅ **24/7 Çalışır** - Her zaman açık  
✅ **WebSocket Desteği** - Real-time iletişim  
✅ **750 Saat/Ay Ücretsiz** - Limit dahilinde  
✅ **HTTPS SSL** - Güvenli bağlantı  
✅ **Global CDN** - Hızlı erişim  
✅ **Auto-Deploy** - Git push ile otomatik güncelleme  

## 📋 Gereksinimler

1. GitHub hesabı
2. Render.com hesabı (ücretsiz)
3. Git kurulu bilgisayar

## 🛠️ Adım Adım Deployment

### 1. GitHub Repository Oluşturun

```bash
# Proje klasörünüzde
cd /Users/yusufsimply/Desktop/screenrecord-system
git init
git add render-deploy/
git commit -m "Initial commit for Render deployment"

# GitHub'da yeni repo oluşturun: screenrecord-system
git remote add origin https://github.com/KULLANICI_ADINIZ/screenrecord-system.git
git branch -M main
git push -u origin main
```

### 2. Render.com'da Hesap Oluşturun

1. [render.com](https://render.com) sitesine gidin
2. **Sign Up** ile GitHub hesabınızla kayıt olun
3. GitHub'a erişim izni verin

### 3. Web Service Oluşturun

1. Dashboard'da **New +** butonuna tıklayın
2. **Web Service** seçin
3. GitHub repository'nizi seçin: `screenrecord-system`
4. Şu ayarları yapın:

```
Name: screenrecord-system
Environment: Python 3
Region: Oregon (US West)
Branch: main
Root Directory: render-deploy
Build Command: pip install -r requirements.txt
Start Command: python app.py
```

### 4. Environment Variables (Opsiyonel)

Advanced bölümünde:
```
SECRET_KEY: [Generate] (otomatik oluştur)
RENDER: true
```

### 5. Deploy Edin

- **Create Web Service** butonuna tıklayın
- Deploy işlemi 2-3 dakika sürer
- Başarılı olursa URL'niz hazır: `https://your-app-name.onrender.com`

## 🌐 Sonuç URL'niz

Deploy tamamlandıktan sonra şu adresi alacaksınız:
```
https://screenrecord-system-xyz.onrender.com
```

Bu adres:
- ✅ **7/24 aktif** olacak
- ✅ **HTTPS güvenli** bağlantı
- ✅ **Global erişim** mümkün
- ✅ **WebSocket** destekli

## 📱 Client Bağlantısı

### Go Client'ınızı Güncelleyin

`client/main.go` dosyasında:

```go
// serverURL := "https://go-record2-d5bef2a4b84c.herokuapp.com"
serverURL := "https://your-app-name.onrender.com"
```

### Client'ı Çalıştırın

```bash
cd client/
SERVER_URL=https://your-app-name.onrender.com go run main.go
```

## 💰 Ücretsiz Limitler

### Render.com Free Plan:
- **750 saat/ay** çalışma süresi
- **512 MB RAM**
- **0.1 CPU**
- **Idle timeout**: 15 dakika (aktivite yoksa uyur)
- **Cold start**: ~30 saniye (uyandırma süresi)

### Limit Yönetimi:
```bash
# 750 saat = ~25 gün sürekli çalışma
# Aylık kullanım planlaması yapın
# İhtiyaç dışında durdurabilirsiniz
```

## 🔧 Troubleshooting

### Problem: Deploy Hatası
```bash
# Build logs'u kontrol edin
# Requirements.txt'deki versiyonları kontrol edin
# Python 3.11 uyumluluğunu doğrulayın
```

### Problem: WebSocket Çalışmıyor
```bash
# HTTPS kullandığınızdan emin olun
# Browser console'da hata kontrol edin
# SocketIO transport ayarlarını kontrol edin
```

### Problem: Cold Start Gecikmesi
```bash
# 15 dakika idle'dan sonra uyur
# İlk erişim 30-60 saniye sürebilir
# Keep-alive service kullanabilirsiniz (ücretli)
```

## 🎛️ Monitoring & Logs

### Render Dashboard'da:
- **Logs** sekmesinden real-time logları izleyin
- **Metrics** ile CPU/Memory kullanımını takip edin
- **Events** ile deploy geçmişini görün

### Health Check:
```bash
curl https://your-app-name.onrender.com/health
```

## 🔄 Güncelleme

Kod değişiklikleri yaptığınızda:

```bash
git add .
git commit -m "Update: açıklama"
git push origin main
# Render otomatik olarak yeniden deploy eder
```

## 🆚 Alternatif Platformlar

Eğer Render.com'da sorun yaşarsanız:

### Railway.app:
- $5/ay ücretsiz kredi
- Daha fazla kaynak
- WebSocket desteği

### Fly.io:
- Shared CPU ücretsiz
- Global deployment
- Docker desteği

## 📞 Destek

### Faydalı Linkler:
- [Render.com Docs](https://render.com/docs)
- [Python Deployment Guide](https://render.com/docs/deploy-flask)
- [WebSocket Support](https://render.com/docs/websockets)

### Sorun Yaşarsanız:
1. Render logs'unu kontrol edin
2. GitHub repository ayarlarını doğrulayın
3. Environment variables'ları kontrol edin
4. Build/Start command'larını doğrulayın

## 🎉 Test Etme

Deploy tamamlandıktan sonra:

1. **Web arayüze gidin**: `https://your-app-name.onrender.com`
2. **Health check**: `https://your-app-name.onrender.com/health`
3. **Client'ı bağlayın**: `go run main.go https://your-app-name.onrender.com`
4. **Canlı yayını test edin**: Ekranınız web'de görünmeli

🎯 **Başarılı deployment sonrası sisteminiz dünya çapında erişilebilir olacak!**
