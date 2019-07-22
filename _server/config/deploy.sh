#!/bin/bash
echo "Compiling"
go build -o /var/www/shuffle /var/www/shuffle.go
echo "Restarting Service"
sudo systemctl restart shuffle.service
echo "Showing Logs"
sudo journalctl -f -u shuffle.service
