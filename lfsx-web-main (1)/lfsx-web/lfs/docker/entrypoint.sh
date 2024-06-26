#!/bin/sh

# Expose variables
export RUNNING_IN_KUBERNETES=true
export _JAVA_OPTIONS=-Duser.home=/home/oracle

# Starting sway
sway &

# Give sway some time to startup
sleep 0.5

# Start guacamole in foreground
/opt/guacamole/sbin/guacd -b 0.0.0.0 -L debug -f &

# Start a new firefox instance
# firefox --new-instance &

/opt/go-lfs/go-lfs
