
# Flux

Flux is a collection of tools to control batteries within a microgrid.
It includes the `bess-controller` which is a Go service that runs on-site, and a set of database migrations for creating the required Supabase support structures.


### Migrations

To deploy migrations, first setup the appropriate sqitch secrets file (see `sqitch_secrets.conf.example` and LastPass), then for the MGF environmnet :
- `sqitch status --target timescale-mgf`
- `SQITCH_USER_CONFIG=sqitch_secrets_mgf.conf sqitch deploy --target timescale-mgf`
