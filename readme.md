

## Cross-compilation for RPi

`env GOARCH=arm GOARM=5 GOOS=linux go build -o ./deployment/bess_controller_rpi main.go`


## Deployment onto RPi

`scp deployment/waterlilies_config.json pi@waterlillies-rpi:~/bess_controller/config.json`

`scp deployment/bess_controller.service pi@waterlillies-rpi:/lib/systemd/system`