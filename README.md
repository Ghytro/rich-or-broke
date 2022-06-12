# Rich or Broke (Alfa Bank test task)
## Brief description
Mini-project, a service that returns a gif corresponding to the currencies rate change. If the rate of currency raised, compared to yesterday, the service returns a random "rich" gif, otherwise returns "broke" gif. All of the requests are handled on the endpoint of the following format:

```https://rich-or-broke.org/api/diff/{value_id}```

```value_id``` - a parameter with id of the currency. Full list of currencies ids according to ISO 4217 can be found [here](https://en.wikipedia.org/wiki/ISO_4217#Active_codes)

## Configuration
Configuration file is stored in [config/config.json](https://github.com/Ghytro/rich-or-broke/tree/main/config/config.json). The configuration file must be of the following format:
```json
{
    "port": 8080,
    "openexchange_api_token": "open exchange api token",
    "openexchange_base_url": "https://openexchangerates.org/api/",
    "tenor_api_token": "tenor api token",
    "tenor_base_url": "https://g.tenor.com/v1/",
    "redis_client_options": {
        "db": 0,
        "addr": "127.0.0.1:6379",
        "password": ""
    },
    "base_currency_id": "USD"
}
```
All of the config parameters are necessary for the service.

## How to launch
### Build from scratch (needs Go compiler and Redis server to be installed)
- Install Redis: [installation guide](https://redis.io/docs/getting-started/)
- Build the service executable
    1. Clone this repository on your machine via https: ```git clone https://github.com/Ghytro/rich-or-broke.git```
    2. Go to the root of the module and build the executable (you can specify the name of the executable by adding ```-o``` flag): ```cd rich-or-broke && go build -o executable```
    3. Specify configuration parameters in config/config.json
    4. Run the executable when your Redis is ready: ```./executable```
