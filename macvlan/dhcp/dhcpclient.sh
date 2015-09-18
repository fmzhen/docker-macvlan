#!/bin/bash

# Check for first available dhcp client
DHCP_CLIENT_LIST="udhcpc dhcpcd dhclient"
for CLIENT in $DHCP_CLIENT_LIST; do                                                 
    which $CLIENT >/dev/null && {
        DHCP_CLIENT=$CLIENT
        break
    }
done

[ -z $DHCP_CLIENT ] && {
echo "You asked for DHCP; but no DHCP client could be found."
exit 1
}

echo $DHCP_CLIENT