# 🖥️ Screen Recorder System

Real-time screen sharing system with WebSocket support. Share your screen live through a web interface.

## 🌟 Features

- **Real-time screen sharing** at 30 FPS
- **Multi-client support** - Multiple computers can stream simultaneously  
- **Web viewer interface** - Watch streams from any browser
- **Cross-platform** - Works on Windows, macOS, and Linux
- **WebSocket technology** - Low latency streaming
- **Auto-reconnection** - Handles network interruptions gracefully

## 🚀 Live Demo

**Web Viewer**: [https://screenrecord-system.onrender.com](https://screenrecord-system.onrender.com)

## 📱 How to Connect Your Computer

### 1. Download Go Client

```bash
git clone https://github.com/yusuffkarac/screenrecord-system.git
cd screenrecord-system/client
```

### 2. Run Client

```bash
# Set server URL and run
SERVER_URL=https://screenrecord-system.onrender.com go run main.go
```

### 3. View Stream

Open [https://screenrecord-system.onrender.com](https://screenrecord-system.onrender.com) in your browser to see your screen live!

## 🛠️ Local Development

### Server
```bash
pip install -r requirements.txt
python app.py
```

### Client
```bash
cd client/
go run main.go http://localhost:5000
```

## 🔧 System Requirements

- **Go 1.19+** for client
- **Python 3.11+** for server
- **Network connection** for streaming

## 📊 Architecture

```
[Your Computer] → [Go Client] → [WebSocket] → [Server] → [Web Interface]
```

## 🌍 Deployment

This system is deployed on [Render.com](https://render.com) with:
- 24/7 availability
- HTTPS security
- Global CDN
- WebSocket support

## 👨‍💻 Developer

Created by [Yusuf Karaç](https://github.com/yusuffkarac)

## 📄 License

MIT License - Feel free to use and modify!
