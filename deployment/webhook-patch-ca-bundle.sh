#!/bin/bash

ROOT=$(cd $(dirname $0)/../../; pwd)

set -o errexit
set -o nounset
set -o pipefail

export CA_BUNDLE=$(kubectl config view --raw --flatten | grep certificate-authority-data | awk '{print $2}')
sed -i -e "s|\${CA_BUNDLE}|${CA_BUNDLE}|g" deployment/mutatingwebhook.yaml
