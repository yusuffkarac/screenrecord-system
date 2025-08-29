# 🚂 Railway.app Deployment Rehberi

Railway.app ile daha güçlü kaynaklar ve $5/ay ücretsiz kredi.

## 🎯 Railway.app Avantajları

✅ **$5/ay ücretsiz kredi** - Render'dan daha cömert  
✅ **Daha güçlü kaynaklar** - 1GB RAM, 1 vCPU  
✅ **Always-on deployment** - Sleep yok  
✅ **WebSocket support** - Full support  
✅ **PostgreSQL database** - Ücretsiz DB  

## 🚀 Hızlı Deployment

### 1. Railway'de Hesap Oluşturun
```
https://railway.app
GitHub ile sign up
```

### 2. Proje Oluşturun
```bash
# Terminal'de
cd render-deploy/
npm init -y
# railway.json oluşturun
```

### 3. Railway.json Dosyası
```json
{
  "$schema": "https://railway.app/railway.schema.json",
  "build": {
    "builder": "NIXPACKS"
  },
  "deploy": {
    "startCommand": "python app.py",
    "healthcheckPath": "/health"
  }
}
```

### 4. Deploy Edin
```bash
# Railway CLI kurulumu
npm install -g @railway/cli

# Login ve deploy
railway login
railway init
railway up
```

## 💰 Maliyet Karşılaştırması

| Platform | Ücretsiz Limit | RAM | CPU | Always-On |
|----------|----------------|-----|-----|-----------|
| **Railway** | $5/ay kredi | 1GB | 1 vCPU | ✅ |
| **Render** | 750 saat/ay | 512MB | 0.1 CPU | ❌ |
| **Fly.io** | Shared CPU | 256MB | Shared | ❌ |

## 📋 Sonuç

Railway.app **en güçlü ücretsiz seçenek** ancak kredi biterse ücretli hale gelir.
