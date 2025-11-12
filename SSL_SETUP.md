# SSL Configuration for Snippy Backend

Esta guía te ayudará a configurar SSL/TLS tanto para la API como para la base de datos PostgreSQL.

## 🚀 Setup Rápido

```bash
# 1. Ejecutar el script de setup SSL
chmod +x scripts/setup-ssl.sh
./scripts/setup-ssl.sh

# 2. Compilar la aplicación
go build

# 3. Iniciar con Docker
docker-compose up -d
```

## 📋 ¿Qué se configura?

### 🔐 PostgreSQL SSL:

- Certificados SSL auto-generados para PostgreSQL
- Conexiones SSL obligatorias (`sslmode=require`)
- Configuración de seguridad mejorada
- Protocolos TLS 1.2+

### 🌐 API HTTPS:

- Certificados SSL auto-generados para la API
- Servidor HTTPS en puerto 443
- Soporte para HTTP y HTTPS simultáneo
- Headers de seguridad

## 🔧 Archivos Creados

```
ssl/
├── postgres/
│   ├── server.key     # Llave privada PostgreSQL
│   ├── server.crt     # Certificado PostgreSQL
│   └── root.crt       # Certificado raíz para clientes
└── api/
    ├── api.key        # Llave privada API
    ├── api.crt        # Certificado API
    └── api.conf       # Configuración del certificado

config/
└── postgresql.conf    # Configuración SSL PostgreSQL

scripts/
├── setup-ssl.sh            # Script completo de setup
├── setup-postgres-ssl.sh   # Solo PostgreSQL
└── setup-api-ssl.sh        # Solo API
```

## 🌍 Variables de Entorno

En `.env.production`:

```bash
# Base de datos con SSL
DATABASE_URL=postgres://user:pass@postgres:5432/db?sslmode=require&sslcert=/app/ssl/postgres/root.crt

# API con HTTPS
PORT=443
SSL_CERT_FILE=/app/ssl/api/api.crt
SSL_KEY_FILE=/app/ssl/api/api.key
```

## 🔒 Certificados de Producción

Para producción, reemplaza los certificados auto-generados con certificados reales:

### Option 1: Let's Encrypt (Recomendado)

```bash
# Instalar certbot
sudo apt update && sudo apt install certbot

# Obtener certificados
sudo certbot certonly --standalone -d tudominio.com

# Actualizar .env.production
SSL_CERT_FILE=/etc/letsencrypt/live/tudominio.com/fullchain.pem
SSL_KEY_FILE=/etc/letsencrypt/live/tudominio.com/privkey.pem
```

### Option 2: Certificados comerciales

```bash
# Copiar tus certificados
cp tu-certificado.crt ssl/api/api.crt
cp tu-llave.key ssl/api/api.key
```

## 🐳 Docker Configuration

El `docker-compose.yml` está configurado para:

- PostgreSQL con SSL en puerto 5432
- API con HTTPS en puerto 443
- API con HTTP en puerto 80 (opcional, para redirección)
- Volúmenes para certificados SSL

## 🧪 Verificación

### Verificar PostgreSQL SSL:

```bash
# Desde dentro del contenedor
docker exec -it snippy-postgres psql -U snippy_user -d snippy_production -c "SELECT ssl_is_used();"
```

### Verificar API HTTPS:

```bash
# Verificar que HTTPS funciona
curl -k https://tudominio.com/api/v1/health

# Verificar certificado
openssl s_client -connect tudominio.com:443 -servername tudominio.com
```

## 🔧 Troubleshooting

### Error: "permission denied for SSL key file"

```bash
# Arreglar permisos
chmod 600 ssl/api/api.key ssl/postgres/server.key
chmod 644 ssl/api/api.crt ssl/postgres/server.crt
```

### Error: "certificate verify failed"

```bash
# Para certificados auto-generados, usar -k en curl
curl -k https://tudominio.com/api/v1/health
```

### Error: "connection refused"

```bash
# Verificar que el contenedor está corriendo
docker ps

# Ver logs
docker logs snippy-api
docker logs snippy-postgres
```

## 🛡️ Configuración de Seguridad

La configuración incluye:

- **TLS 1.2+ only** - Protocolos seguros únicamente
- **Strong ciphers** - Cifrados seguros
- **SCRAM-SHA-256** - Autenticación PostgreSQL segura
- **SSL obligatorio** - Conexiones no cifradas rechazadas
- **Headers de seguridad** - X-Frame-Options, etc.

## 📚 Referencias

- [PostgreSQL SSL Documentation](https://www.postgresql.org/docs/current/ssl-tcp.html)
- [Go TLS Documentation](https://pkg.go.dev/crypto/tls)
- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
