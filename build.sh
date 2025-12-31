#!/bin/bash

GO="$(which go)"
SERVICE_FILE="sleepyboi.service"
TIMER_FILE="sleepyboi.timer"
SERVICE_TEMPLATE="sleepyboi.service.template"
TARGET_PATH="$HOME/.config/systemd/user"
STATE_PATH="$HOME/.local/lib/sleepyboi"
STATE_FILE="$STATE_PATH/sleepyboi.json"

function symlink_service {
  if [ ! -e "$TARGET_PATH/$SERVICE_FILE" ]; then
    echo "Creating symlink to $TARGET_PATH"
    ln -sf "$(pwd)/$SERVICE_FILE" "$TARGET_PATH/$SERVICE_FILE"
    ln -sf "$(pwd)/$TIMER_FILE" "$TARGET_PATH/$TIMER_FILE"
  else
    echo "File already exists. Skipping symlink..."
  fi
}

function reload_service {
  systemctl --user daemon-reload

  if [ $? != 0 ]; then
      echo "Reloading daemon failed"
      exit 1
  else
      echo "Reloading daemon successful"
  fi

  systemctl --user enable sleepyboi.timer
 
  if [ $? != 0 ]; then
      echo "Enabling sleepyboi failed"
      exit 1
  else
      echo "Enabling sleepyboi successful"
  fi
}

# if [ "$EUID" -ne 0 ]; then
#   echo "This script must be run with elevated permissions (as root)."
#   exit 1
# fi

if [ ! -f "$SERVICE_FILE" ]; then
  echo "Service file $SERVICE_FILE not found in the current directory."
  echo "Creating $SERVICE_FILE from template..."
  cp $SERVICE_TEMPLATE $SERVICE_FILE
  sed -i "s|ExecStart=|ExecStart=$GOBIN/sleepyboi|" $SERVICE_FILE
fi

if [ ! -f "$STATE_FILE" ]; then 
  echo "State file not found"
  echo "Creating state file..."
  mkdir "$STATE_PATH" && touch "$STATE_FILE"
  echo "State file created"
fi

echo "Building Sleepyboi..."

$GO build -o "$GOBIN/"

if [ $? != 0 ]; then
  echo "Build failed"
  exit 1
fi

echo "Sleepyboi built successfully."

symlink_service

echo "Reloading systemd daemon..."

reload_service
