#!/bin/bash

echo "Starting tunnel to 192.168.8.110 on 1502..."
ssh -p 6788 -f -N -L 1502:192.168.8.110:502 pi@wlce-robustel

# echo "Starting tunnel to 192.168.8.111 on 1503..."
# ssh -f -N -L 1503:192.168.8.111:502 pi@100.99.152.30

# echo "Starting tunnel to 192.168.10.64 on 1504..."
# ssh -f -N -L 1504:192.168.10.64:502 pi@100.99.152.30
