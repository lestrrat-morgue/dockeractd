dockeractd
==========

React to docker daemon's events

# WHAT IS THIS?

There's always something you'd like to trigger upon bringing up a new container.
There's [Consul](https://consul.io) and the likes to do such things, but these
are ... big. They make sense when you want to scale, but there are times
when all you really need to do is just updating that one local file or
something like that.

`dockeractd` is basically the answer for that. It just listens for events
emitted by the local docker daemon, and calls the registered handler. That's
all. Nothing more. No networks to configure, no ports to open, no agents to
install.

A typical usage would be something like this:

```
$ dockeractd -exec=/path/to/handler
```

This registered `dockeracd` to listen to `/var/run/docker.sock`, and when
an event arrives, the handler specified in `-exec` will be executed.

The handler can be anything: a shell script, a compiled binary, whatever.
It receives a JSON string from the standard input, which consists of 
two top level keys:

```json
{
  "Event": { ... },
  "Container": { ... }
}
```

The `Container` key may be null depending on the event type. Available events
depend on the API version that your docker daemon supports.

An example handler shell script my look like this:

```bash
#!/bin/bash

WORKFILE=/tmp/work.json
cat > $WORKFILE

event=$(jq -r .Event.Status $WORKFILE)
if [ "$event" != "start" ]
    rm -f $WORKFILE
    exit 0
fi

ip=$(jq -r .Container.NetworkSettings.IPAddress $WORKFILE)
if [ "$ip" != "null" ]; then
    echo "Container's IP is $ip"
fi

rm -f $WORKFILE
```

This just prints out the IP address of the newly started container.
