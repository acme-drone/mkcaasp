#cloud-config

# set locale
locale: en_US.UTF-8

# set timezone
timezone: Europe/Warsaw

# set as admin node
suse_caasp:
  role: cluster
  admin_node: ${admin_address}

final_message: "The system is finally up, after $UPTIME seconds"
