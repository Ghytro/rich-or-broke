# Rich or Broke (Alfa Bank test task)
Read this in other languages: [Русский](https://github.com/Ghytro/rich-or-broke/blob/main/README.ru.md)
## Brief description
Mini-project, a service that returns a gif corresponding to the currencies rate change. If the rate of currency raised, compared to yesterday, the service returns a random "rich" gif, otherwise returns "broke" gif. All of the requests are handled on the endpoint of the following format:

```https://rich-or-broke.org/api/diff/{value_id}```

```value_id``` - a parameter with id of the currency. Full list of currencies ids according to ISO 4217 can be found [here](https://en.wikipedia.org/wiki/ISO_4217#Active_codes)

## Tech stack & implementation details
The service itself is written in Go, Redis is used for caching requests to external APIs. The service can work without Redis, but responses will be sufficiently slower because of the requests to the external services. Some of the requests are performed in asynchronous way, but it is still slower than getting requests cache from Redis.

Service is able to work without Redis because of the implemented health checker and circuit breaker for Redis. If Redis is not responding, service will fallback to external APIs. Redis health is checked once a minute.

Currencies rates from [openexchangerates](https://openexchangerates.org/) are updated in cache once in 10 minutes, cached gifs from [tenor](https://tenor.com/) are updated once a day.

## Configuration
Configuration file is stored in [config/config.json](https://github.com/Ghytro/rich-or-broke/tree/main/config/config.json). The configuration file must be of the following format:
```json
{
    "verbose": true,
    "port": 8080,
    "openexchange_api_token": "open exchange api token",
    "openexchange_base_url": "https://openexchangerates.org/api/",
    "tenor_api_token": "tenor api token",
    "tenor_base_url": "https://g.tenor.com/v1/",
    "tenor_media_storage_base_url": "https://media.tenor.com/images/",
    "tenor_search_query_limit": 100,
    "redis_client_options": {
        "db": 0,
        "addr": "127.0.0.1:6379",
        "password": ""
    },
    "base_currency_id": "USD"
}
```
Precense of all the config parameters is necessary to run the service.

## How to launch
### (Recommended) Build docker image and launch in container (needs Docker to be installed)
1. Install Redis Docker image: ```docker pull redis```
2. Create Docker subnet: ```docker network create mynet --subnet=172.18.0.0/16```. You can specify any address/mask and name of network you want. In this example network named 'mynet' with address 172.18.0.0/16 will be used.
3. Start container with Redis in created subnet: ```docker run -d -p 6379:6379 --network mynet --ip 172.18.0.23 --name rich_or_broke_cache --rm redis```
4. Clone git repository with application to your machine via https: ```git clone https://github.com/Ghytro/rich-or-broke.git```
5. Go to the root of repository and specify Redis server address of the container with Redis you run in step 3 in the application configuration file (key "addr" in object "redis_client_options").
6. Specify other parameters in the configuration file, like port you want your server to listen on (8080 by default) and api tokens for openexchange and tenor.
7. Build Docker image of the application: ```docker build -t rich_or_broke .```
8. Start container with application in created subnet: ```docker run -it -p 8080:8080 --name rich_or_broke --network mynet --ip 172.18.0.22 --rm rich_or_broke```
9. Server logs will be printed in stdout if "verbose" was enabled in config.
10. Press Ctrl+C to stop the service, then stop the container with Redis when you're done, it will be removed automatically if you specified ```--rm``` flag when launching the container: ```docker stop rich_or_broke_cache```

### Build from scratch (needs Go compiler and Redis server to be installed)
- Install and start Redis server: [installation guide](https://redis.io/docs/getting-started/)
- Build and launch the service executable
    1. Clone git repository to your machine via https: ```git clone https://github.com/Ghytro/rich-or-broke.git```
    2. Go to the root of the module and build the executable (you can specify the name of the executable by adding ```-o``` flag): ```cd rich-or-broke && go build -o executable```
    3. Specify configuration parameters in config/config.json
    4. Run the executable when your Redis is ready: ```./executable```
