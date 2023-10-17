#!/bin/bash

ADDR=$1
FILE=$2


while [ 1 ]
do 
    RESULT=$(ping -t 1 -c 1 "$ADDR")
    EXIT_CODE=$?

    NOW=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    if [[ $EXIT_CODE -gt 0 ]]; then
        echo "ping failed"
        echo "$NOW, -1" >> $FILE
    else
        RTT=$(echo "$RESULT" | tail -1 | awk '{print $4}' | cut -d '/' -f 2)
        echo "ping succeeded: $RTT"
        echo "$NOW, $RTT" >> $FILE
    fi

    sleep 2
done
