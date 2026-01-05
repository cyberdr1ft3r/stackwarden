# Portainer CE (LTS) stack

Run Portainer CE using Docker Compose.

## Steps

1) (Optional) adjust the ports in `.env` (defaults: 9443 for HTTPS UI, 8000 for Edge/agent listeners).  
2) Create the data volume (if it doesn't already exist):
   ```sh
   docker volume create portainer_data
   ```
3) Start the stack:
   ```sh
   docker compose up -d
   ```
4) Open the UI at `https://<host>:9443`.
