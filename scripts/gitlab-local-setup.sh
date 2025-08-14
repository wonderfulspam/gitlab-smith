#!/bin/bash
# Local GitLab setup for pipeline rendering

# 1. Start GitLab
docker run -d \
  --name gitlab \
  --hostname localhost \
  -p 8080:80 \
  -p 8443:443 \
  -p 2222:22 \
  --env GITLAB_ROOT_PASSWORD=password123 \
  --env GITLAB_OMNIBUS_CONFIG="
    gitlab_rails['initial_root_password'] = 'password123';
    gitlab_rails['gitlab_shell_ssh_port'] = 2222;
    external_url 'http://localhost:8080';
    # Speed up pipeline creation
    gitlab_rails['pipeline_schedule_worker_cron'] = '*/10 * * * * *';
    # Disable some features for faster startup
    prometheus['enable'] = false;
    alertmanager['enable'] = false;
    grafana['enable'] = false;
  " \
  gitlab/gitlab-ce:latest

echo "Waiting for GitLab to start (this takes ~3-5 minutes)..."
until curl -s http://localhost:8080/users/sign_in | grep -q "GitLab"; do
  echo -n "."
  sleep 10
done

echo -e "\n✅ GitLab is running at http://localhost:8080"
echo "Login: root / password123"