#!/bin/bash
set -xeu

CURRENT_DIR=$(cd $(dirname $0); pwd)

source ${CURRENT_DIR}/env

${CURRENT_DIR}/create_cluster.sh
${CURRENT_DIR}/init_clean_cloudprovider_config.sh
${CURRENT_DIR}/init_cluster.sh
${CURRENT_DIR}/sync_first_image_to_harbor.sh
${CURRENT_DIR}/init_credential.sh

