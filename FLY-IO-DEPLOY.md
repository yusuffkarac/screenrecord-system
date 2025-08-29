# 🪰 Fly.io Deployment - Daha Stabil Alternatif

Render.com'da sorun yaşıyorsanız Fly.io daha stabil olabilir.

## 🎯 Fly.io Avantajları

✅ **Shared CPU ücretsiz** - Hiç çökmez  
✅ **Always-on** - Sleep yok  
✅ **Global deployment** - Dünya çapında  
✅ **Docker support** - Daha stabil  
✅ **WebSocket full support**  

## 🚀 Hızlı Setup

### 1. Fly.io CLI Kurulumu
```bash
# macOS
brew install flyctl

# Login
fly auth login
```

### 2. Dockerfile Oluştur
```dockerfile
FROM python:3.11-slim

WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt

COPY . .
EXPOSE 8080

CMD ["python", "app.py"]
```

### 3. Fly.io Deploy
```bash
# App oluştur
fly apps create screenrecord-system

# Deploy et
fly deploy
```

## 💰 Maliyet
- **Shared CPU**: Tamamen ücretsiz
- **Always-on**: Ek ücret yok
- **Global**: Tüm dünyadan erişim

## 🔗 Sonuç URL
```
https://screenrecord-system.fly.dev
```

Bu çok daha stabil çalışır!
