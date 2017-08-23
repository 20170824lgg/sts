Social tournament service
=========================

This is a demo social tournament service application.

Application can be started using docker-compose script. Web service will liten
on 8009 port, example:

```sh
docker-compose up -d
curl -i 'http://localhost:8009/fund?playerId=P1&points=300'
```
