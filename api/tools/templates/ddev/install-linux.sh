#!/usr/bin/env bash
set -euo pipefail

cat <<'STEPS'
DDEV install options (choose one to run manually):

1) One-line install script:
   curl -fsSL https://ddev.com/install.sh | bash

2) Debian/Ubuntu apt repository:
   sudo apt-get update
   sudo apt-get install -y ca-certificates curl gnupg
   sudo install -m 0755 -d /etc/apt/keyrings
   curl -fsSL https://ddev.com/install/ddev.gpg | sudo gpg --dearmor -o /etc/apt/keyrings/ddev.gpg
   sudo sh -c 'echo "deb [signed-by=/etc/apt/keyrings/ddev.gpg] https://ddev.com/apt/ stable main" > /etc/apt/sources.list.d/ddev.list'
   sudo apt-get update
   sudo apt-get install -y ddev mkcert
   mkcert -install

After installing, create or enter a project directory and run:
   ddev config
   ddev start
STEPS
