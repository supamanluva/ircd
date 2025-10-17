# IRC Server - Production Deployment Guide

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Docker Deployment](#docker-deployment)
3. [systemd Deployment](#systemd-deployment)
4. [TLS Configuration](#tls-configuration)
5. [Firewall Configuration](#firewall-configuration)
6. [Monitoring](#monitoring)
7. [Backup & Recovery](#backup--recovery)
8. [Security Hardening](#security-hardening)
9. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### System Requirements
- **OS**: Linux (Ubuntu 20.04+, Debian 11+, or RHEL/CentOS 8+)
- **CPU**: 1 core minimum (2+ recommended)
- **RAM**: 512MB minimum (1GB+ recommended)
- **Disk**: 1GB minimum
- **Network**: Open ports 6667 (IRC) and 7000 (TLS)

### Software Requirements
- Go 1.21+ (for building from source)
- Docker & Docker Compose (for container deployment)
- systemd (for service deployment)
- certbot (for Let's Encrypt certificates)

---

## Docker Deployment

### Quick Start

1. **Clone and build**:
```bash
git clone https://github.com/supamanluva/ircd.git
cd ircd
make build
```

2. **Generate certificates**:
```bash
./generate_cert.sh
```

3. **Configure server**:
```bash
nano config/config.yaml
```

4. **Start with Docker Compose**:
```bash
docker-compose up -d
```

5. **Check status**:
```bash
docker-compose ps
docker-compose logs -f ircd
```

### Docker Compose Profiles

**Basic deployment** (IRC server only):
```bash
docker-compose up -d
```

**With monitoring** (Prometheus + Grafana):
```bash
docker-compose --profile monitoring up -d
```

Access Grafana at http://localhost:3000 (admin/admin)

### Docker Commands

```bash
# Start server
docker-compose up -d

# Stop server
docker-compose down

# Restart server
docker-compose restart ircd

# View logs
docker-compose logs -f ircd

# Update server
git pull
make build
docker-compose build
docker-compose up -d

# Backup data
docker-compose exec ircd tar czf /backup/ircd-$(date +%Y%m%d).tar.gz /app/config /app/logs
```

---

## systemd Deployment

### Installation

1. **Build server**:
```bash
make build
```

2. **Install with script**:
```bash
sudo ./deploy/install.sh
```

3. **Configure**:
```bash
sudo nano /opt/ircd/config/config.yaml
```

4. **Generate TLS certificates** (if needed):
```bash
cd /opt/ircd
sudo -u ircd ./generate_cert.sh
```

5. **Start service**:
```bash
sudo systemctl enable ircd
sudo systemctl start ircd
```

### systemd Commands

```bash
# Start server
sudo systemctl start ircd

# Stop server
sudo systemctl stop ircd

# Restart server
sudo systemctl restart ircd

# Check status
sudo systemctl status ircd

# View logs
sudo journalctl -u ircd -f

# Enable auto-start
sudo systemctl enable ircd

# Disable auto-start
sudo systemctl disable ircd
```

### Manual Installation

If you prefer manual installation:

```bash
# Create user
sudo useradd --system --home /opt/ircd --shell /bin/false ircd

# Create directories
sudo mkdir -p /opt/ircd/{bin,config,certs,logs}

# Copy files
sudo cp bin/ircd /opt/ircd/bin/
sudo cp config/config.yaml /opt/ircd/config/

# Set permissions
sudo chown -R ircd:ircd /opt/ircd
sudo chmod 755 /opt/ircd/bin/ircd

# Install service
sudo cp deploy/ircd.service /etc/systemd/system/
sudo systemctl daemon-reload
```

---

## TLS Configuration

### Self-Signed Certificates (Testing)

```bash
./generate_cert.sh
```

This creates:
- `certs/server.crt` - Certificate
- `certs/server.key` - Private key

### Let's Encrypt (Production)

1. **Install certbot**:
```bash
# Ubuntu/Debian
sudo apt install certbot

# RHEL/CentOS
sudo yum install certbot
```

2. **Get certificate**:
```bash
sudo certbot certonly --standalone -d irc.yourdomain.com
```

3. **Update config.yaml**:
```yaml
server:
  tls:
    enabled: true
    port: 7000
    cert_file: "/etc/letsencrypt/live/irc.yourdomain.com/fullchain.pem"
    key_file: "/etc/letsencrypt/live/irc.yourdomain.com/privkey.pem"
```

4. **Set permissions** (for systemd):
```bash
# Allow ircd user to read certificates
sudo setfacl -R -m u:ircd:rx /etc/letsencrypt/live
sudo setfacl -R -m u:ircd:rx /etc/letsencrypt/archive
```

5. **Auto-renewal**:
```bash
# Test renewal
sudo certbot renew --dry-run

# Setup auto-renewal (cron)
echo "0 3 * * * certbot renew --quiet && systemctl reload ircd" | sudo tee -a /etc/crontab
```

---

## Firewall Configuration

### UFW (Ubuntu/Debian)

```bash
# Allow IRC ports
sudo ufw allow 6667/tcp comment 'IRC plaintext'
sudo ufw allow 7000/tcp comment 'IRC TLS'

# Enable firewall
sudo ufw enable

# Check status
sudo ufw status
```

### firewalld (RHEL/CentOS)

```bash
# Allow IRC ports
sudo firewall-cmd --permanent --add-port=6667/tcp
sudo firewall-cmd --permanent --add-port=7000/tcp

# Reload
sudo firewall-cmd --reload

# Check
sudo firewall-cmd --list-ports
```

### iptables

```bash
# Allow IRC ports
sudo iptables -A INPUT -p tcp --dport 6667 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 7000 -j ACCEPT

# Save rules
sudo iptables-save > /etc/iptables/rules.v4
```

---

## Monitoring

### Logs

**Docker**:
```bash
docker-compose logs -f ircd
```

**systemd**:
```bash
# Real-time
sudo journalctl -u ircd -f

# Last 100 lines
sudo journalctl -u ircd -n 100

# Since boot
sudo journalctl -u ircd -b

# Errors only
sudo journalctl -u ircd -p err
```

### Health Checks

**TCP connection test**:
```bash
nc -vz localhost 6667
nc -vz localhost 7000
```

**TLS test**:
```bash
openssl s_client -connect localhost:7000 -brief
```

**IRC command test**:
```bash
echo -e "NICK testbot\nUSER test 0 * :Test\nQUIT" | nc localhost 6667
```

### Metrics (with Prometheus)

Access Prometheus at http://localhost:9090

Key metrics to monitor:
- Connection count
- Message rate
- Channel count
- Error rate
- Memory usage
- CPU usage

---

## Backup & Recovery

### What to Backup

```bash
/opt/ircd/config/      # Configuration
/opt/ircd/certs/       # TLS certificates
/opt/ircd/logs/        # Logs (optional)
```

### Backup Script

```bash
#!/bin/bash
BACKUP_DIR="/backup/ircd"
DATE=$(date +%Y%m%d-%H%M%S)

mkdir -p $BACKUP_DIR
tar czf $BACKUP_DIR/ircd-$DATE.tar.gz \
    /opt/ircd/config \
    /opt/ircd/certs

# Keep last 7 days
find $BACKUP_DIR -name "ircd-*.tar.gz" -mtime +7 -delete
```

### Recovery

```bash
# Stop server
sudo systemctl stop ircd

# Extract backup
sudo tar xzf /backup/ircd-20251017.tar.gz -C /

# Restore permissions
sudo chown -R ircd:ircd /opt/ircd

# Start server
sudo systemctl start ircd
```

---

## Security Hardening

### System Level

1. **Run as non-root user** ✅ (ircd user)
2. **Disable unnecessary services**
3. **Keep system updated**:
```bash
sudo apt update && sudo apt upgrade -y
```

4. **Configure fail2ban** (optional):
```bash
sudo apt install fail2ban
```

### Application Level

1. **Enable TLS only** (disable plaintext):
```yaml
server:
  port: 0  # Disable plaintext
  tls:
    enabled: true
    port: 7000
```

2. **Rate limiting** (already enabled):
```yaml
server:
  rate_limit:
    enabled: true
    messages_per_second: 5
    burst: 10
```

3. **Reduce max clients** (if needed):
```yaml
server:
  max_clients: 100  # Adjust based on capacity
```

### Network Level

1. **Use reverse proxy** (optional - for DDoS protection):
```bash
# HAProxy, nginx, or Cloudflare
```

2. **IP whitelisting** (for private servers):
```bash
# iptables
sudo iptables -A INPUT -p tcp --dport 7000 -s 192.168.1.0/24 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 7000 -j DROP
```

---

## Troubleshooting

### Server Won't Start

1. **Check logs**:
```bash
sudo journalctl -u ircd -n 50
```

2. **Check config**:
```bash
/opt/ircd/bin/ircd -config /opt/ircd/config/config.yaml
```

3. **Check ports**:
```bash
sudo netstat -tlnp | grep -E '(6667|7000)'
```

4. **Check permissions**:
```bash
ls -la /opt/ircd/
sudo -u ircd /opt/ircd/bin/ircd -config /opt/ircd/config/config.yaml
```

### TLS Issues

1. **Verify certificates**:
```bash
openssl x509 -in /opt/ircd/certs/server.crt -text -noout
openssl rsa -in /opt/ircd/certs/server.key -check
```

2. **Test TLS connection**:
```bash
openssl s_client -connect localhost:7000
```

### Connection Issues

1. **Check firewall**:
```bash
sudo ufw status
sudo iptables -L -n
```

2. **Check listening ports**:
```bash
sudo ss -tlnp | grep ircd
```

3. **Test from external**:
```bash
telnet your-server-ip 6667
openssl s_client -connect your-server-ip:7000
```

### Performance Issues

1. **Check resource usage**:
```bash
# CPU/Memory
top -p $(pgrep ircd)

# Connections
ss -s
netstat -an | grep ESTABLISHED | wc -l
```

2. **Check limits**:
```bash
# File descriptors
ulimit -n
cat /proc/$(pgrep ircd)/limits
```

3. **Adjust rate limits**:
```yaml
server:
  rate_limit:
    messages_per_second: 3  # Lower = stricter
    burst: 5
```

---

## Production Checklist

- [ ] Build server with `make build`
- [ ] Configure `config/config.yaml`
- [ ] Generate/install TLS certificates
- [ ] Configure firewall
- [ ] Install and enable systemd service
- [ ] Test IRC connections (plaintext & TLS)
- [ ] Setup log rotation
- [ ] Configure backups
- [ ] Setup monitoring (optional)
- [ ] Document admin procedures
- [ ] Test disaster recovery

---

## Support

- **Documentation**: See [README.md](../README.md) and [PROJECT_STATUS.md](../PROJECT_STATUS.md)
- **Issues**: GitHub Issues
- **Logs**: `/opt/ircd/logs/` or `journalctl -u ircd`

---

**Congratulations! Your IRC server is now production-ready!** 🎉
