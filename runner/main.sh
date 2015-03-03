#!/bin/sh

set -eux

SOURCE=$1
TARGET=$2

# TODO: notification
archive() {
  tar c $SOURCE
}

chunksize=1025

archive | go run zbackup-splitter/main.go $chunksize $TARGET
archive | zbackup backup $TARGET
