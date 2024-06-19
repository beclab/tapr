#!/bin/bash

arch=$1

if [[ "$arch" == "arm64" ]]; then 
    curl -s https://packagecloud.io/install/repositories/j-white/citusdata-community-arm64/script.deb.sh
else 
    curl -s https://install.citusdata.com/community/deb.sh
fi