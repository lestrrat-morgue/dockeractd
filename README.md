dockeractd
==========

React to docker daemon's events

# WHAT IS THIS?

Think "Serf for Docker" (I promise I will put more documentation later)

```
$ dockeractd -exec=/path/to/handler
```

handler can be anything. it received a JSON string from the standard input.

Example:

```bash
#!/bin/bash

ip=$(jq -r .Container.NetworkSettings.IPAddress)
if [ "$ip" != "null" ]; then
    echo "Container's IP is $ip"
fi
```

This only works for "start" events. Obviously, you can create handlers
that filter out other event types
