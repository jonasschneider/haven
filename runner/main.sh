#!/bin/sh

set -eux

spec=$1

SOURCE=$(cat $spec | jq --raw-output .source)
TARGET=$(cat $spec | jq --raw-output .z_backupfile)
gdrive_folder=$(cat $spec | jq --raw-output .gdrive_folder_id)

# TODO: notification
archive() {
  tar c $SOURCE
}

chunksize=$((1024 * 1000))

archive | go run zbackup-splitter/main.go $chunksize $TARGET runner/after_part
archive | zbackup backup $TARGET
runner/after_part $TARGET
pushd $(dirname $TARGET)/..
go run ~/code/haven/gdrivesync/main.go $gdrive_folder
popd
