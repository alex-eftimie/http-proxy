# Add new server
```bash
        curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X PUT -H "Content-Type: application/json" --data @server.json http://127.0.0.1:65001/server
```


server.json:
```json
    {
    "email": "eftimie.alexandru@gmail.com",
    "bandwidth": "10.00 GB"
    }
```


It will generate a password, or optionally you can set it

server.json with password:
```json
    {
    "email": "eftimie.alexandru@gmail.com",
    "password": "my password",
    "bandwidth": "10.00 GB"
    }
```

Response:
```json
{
    "AuthToken": "ASUdsduHaKHooytduEhncLNBwbXRpXgauBzcgQYvlQBCUoTTCUHRLBynlqCmXWs",
    "Email": "eftimie.alexandru@gmail.com",
    "Username": "eftimie.alexandru+gmail.com",
    "Password": "cucubau",
    "Host": "127.0.0.1",
    "Port": 1015,
    "ID": "DStQnsRJExLkPbu",
    "Bandwidth": "10.00 GB"
}
```

Take note of ID as it is used for the rest of the requests (i.e. DStQnsRJExLkPbu)

# Get Server Info
```bash
curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" http://127.0.0.1:65001/server/DStQnsRJExLkPbu
```

# Delete Server
```bash
    curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X DELETE -H "Content-Type: application/json" http://127.0.0.1:65001/server/DStQnsRJExLkPbu
```



# Set Bandwidth to server ( Absolute Value )
```bash
    curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X PUT -H "Content-Type: application/json" --data @bytes.json http://127.0.0.1:65001/bandwidth/DStQnsRJExLkPbu
```

bytes.json:
```json
        {
            "Readable": "10.00 GB"
        }
```

# Increase Bandwidth to Server
Add ?add=true
```bash
    curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X PUT -H "Content-Type: application/json" --data @bytes.json http://127.0.0.1:65001/bandwidth/WDlyeoxKtfNzeRl\?add\=true
```

# Disable Bandwidth to Server
```bash
    curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X PUT -H "Content-Type: application/json" --data '{}' http://127.0.0.1:65001/bandwidth/DStQnsRJExLkPbu
```


# Set Time to server ( Absolute Value )
```bash
    curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X PUT -H "Content-Type: application/json" --data @bytes.json http://127.0.0.1:65001/time/DStQnsRJExLkPbu
```

bytes.json:
```json
        {
            "Readable": "10.00 GB"
        }
```

# Increase Time to Server
Add ?add=true
```bash
    curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X PUT -H "Content-Type: application/json" --data @bytes.json http://127.0.0.1:65001/time/WDlyeoxKtfNzeRl\?add\=true
```

# Disable Time to Server
```bash
    curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X PUT -H "Content-Type: application/json" --data '{}' http://127.0.0.1:65001/time/DStQnsRJExLkPbu
```


# Get Server    
```bash
    curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" http://127.0.0.1:65001/threads/DStQnsRJExLkPbu
```

# Change Server Threads
```bash
    curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X PUT -H "Content-Type: application/json" --data @threads.json http://127.0.0.1:65001/threads/DStQnsRJExLkPbu
```



# Get Server Auth
```bash
    curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" http://127.0.0.1:65001/auth/DStQnsRJExLkPbu
```

# Change Server Auth
```bash
    curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X PUT -H "Content-Type: application/json" --data @auth.json http://127.0.0.1:65001/auth/DStQnsRJExLkPbu
```


auth.json with IP Authentication:
```json
    {
        "AuthToken": "XpMooQlTeYXufEZIINXLTExstWjYpRoHvRClxlQXxClzmlsMmJRgPgvzcEWwCcI",
        "   ": "IP",
        "User": "eftimie.alexandru+gmail.com",
        "Pass": "fAg3E45hG6qV0Y2w1",
        "IP": {
            "127.0.0.1": true,
            "127.0.0.2": true
        }
    }
```

```bash
    curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X PUT -H "Content-Type: application/json" --data @auth_user_pass.json http://127.0.0.1:65001/auth/DStQnsRJExLkPbu
```
auth_user_pass.json with User&Pass Authentication:
```json
    {
        "AuthToken": "XpMooQlTeYXufEZIINXLTExstWjYpRoHvRClxlQXxClzmlsMmJRgPgvzcEWwCcI",
        "Type": "UserPass",
        "User": "eftimie.alexandru+gmail.com",
        "Pass": "fAg3E45hG6qV0Y2w1",
        "IP": {
            "127.0.0.1": true,
            "127.0.0.2": true
        }
    }
```

Only IPs that have "ip": true are allowed, ips set with "ip": false are blocked

```bash
    curl -i -H "Authorization: Bearer NOpyVnMKhI4680dxQjdGX6jtUIKq3od3UlQXcQDVrRkCCDuiv2gzDs7ryniuffgwmJO1IqGf" -X PUT -H "Content-Type: application/json" --data @auth_banned_ips.json http://127.0.0.1:65001/auth/DStQnsRJExLkPbu
```
auth_banned_ips.json with IP Authentication and blocked IPs:
```json
    {
        "AuthToken": "XpMooQlTeYXufEZIINXLTExstWjYpRoHvRClxlQXxClzmlsMmJRgPgvzcEWwCcI",
        "Type": "IP",
        "User": "eftimie.alexandru+gmail.com",
        "Pass": "fAg3E45hG6qV0Y2w1",
        "IP": {
            "127.0.0.1": false,
            "127.0.0.2": false,
            "8.8.8.8": true
        }
    }
```
