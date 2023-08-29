#!/bin/bash

echo "Starting tunnel to 192.168.8.69 on 1502..."
ssh -f -N -L 1502:192.168.8.69:502 pi@waterlillies-rpi

echo "Starting tunnel to 192.168.8.78 on 1503..."
ssh -f -N -L 1503:192.168.8.78:502 pi@waterlillies-rpi
