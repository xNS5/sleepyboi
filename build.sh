#!/bin/bash

GO="/usr/local/go/bin/go"
SERVICE_FILE="sleepyboi.service"
SERVICE_TEMPLATE="sleepyboi.service.template"
TARGET_PATH="/etc/systemd/system/$SERVICE_FILE"

function symlink_service {
  echo "Creating symlink to /etc/systemd/system for $SERVICE_FILE..."
  sudo ln -sf "$(pwd)/$SERVICE_FILE" "$TARGET_PATH"
}

function reload_service {
  sudo systemctl daemon-reload

  if [ $? != 0 ]; then
      echo "Reloading failed"
      exit 1
  else
      echo "Reload successful"
  fi
}

if [ "$EUID" -ne 0 ]; then
  echo "This script must be run with elevated permissions (as root)."
  exit 1
fi

if [ ! -f "$SERVICE_FILE" ]; then
  echo "Service file $SERVICE_FILE not found in the current directory."
  echo "Creating $SERVICE_FILE from template..."
  cp $SERVICE_TEMPLATE $SERVICE_FILE
  sed -i "s|ExecStart=|ExecStart=$GOBIN/geoboi|" $SERVICE_FILE
fi

echo "Building Geoboi..."

$GO build -o "$GOBIN/"

if [ $? != 0 ]; then
  echo "Build failed"
  exit 1
fi

echo "Geoboi built successfully."

symlink_service

echo "Reloading systemd daemon..."
reload_service
