# Automated SSL Deployment for api.snippy.jheysonsaavedra.com

Esta guía explica cómo está configurado el deployment automático con SSL para el dominio `api.snippy.jheysonsaavedra.com`.

## 🚀 Configuración Automática

### GitHub Actions Workflows

#### 1. `build-and-deploy.yml` - Deploy Principal

- **Trigger**: Manual o tags `v*.*.*`
- **Funciones**:
  - ✅ Ejecuta tests y security checks
  - ✅ Compila el binario Go
  - ✅ Configura certificados SSL automáticamente
  - ✅ Despliega con Docker Compose
  - ✅ Configura renovación automática

#### 2. `ssl-renewal.yml` - Renovación SSL

- **Trigger**: Semanal (domingos 3 AM) o manual
- **Funciones**:
  - ✅ Renueva certificados Let's Encrypt
  - ✅ Actualiza certificados en el servidor
  - ✅ Verifica que SSL funcione correctamente
  - ✅ Registra logs de renovación

### 🔐 Certificados SSL

#### Let's Encrypt (Automático)

```bash
# El sistema automáticamente:
# 1. Instala certbot
# 2. Obtiene certificados para api.snippy.jheysonsaavedra.com
# 3. Configura renovación automática
# 4. Actualiza docker-compose con SSL
```

#### PostgreSQL SSL (Auto-generado)

```bash
# Certificados internos para PostgreSQL:
# - Generados automáticamente
# - Solo para comunicación interna
# - Renovados en cada deploy si es necesario
```

## 🛠️ Scripts de Automatización

### `/scripts/renew-ssl.sh`

- Renueva certificados Let's Encrypt
- Actualiza certificados de la API
- Reinicia servicios automáticamente
- Verifica que todo funcione

### `/scripts/install-cron.sh`

- Configura cron job para renovación automática
- Ejecuta diariamente a las 3 AM
- Logs en `/var/log/ssl-renewal.log`

## 🔧 Secrets de GitHub Requeridos

Para que el deployment automático funcione, necesitas configurar estos secrets en GitHub:

```bash
# Servidor
DROPLET_HOST=tu-servidor.com
DROPLET_USERNAME=root
DROPLET_SSH_KEY=tu-llave-ssh-privada

# Let's Encrypt
LETSENCRYPT_EMAIL=jheyson@jheysonsaavedra.com
```

### Configurar Secrets:

1. Ve a GitHub → Settings → Secrets and variables → Actions
2. Click "New repository secret"
3. Agrega cada secret con su valor

## 🌐 URLs del Sistema

### Producción

- **API HTTPS**: `https://api.snippy.jheysonsaavedra.com`
- **Health Check**: `https://api.snippy.jheysonsaavedra.com/api/v1/health`
- **API Docs**: `https://api.snippy.jheysonsaavedra.com/api/v1`

### Certificados

- **Let's Encrypt**: `/etc/letsencrypt/live/api.snippy.jheysonsaavedra.com/`
- **API SSL**: `/root/snippy-api/ssl/api/`
- **PostgreSQL SSL**: `/root/snippy-api/ssl/postgres/`

## 📋 Proceso de Deploy

### 1. Deploy Manual

```bash
# Desde GitHub:
# Actions → Build and Deploy → Run workflow
# Selecciona "production"
```

### 2. Deploy con Tag

```bash
# Crear release tag:
git tag v1.0.0
git push origin v1.0.0
# Deploy automático se ejecuta
```

### 3. Verificación Post-Deploy

```bash
# El workflow automáticamente verifica:
# ✅ Servicios Docker corriendo
# ✅ Health check HTTPS
# ✅ Certificado SSL válido
# ✅ Dominio accesible
```

## 🔍 Monitoreo y Logs

### Ver Estado del Sistema

```bash
# SSH al servidor
ssh root@tu-servidor.com

# Ver servicios
cd /root/snippy-api
docker-compose ps

# Ver logs
docker-compose logs -f api
docker-compose logs -f postgres

# Ver logs SSL
tail -f /var/log/ssl-renewal.log
```

### Ver Certificados

```bash
# Ver expiración del certificado
openssl x509 -enddate -noout -in /etc/letsencrypt/live/api.snippy.jheysonsaavedra.com/fullchain.pem

# Verificar SSL del dominio
curl -I https://api.snippy.jheysonsaavedra.com/api/v1/health

# Test SSL completo
openssl s_client -connect api.snippy.jheysonsaavedra.com:443 -servername api.snippy.jheysonsaavedra.com
```

## 🚨 Troubleshooting

### Certificado SSL Expirado

```bash
# Renovar manualmente
cd /root/snippy-api
export LETSENCRYPT_EMAIL=jheyson@jheysonsaavedra.com
./scripts/renew-ssl.sh
```

### Servicios No Responden

```bash
# Verificar estado
systemctl status snippy-api
docker-compose ps

# Reiniciar
systemctl restart snippy-api
```

### GitHub Actions Falla

1. Verifica secrets configurados
2. Revisa logs del workflow
3. Verifica conectividad SSH
4. Verifica que el dominio apunte al servidor

## 🔄 Renovación Automática

### Cron Job Local (Backup)

```bash
# Se instala automáticamente, pero puedes verificar:
crontab -l
# Debe mostrar: 0 3 * * * /root/snippy-api/scripts/renew-ssl.sh
```

### GitHub Actions (Principal)

- Se ejecuta semanalmente
- Logs disponibles en Actions tab
- Notificaciones en caso de fallo

## 📊 Configuración Final

Tu sistema queda configurado con:

- ✅ **HTTPS obligatorio** (puerto 443)
- ✅ **SSL automático** con Let's Encrypt
- ✅ **Renovación automática** (GitHub Actions + Cron)
- ✅ **PostgreSQL SSL** para seguridad interna
- ✅ **Deploy automático** con tags/manual
- ✅ **Health checks** automáticos
- ✅ **Logs centralizados**

## 🎯 Próximos Pasos

1. **DNS**: Asegúrate que `api.snippy.jheysonsaavedra.com` apunte a tu servidor
2. **Firewall**: Abre puertos 80, 443, y 22
3. **Monitoring**: Considera agregar alertas (Slack, email)
4. **Backup**: Configura backups automáticos de la base de datos
