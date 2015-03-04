#!/bin/sh

set -eux

SOURCE=$1
TARGET=$2

# TODO: notification
archive() {
  tar c $SOURCE
}

chunksize=$((1024 * 1000))

archive | go run zbackup-splitter/main.go $chunksize $TARGET runner/after_part
archive | zbackup backup $TARGET
runner/after_part $TARGET
