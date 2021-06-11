# Components

The extensible components init for multipurpose. Based on some best practices in production.

## Configure

Configure defaults is `config.yml`.

```yaml
mysql:
  dsn: user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local
redis:
  addr: 127.0.0.1:6379
  username:
  password:
  db: 0
nats:
  server:
  name:
```
