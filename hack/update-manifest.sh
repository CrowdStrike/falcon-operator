#!/bin/bash

set -e -o pipefail

readme="${2}"
last_banner_line=$(cat $readme | grep --line-number '\[!\[' | tail -n 1 | sed 's/:.*$//g')
let "first_readme_line=last_banner_line+1"
export content="$(tail -n +${first_readme_line} ${readme} | sed 's/(docs\//(https:\/\/github.com\/CrowdStrike\/falcon-operator\/tree\/main\/docs\//g' )"
yq -i e '.spec.description=strenv(content)' "${1}"
operator-sdk generate kustomize manifests -q
