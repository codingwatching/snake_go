# Deployment Guide for Ubuntu Server

This guide explains how to deploy the Snake Game webserver to your Ubuntu server using Docker.

## 1. Prerequisites

Ensure your Ubuntu server has Docker and Docker Compose installed:

```bash
# Update package list
sudo apt-get update

# Install Docker
sudo apt-get install -y docker.io

# Install Docker Compose
sudo apt-get install -y docker-compose-v2

# (Optional) Allow current user to run docker
sudo usermod -aG docker $USER
# Note: Log out and back in for this to take effect
```

## 2. Prepare the Code

You can either clone the repository directly to your server or upload the files.

```bash
# Example: Clone to server
git clone <your-repo-url> snake_go
cd snake_go
```

## 3. Deploy with Docker Compose

Run the following command to build and start the server:

```bash
docker compose up -d --build
```

The server will now be running on port `8080`.

## 4. Accessing the Game

Open your browser and visit: `http://<your-server-ip>:8080`

## 5. (Optional) Configure Nginx with SSL

For a production setup, it's recommended to use Nginx as a reverse proxy with Let's Encrypt (Certbot).

### Example Nginx Config (`/etc/nginx/sites-available/snake`)

```nginx
server {
    listen 80;
    server_name yourdomain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }
}
```

Then enable the site and run certbot:
```bash
sudo ln -s /etc/nginx/sites-available/snake /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
sudo certbot --nginx -d yourdomain.com
```

## 7. Public Testing Checklist 🚀

Before sharing your URL with the public, go through this checklist:

- [ ] **Firewall**: Ensure port `8080` (or `80/443` if using Nginx) is open in your cloud provider's security group (AWS, GCP, DigitalOcean, etc.).
- [ ] **Domain**: If possible, map a domain name. It's much easier for people to remember than an IP.
- [ ] **SSL**: Use Nginx + Certbot (Section 5) to enable `https://`. WebSocket connections are more stable over HTTPS.
- [ ] **Resources**: Ensure your server has at least 1GB of RAM. The AI inference service is efficient but needs steady CPU/RAM.
- [ ] **Monitoring**: Run `docker compose logs -f` during the first few hours of public testing to catch any unexpected errors.

## 8. Maintenance & Backup

- **View Logs:** `docker compose logs -f`
- **Stop Server:** `docker compose down`
- **Update Server:** `git pull && docker compose up -d --build`
- **Database & Records Backup:** All your persistent data is in the `data/` folder.
  - To backup: `cp -r data data_bak`
  - To restore: Just ensure the `data` folder is present and restart.
  - The folder includes `game.db` (users/scores) and `records/` (AI training data).
