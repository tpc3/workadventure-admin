# WorkAdventure Admin API simple implementation

This is intended to be used internally in our organization.
But you can use this if you want to do.

## Hints to use

Required files
- [config.yaml](./config.template.yaml)
- [woka.json](https://github.com/workadventure/workadventure/raw/refs/heads/develop/play/src/pusher/data/woka.json)
- [companions.json](https://github.com/workadventure/workadventure/blob/develop/play/src/pusher/data/companions.json)

compose.yaml
```
  admin:
    image: ghcr.io/tpc3/workadventure-admin
    volumes:
    - type: bind
      source: ./admin
      target: /data
```

.env
```
ADMIN_API_URL=http://admin:8080
```
