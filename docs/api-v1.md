# Add new server
curl -i -H "X-Token: NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X PUT -H "Content-Type: application/json" --data @server.json http://127.0.0.1:10000/server/:1001

server.json:
```json
    {
        "Client": "my awesome client2",
        "Addr": ":1001",
        "Auth": {
            "Type": "UserPass",
            "User": "alex",
            "Pass": "testing",
            "IP": null
        },
        "Bytes": null
    }
```
# Get Server Info
curl -i -H "X-Token: NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" http://127.0.0.1:10000/server/:1001



# Add Bandwidth to server
curl -i -H "X-Token: NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X PUT -H "Content-Type: application/json" --data @bytes.json http://127.0.0.1:10000/server/:1001/bandwidth

bytes.json
```json
{
    "Readable": "100 MB"
}
```

# Change Auth on server
curl -i -H "X-Token: NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X PUT -H "Content-Type: application/json" --data @auth.json http://127.0.0.1:10000/server/:1001/auth

auth.json
```json
{
    "Type": "IP",
    "User": "someuser",
    "Pass": "mypass",
    "IP": {
        "127.0.0.1": true,
        "8.8.8.8": true
    }
}
```
