#!/bin/bash

GO="/usr/local/go/bin/go"
SERVICE_FILE="sleepyboi.service"
TIMER_FILE="sleepyboi.timer"
SERVICE_TEMPLATE="sleepyboi.service.template"
TARGET_PATH="$HOME/.config/systemd/user"

function symlink_service {
  echo "Creating symlink to $TARGET_PATH"
  ln -sf "$(pwd)/$SERVICE_FILE" "$TARGET_PATH/$SERVICE_FILE"
  if [ ! -e "$TARGET_PATH/$SERVICE_FILE" ]; then
    ln -sf "$(pwd)/$TIMER_FILE" "$TARGET_PATH/$TIMER_FILE"
  fi
}

function reload_service {
  systemctl --user daemon-reload

  if [ $? != 0 ]; then
      echo "Reloading failed"
      exit 1
  else
      echo "Reload successful"
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
