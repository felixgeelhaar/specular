#!/bin/bash
# Demo recording script for asciinema

cd /tmp
rm -rf specular-demo
mkdir specular-demo
cd specular-demo

clear

echo -e "\033[36m$\033[0m specular init --yes --no-detect"
sleep 0.5
specular init --yes --no-detect
sleep 1

echo ""
echo -e "\033[36m$\033[0m specular doctor"
sleep 0.5
specular doctor
sleep 1

echo ""
echo -e "\033[36m$\033[0m specular policy validate"
sleep 0.5
specular policy validate
sleep 1

echo ""
echo -e "\033[36m$\033[0m specular drift check"
sleep 0.5
specular drift check
sleep 2
