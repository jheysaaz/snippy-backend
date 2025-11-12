#!/bin/bash
# Install SSL renewal cron job
# This script sets up automatic SSL certificate renewal

set -e

CRON_SCRIPT="/root/snippy-api/scripts/renew-ssl.sh"
CRON_JOB="0 3 * * * $CRON_SCRIPT >> /var/log/ssl-renewal.log 2>&1"

echo "📅 Setting up SSL renewal cron job..."

# Make renewal script executable
chmod +x "$CRON_SCRIPT"

# Check if cron job already exists
if crontab -l 2>/dev/null | grep -q "renew-ssl.sh"; then
    echo "✅ SSL renewal cron job already exists"
else
    # Add cron job (runs daily at 3 AM)
    (crontab -l 2>/dev/null; echo "$CRON_JOB") | crontab -
    echo "✅ SSL renewal cron job installed"
    echo "   Runs daily at 3:00 AM"
    echo "   Logs to: /var/log/ssl-renewal.log"
fi

# Create log file
touch /var/log/ssl-renewal.log
chmod 644 /var/log/ssl-renewal.log

# Show current cron jobs
echo ""
echo "📋 Current cron jobs:"
crontab -l

echo ""
echo "🔧 To manually renew SSL:"
echo "   sudo $CRON_SCRIPT"
echo ""
echo "📝 To check renewal logs:"
echo "   tail -f /var/log/ssl-renewal.log"
