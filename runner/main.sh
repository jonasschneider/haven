#!/bin/sh

set -eux
shopt -s dotglob

spec=$1
name=$2

SOURCE=$(cat $spec | jq --raw-output .source)
repo=$(cat $spec | jq --raw-output .repo)
zbackup_file=$repo/backups/$name
gdrive_folder=$(cat $spec | jq --raw-output .gdrive_folder_id)

archive() {
  cd $SOURCE
  tar c *
}

chunksize=$((1024 * 1000))

archive | go run zbackup-splitter/main.go $chunksize $zbackup_file runner/after_part
archive | zbackup backup $zbackup_file
runner/after_part $zbackup_file
pushd $repo
go run ~/code/haven/gdrivesync/main.go $gdrive_folder
popd
