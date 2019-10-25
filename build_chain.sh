#!/usr/bin/env bash
echo "##Start building Incognito Chain"
rm -rfv bootnode/bootnode
cd bootnode
go build -o bootnode
echo "##build bootnode success"
sleep 1

cd ..
rm -rfv data/*
echo "##deleted chain data"

rm -rfv incognito
go build -o incognito
echo "##build incognito binary success"
sleep 1

echo "##Incognito Chain is ready to start"
