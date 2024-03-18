### GeoLog2MQTT

Tail server log, get geocoordinates with Maxminds GeoLite2 database and publish results to a MQTT broker.

```
usage: run [-h|--help] -l|--log_file "<value>" -g|--geodb_file "<value>"
           -m|--mqtt_server "<value>" [-p|--mqtt_port <integer>] [-u|--username
           "<value>"] [-P|--password "<value>"] [-t|--topic "<value>"]
           [-T|--throttle_duration <integer>]

Arguments:

  -h  --help               Print help information
  -l  --log_file           log file to tail
  -g  --geodb_file         geolite db to use
  -m  --mqtt_server        mqtt server to use
  -p  --mqtt_port          mqtt port to use. Default: 1884
  -u  --username           mqtt username to use
  -P  --password           mqtt password to use
  -t  --topic              mqtt topic to use. Default: location
  -T  --throttle_duration  throttle in seconds. Default: 5
```

Example without credentials:

    go run . -l /var/log/nginx/mysite.log -g ~/tmp/GeoLite2-City.mmdb -m localhost
