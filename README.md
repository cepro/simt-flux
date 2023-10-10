
# bess-controller

A Go service to control batteries and associated metering. The service is currently compiled and run as a systemd service, to build it run: `go build main.go -o controller`.

The high-level architecture of the program is shown in the following illustration: ![high_level](docs/high_level.png)

## Configuration / Env Vars

Configuration of the devices, IP address, and polling intervals etc is via a JSON configuration file, use the `-f` command line flag to specify the file. An example configuration file is `./deployment/waterlilies_config.json`.

Secrets are supplied by environment variable:
- `SUPABASE_ANON_KEY`: The supabase anon JWT
- `SUPABASE_USER_KEY`: A supabase JWT for the user role 


## Testing

To run unit tests:

`go test ./...`


## Cross-compilation

To compile a binary that will run on a 32bit ARM processor like the RPi:

`env GOARCH=arm GOARM=5 GOOS=linux go build -o ./deployment/bess_controller_rpi main.go`


## Deployment onto RPi

To deploy the service and config files onto a RPi over SSH:

`scp deployment/bess_controller_rpi pi@waterlillies-rpi:~/bess_controller/bess_controller_rpi`

`scp deployment/waterlilies_config.json pi@waterlillies-rpi:~/bess_controller/config.json`

`scp deployment/bess_controller.service pi@waterlillies-rpi:/lib/systemd/system/`

