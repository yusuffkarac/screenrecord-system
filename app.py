#!/usr/bin/env python3
"""
Render.com deployment için optimize edilmiş server
WebSocket + SocketIO desteği ile
"""
import os
import sys
import time
import json
import threading
from datetime import datetime
from flask import Flask, render_template, request, jsonify
from flask_socketio import SocketIO, emit, disconnect
import socket

app = Flask(__name__)
app.config['SECRET_KEY'] = os.environ.get('SECRET_KEY', 'screen-recorder-secret-key-2024')

# SocketIO - Render.com için optimize edilmiş ayarlar
socketio = SocketIO(
    app, 
    cors_allowed_origins="*",
    async_mode='threading',
    transports=['websocket', 'polling'],  # Both WebSocket and polling
    ping_timeout=60,
    ping_interval=25
)

# Global değişkenler
connected_clients = {}  # clientId -> client_info
latest_screens = {}     # clientId -> latest_screen_data
viewers = {}           # socketId -> viewer_info
stats = {
    'server_start_time': datetime.now(),
    'total_frames': 0,
    'total_data_mb': 0.0
}

@app.route('/')
def index():
    """Ana sayfa - Web viewer"""
    return render_template('index.html')

@app.route('/health')
def health():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'timestamp': int(time.time()),
        'service': 'screen-recorder-render',
        'uptime_seconds': int((datetime.now() - stats['server_start_time']).total_seconds()),
        'clients': len(connected_clients),
        'viewers': len(viewers),
        'frames_processed': stats['total_frames']
    })

@app.route('/api/stats')
def api_stats():
    """Detaylı istatistikler"""
    uptime = datetime.now() - stats['server_start_time']
    return jsonify({
        'server': {
            'uptime_seconds': int(uptime.total_seconds()),
            'uptime_human': str(uptime).split('.')[0],
            'version': '2.0-render'
        },
        'connections': {
            'clients': len(connected_clients),
            'viewers': len(viewers),
            'total_clients_ever': len(connected_clients)
        },
        'performance': {
            'total_frames': stats['total_frames'],
            'total_data_mb': round(stats['total_data_mb'], 2),
            'avg_fps': round(stats['total_frames'] / max(uptime.total_seconds(), 1), 1)
        }
    })

# WebSocket Events
@socketio.on('connect')
def handle_connect():
    """Bağlantı kuruldu"""
    print(f"📡 Yeni bağlantı: {request.sid}")

@socketio.on('disconnect')
def handle_disconnect():
    """Bağlantı kesildi"""
    client_id = None
    
    # Client mi viewer mı kontrol et
    for cid, client in connected_clients.items():
        if client.get('socket_id') == request.sid:
            client_id = cid
            break
    
    if client_id:
        print(f"📱 Client ayrıldı: {client_id}")
        if client_id in connected_clients:
            del connected_clients[client_id]
        if client_id in latest_screens:
            del latest_screens[client_id]
    
    if request.sid in viewers:
        print(f"👁️ Viewer ayrıldı: {request.sid}")
        del viewers[request.sid]
    
    # Tüm viewer'lara client listesini güncelle
    emit_client_list()

@socketio.on('client_register')
def handle_client_register(data):
    """Go client kaydı"""
    client_id = data.get('clientId', f'client_{int(time.time())}')
    
    connected_clients[client_id] = {
        'id': client_id,
        'socket_id': request.sid,
        'connected_at': datetime.now(),
        'last_seen': datetime.now(),
        'frames_sent': 0,
        'user_agent': request.headers.get('User-Agent', 'Unknown')
    }
    
    print(f"📱 Client kaydedildi: {client_id}")
    emit_client_list()

@socketio.on('join_web_viewer')
def handle_join_web_viewer():
    """Web viewer katılımı"""
    viewers[request.sid] = {
        'joined_at': datetime.now(),
        'user_agent': request.headers.get('User-Agent', 'Web Browser')
    }
    
    print(f"👁️ Web viewer katıldı: {request.sid}")
    
    # Mevcut client'ları ve ekranları gönder
    emit('client_list', {
        'clients': list(connected_clients.keys()),
        'latest_screens': latest_screens
    })

@socketio.on('screen_update')
def handle_screen_update(data):
    """Ekran güncellemesi (Go client'tan)"""
    client_id = data.get('clientId')
    if not client_id:
        return
    
    # Client bilgilerini güncelle
    if client_id in connected_clients:
        connected_clients[client_id]['last_seen'] = datetime.now()
        connected_clients[client_id]['frames_sent'] += 1
    
    # Ekran verisini sakla
    latest_screens[client_id] = {
        'clientId': client_id,
        'image': data.get('image'),
        'timestamp': data.get('timestamp', int(time.time())),
        'type': 'screen_update'
    }
    
    # İstatistikleri güncelle
    stats['total_frames'] += 1
    if data.get('image'):
        # Base64 image size estimate (rough)
        image_size_mb = len(data.get('image', '')) * 0.75 / (1024 * 1024)  # Base64 overhead
        stats['total_data_mb'] += image_size_mb
    
    # Tüm web viewer'lara gönder
    socketio.emit('screen_update', latest_screens[client_id], room=None, include_self=False)
    
    # Her 50 frame'de log
    if stats['total_frames'] % 50 == 0:
        print(f"📺 Frame #{stats['total_frames']} işlendi. Clients: {len(connected_clients)}, Viewers: {len(viewers)}")

def emit_client_list():
    """Güncel client listesini tüm viewer'lara gönder"""
    socketio.emit('client_list', {
        'clients': list(connected_clients.keys()),
        'latest_screens': latest_screens
    })

def cleanup_inactive_clients():
    """İnaktif client'ları temizle"""
    while True:
        try:
            current_time = datetime.now()
            inactive_clients = []
            
            for client_id, client in connected_clients.items():
                last_seen = client.get('last_seen', client.get('connected_at'))
                if (current_time - last_seen).total_seconds() > 300:  # 5 dakika
                    inactive_clients.append(client_id)
            
            for client_id in inactive_clients:
                print(f"🧹 İnaktif client temizlendi: {client_id}")
                if client_id in connected_clients:
                    del connected_clients[client_id]
                if client_id in latest_screens:
                    del latest_screens[client_id]
            
            if inactive_clients:
                emit_client_list()
                
        except Exception as e:
            print(f"⚠️ Cleanup hatası: {e}")
        
        time.sleep(60)  # Her dakika kontrol et

if __name__ == '__main__':
    # Port ayarları
    port = int(os.environ.get('PORT', 5000))
    
    print("=" * 60)
    print("🚀 SCREEN RECORDER SERVER (Render.com)")
    print("=" * 60)
    print(f"🌍 Port: {port}")
    print(f"🔧 Environment: {os.environ.get('RENDER', 'Development')}")
    print("=" * 60)
    
    # Cleanup thread başlat
    cleanup_thread = threading.Thread(target=cleanup_inactive_clients, daemon=True)
    cleanup_thread.start()
    
    # SocketIO sunucusunu başlat
    try:
        socketio.run(
            app, 
            host='0.0.0.0',
            port=port,
            debug=False,
            allow_unsafe_werkzeug=True
        )
    except Exception as e:
        print(f"⚠️ SocketIO başlatma hatası: {e}")
        # Fallback to basic Flask
        app.run(host='0.0.0.0', port=port, debug=False)
