# HTTP to HTTPS Redirection Setup

Este documento explica cómo funciona la redirección automática de HTTP a HTTPS en el sistema.

## 🏗️ Arquitectura

### Dos Servicios Separados:

1. **`snippy-api`** - API principal (Docker)

   - Puerto: 443 (HTTPS)
   - Maneja: Todas las requests de la API
   - SSL: Let's Encrypt certificates

2. **`http-redirect`** - Servidor de redirección (Nativo)
   - Puerto: 80 (HTTP)
   - Maneja: Redirecciones HTTP → HTTPS
   - Función: Redirige todo tráfico HTTP a HTTPS

## 🔄 Flujo de Requests

```
Cliente → HTTP (port 80) → http-redirect → 301 Redirect → Cliente → HTTPS (port 443) → snippy-api
```

### Ejemplo:

```bash
# Request HTTP
curl http://api.snippy.jheysonsaavedra.com/api/v1/health

# Respuesta 301 Redirect
HTTP/1.1 301 Moved Permanently
Location: https://api.snippy.jheysonsaavedra.com/api/v1/health

# Cliente automáticamente hace nueva request
curl https://api.snippy.jheysonsaavedra.com/api/v1/health

# Respuesta exitosa del API
{"status":"ok"}
```

## 🛠️ Implementación

### 1. Binarios Compilados

```bash
# API principal
go build -o snippy-api .

# Servidor de redirección
go build -o http-redirect ./cmd/redirect
```

### 2. Servicios systemd

#### `/etc/systemd/system/snippy-api.service`

```ini
[Unit]
Description=Snippy API with SSL
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/root/snippy-api
ExecStart=/usr/bin/docker-compose up -d
# ... (puerto 443)
```

#### `/etc/systemd/system/http-redirect.service`

```ini
[Unit]
Description=HTTP to HTTPS Redirect Server
After=network.target

[Service]
Type=simple
WorkingDirectory=/root/snippy-api
ExecStart=/root/snippy-api/http-redirect
Environment=HTTP_PORT=80
# ... (puerto 80)
```

### 3. Docker Compose

```yaml
api:
  ports:
    - "443:443" # Solo HTTPS
  # Sin puerto 80 - manejado por servicio separado
```

## 🔐 Configuración SSL

### Certificados Let's Encrypt

- **Ubicación**: `/etc/letsencrypt/live/api.snippy.jheysonsaavedra.com/`
- **Uso**: Copiados a `/root/snippy-api/ssl/api/`
- **Renovación**: Automática vía GitHub Actions + cron

### Configuración de Timeouts

```go
// cmd/redirect/main.go
server := &http.Server{
    Addr:         ":80",
    ReadTimeout:  5 * time.Second,   // Seguridad
    WriteTimeout: 10 * time.Second,  // Seguridad
    IdleTimeout:  15 * time.Second,  // Seguridad
}
```

## 🚀 Deploy Automático

### GitHub Actions Steps:

1. **Build**: Compila ambos binarios (`snippy-api`, `http-redirect`)
2. **Copy**: Transfiere binarios y servicios systemd
3. **Install**: Configura ambos servicios
4. **Start**: Inicia ambos servicios
5. **Verify**: Verifica redirección HTTP y API HTTPS

### Verificación Post-Deploy:

```bash
# Verificar servicios
systemctl status snippy-api
systemctl status http-redirect

# Verificar redirección
curl -I http://api.snippy.jheysonsaavedra.com/api/v1/health
# Debe retornar: 301 Moved Permanently

# Verificar API
curl https://api.snippy.jheysonsaavedra.com/api/v1/health
# Debe retornar: {"status":"ok"}
```

## 🔍 Monitoreo

### Logs de Servicios

```bash
# Logs del API principal
journalctl -u snippy-api -f

# Logs del servidor de redirección
journalctl -u http-redirect -f

# Logs de Docker
cd /root/snippy-api
docker-compose logs -f
```

### Health Checks

```bash
# Check redirección HTTP
curl -I http://api.snippy.jheysonsaavedra.com

# Check API HTTPS
curl https://api.snippy.jheysonsaavedra.com/api/v1/health

# Check certificado SSL
openssl s_client -connect api.snippy.jheysonsaavedra.com:443
```

## ⚠️ Troubleshooting

### Puerto 80 ocupado

```bash
# Ver qué proceso usa puerto 80
sudo lsof -i :80

# Detener servicio que ocupa puerto 80
sudo systemctl stop apache2  # o nginx, etc.
sudo systemctl restart http-redirect
```

### Redirección no funciona

```bash
# Verificar que el servicio esté corriendo
systemctl status http-redirect

# Verificar logs
journalctl -u http-redirect -n 50

# Reiniciar servicio
systemctl restart http-redirect
```

### SSL no funciona

```bash
# Verificar certificados
ls -la /etc/letsencrypt/live/api.snippy.jheysonsaavedra.com/

# Verificar copia local
ls -la /root/snippy-api/ssl/api/

# Renovar certificados
cd /root/snippy-api
./scripts/renew-ssl.sh
```

## 🎯 Beneficios

### Seguridad:

- ✅ **Fuerza HTTPS**: Todo tráfico HTTP se redirige automáticamente
- ✅ **Timeouts configurados**: Previene ataques DoS
- ✅ **Headers de seguridad**: HSTS, X-Frame-Options, etc.

### Performance:

- ✅ **Redirección rápida**: Servidor nativo sin overhead
- ✅ **301 Permanent**: Browsers cachean la redirección
- ✅ **Separación de responsabilidades**: HTTP y HTTPS en procesos independientes

### Mantenimiento:

- ✅ **Deploy automático**: GitHub Actions maneja todo
- ✅ **Monitoreo independiente**: Servicios separados
- ✅ **Renovación SSL automática**: Sin intervención manual

## 📊 Estado Final

Con esta configuración tienes:

- **Puerto 80**: `http-redirect` service → Redirecciona a HTTPS
- **Puerto 443**: `snippy-api` Docker → API principal con SSL
- **Deploy automático**: GitHub Actions configura ambos
- **Renovación SSL**: Automática semanalmente
- **Monitoreo completo**: Health checks para ambos servicios
