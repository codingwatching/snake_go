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

## 6. Maintenance

- **View Logs:** `docker compose logs -f`
- **Stop Server:** `docker compose down`
- **Update Server:** `git pull && docker compose up -d --build`
- **Database Backup:** The file `game.db` in the project root is persisted. You can back it up regularly.
