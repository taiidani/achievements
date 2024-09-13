# Achievement Report

The Achievement Report is a webapp for tracking the progress of achievements in games. It can be used to inspect and examine the achievements, trophies, etc of various game networks in a single view.

## Currently Supported Services

* Steam

## Contributing

To run the app locally, you will need:

* [Go](https://go.dev/) - Minimum version specified in the [go.mod](go.mod) file.
* (Optional) [Mise](https://mise.jdx.dev/) for setting env vars and dependencies.
* (Optional) [Docker and Docker Compose](https://docs.docker.com/desktop/), for running Redis.

### Mise

The Mise CLI tool is being used for dependency management and environment variable distribution in this repository. The below examples will assume that you have Mise installed. Mise is used to load the `.env` file [automatically](.mise.toml), but any method of managing of these environment variables will also work.


 If you do not have or want Mise set up you can still set your environment up manually.

The `.env` file being set up in the below sections is normally exported to your shell automatically, but you can instead export these manually with `export KEY="value"`.

### Running Locally

The application is run through Go and uses environment variables to configure its behavior. The environment variables in use are:

* (Required) `STEAM_KEY` - A Steam API Key for communicating with the Steam API. A key can be provisioned [here](https://steamcommunity.com/dev/apikey).
* (Required) `PORT` - The port to host the webapp on.
* (Optional) `DEV` - If set to "true", will disable caching of HTML templates and improve iteration.

To run the application, compile and execute it via Go:

```sh
# Set required env vars
echo "STEAM_KEY = \"${STEAM_KEY}\"" > .env
echo "PORT = \"80\"" >> .env
echo "DEV = \"true\"" >> .env

go run main.go
```

#### Caching

By default the webapp will use an in-memory cache. There are two other caches available:

* File - Stores cache in the filesystem under a `_cache` folder. Currently unused.
* Redis - Stores cache in a Redis instance, with TLS and authentication available. This is the Production configuration.

The Redis configuration may be used locally. To do this, run a Docker container for the Redis service and set the required environment variables:

```sh
docker compose up -d

echo "REDIS_ADDR = \"$(docker compose port redis 6379)\"" >> .env

go run main.go
```

### Deploying

Deployment and hosting is provided by [@taiidani](https://github.com/taiidani). Please reach out if you have questions about deployment and hosting configurations.
