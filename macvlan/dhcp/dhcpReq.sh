#!/bin/bash

DHCP_CLIENT=$1
NSPID=$2
CONTAINER_IFNAME="eth1"
GUESTNAME=$3



[ $DHCP_CLIENT = "udhcpc"  ] && $DHCP_CLIENT -qi $CONTAINER_IFNAME -x hostname:$GUESTNAME
if [ $DHCP_CLIENT = "dhclient"  ]
then
	echo "dhclient exec "
    # kill dhclient after get ip address to prevent device be used after container close          
    $DHCP_CLIENT -pf "/var/run/dhclient.$NSPID.pid" $CONTAINER_IFNAME
    kill "$(cat "/var/run/dhclient.$NSPID.pid")"
    rm "/var/run/dhclient.$NSPID.pid"
fi
[ $DHCP_CLIENT = "dhcpcd"  ] && $DHCP_CLIENT -q $CONTAINER_IFNAME -h $GUESTNAME