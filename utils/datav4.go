package utils

type MKCaaSPCfg struct {
	Platform    string  `json: "platform"`
	Deploy      string  `json: "deploy"`
	Vmware      *VMWare `json: "vmware"`
	Skubaroot   string  `json: "skubaroot"`
	ClusterName string  `json: "clustername"`
}

type VMWare struct {
	GOVC_URL                     string
	GOVC_USERNAME                string
	GOVC_PASSWORD                string `json: "GOVC_PASSWORD"`
	GOVC_INSECURE                int
	VSPHERE_SERVER               string `json: "VSPHERE_SERVER"`
	VSPHERE_USER                 string
	VSPHERE_PASSWORD             string
	VSPHERE_ALLOW_UNVERIFIED_SSL bool
}

type TFOutput struct {
	IP_Load_Balancer *TFTag `json: ip_load_balancer`
	IP_Masters       *TFTag `json: ip_masters`
	IP_Workers       *TFTag `json: ip_workers`
}

type TFOutput_vmware struct {
	IP_Load_Balancer *TFTag `json: ip_load_balancer`
	IP_Masters       *TFTag `json: ip_masters`
	IP_Workers       *TFTag `json: ip_workers`
}

type TFOutput_openstack struct {
	IP_Load_Balancer *TFTagLoadBalancer `json: ip_load_balancer`
	IP_Masters       *TFTag             `json: ip_masters`
	IP_Workers       *TFTag             `json: ip_workers`
}

type TFTagLoadBalancer struct {
	Sensitive bool   `json: sensitive`
	Type      string `json: type`
	Value     string `json: value`
}

type TFTag struct {
	Sensitive bool     `json: sensitive`
	Type      string   `json: type`
	Value     []string `json: value`
}

type TFTag_variable struct {
	Sensitive bool          `json: sensitive`
	Type      string        `json: type`
	Value     StringOrSlice `json: value`
}

type StringOrSlice interface{}

//type ClusterCheck map[string]Node

type SkubaCluster struct {
	ClusterName string
	Diagnosis   map[string]Node
	TF_ostack   *TFOutput_openstack
	TF_vmware   *TFOutput_vmware
	Testdir     string
	Setup       Setup `json: setup`
}

type Node struct {
	IP         string
	NodeName   string
	K8sName    string
	Role       string
	Username   string
	Network    bool
	Port22     bool
	SSH        bool
	ContHealth bool
	PackHealth bool
	RepoHealth bool
	Services   bool
	Systemd    Systemd
	K8sHealth  *K8s
}

type Systemd struct {
	CriticalChain []CriticalChain
	AnalyzeBlame  string
	AllFine       bool
}

type CriticalChain struct {
	Unit      string
	TimeAt    string
	TimeDelay string
}

type K8s struct {
}

type Setup struct {
	MastCount int `json: WorkCount`
	WorkCount int `json: MastCount`
}

const (
	VmwareVarsTempl = `# datastore to use in vSphere
# EXAMPLE:
# vsphere_datastore = "STORAGE-0"
vsphere_datastore = "3PAR"

# datacenter to use in vSphere
# EXAMPLE:
# vsphere_datacenter = "DATACENTER"
vsphere_datacenter = "PROVO"

# network to use in vSphere
# EXAMPLE:
# vsphere_network = "VM Network"
vsphere_network = "VM Network"

# resource pool the machines will be running in
# EXAMPLE:
# vsphere_resource_pool = "CaaSP_RP"
vsphere_resource_pool = "CaaSP_RP"

# template name the machines will be copied from
# EXAMPLE:
# template_name = "SLES15-SP1-cloud-init"
template_name = "lca-sle15-sp1-guestinfo-kd"

# IMPORTANT: Replace by "efi" string in case your template was created by using EFI firmware
firmware = "bios"

# prefix that all of the booted machines will use
# IMPORTANT: please enter unique identifier below as value of
# stack_name variable to not interfere with other deployments
stack_name = "caasp-v4-alexei"

# Number of master nodes
masters = {{.MastCount}}

# Optional: Size of the root disk in GB on master node
master_disk_size = 50

# Number of worker nodes
workers = {{.WorkCount}}

# Optional: Size of the root disk in GB on worker node
worker_disk_size = 40

# Optional: Define the repositories to use
# EXAMPLE:
# repositories = {
#   repository1 = "http://repo.example.com/repository1/"
#   repository2 = "http://repo.example.com/repository2/"
# }
repositories = {       
        suse_ca            = "http://download.suse.de/ibs/SUSE:/CA/SLE_15_SP1/",
        sle_server_pool    = "http://download.suse.de/ibs/SUSE/Products/SLE-Product-SLES/15-SP1/x86_64/product/",
        basesystem_pool    = "http://download.suse.de/ibs/SUSE/Products/SLE-Module-Basesystem/15-SP1/x86_64/product/",
        containers_pool    = "http://download.suse.de/ibs/SUSE/Products/SLE-Module-Containers/15-SP1/x86_64/product/",
        serverapps_pool    = "http://download.suse.de/ibs/SUSE/Products/SLE-Module-Server-Applications/15-SP1/x86_64/product/",
        sle_server_updates = "http://download.suse.de/ibs/SUSE/Updates/SLE-Product-SLES/15-SP1/x86_64/update/",
        basesystem_updates = "http://download.suse.de/ibs/SUSE/Updates/SLE-Module-Basesystem/15-SP1/x86_64/update/",
        containers_updates = "http://download.suse.de/ibs/SUSE/Updates/SLE-Module-Containers/15-SP1/x86_64/update/",
        serverapps_updates = "http://download.suse.de/ibs/SUSE/Updates/SLE-Module-Server-Applications/15-SP1/x86_64/update/",
        caasp_sprint9 = "http://download.suse.de/ibs/SUSE:/Maintenance:/12065/SUSE_SLE-15-SP1_Update_Products_CASP40_Update/",
        caasp_release = "http://download.suse.de/ibs/SUSE:/SLE-15-SP1:/Update:/Products:/CASP40/standard/",
        caasp_update = "http://download.suse.de/ibs/SUSE:/SLE-15-SP1:/Update:/Products:/CASP40:/Update/standard/",
}

# Minimum required packages. Do not remove them.
# Feel free to add more packages
packages = [
  //"patterns-caasp-Node",
  "ca-certificates-suse",
  "kernel-default",
  "-kernel-default-base"
]

# ssh keys to inject into all the nodes
# EXAMPLE:
# authorized_keys = [
#   "ssh-rsa <key-content>"
# ]
authorized_keys = ["ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC2G7k0zGAjd+0LzhbPcGLkdJrJ/LbLrFxtXe+LPAkrphizfRxdZpSC7Dvr5Vewrkd/kfYObiDc6v23DHxzcilVC2HGLQUNeUer/YE1mL4lnXC1M3cb4eU+vJ/Gyr9XVOOReDRDBCwouaL7IzgYNCsm0O5v2z/w9ugnRLryUY180/oIGeE/aOI1HRh6YOsIn7R3Rv55y8CYSqsbmlHWiDC6iZICZtvYLYmUmCgPX2Fg2eT+aRbAStUcUERm8h246fs1KxywdHHI/6o3E1NNIPIQ0LdzIn5aWvTCd6D511L4rf/k5zbdw/Gql0AygHBR/wnngB5gSDERLKfigzeIlCKf Unsafe Shared Key"]

# IMPORTANT: Replace these ntp servers with ones from your infrastructure
ntp_servers = ["0.novell.pool.ntp.org", "1.novell.pool.ntp.org", "2.novell.pool.ntp.org", "3.novell.pool.ntp.org"]`

	OpenstackVarsTempl = `# Name of the image to use
# EXAMPLE:
# image_name = "SLE-15-SP1-JeOS-GMC"
image_name = "SLE-15-SP1-JeOS-GMC"

# Name of the internal network to be created
# EXAMPLE:
# internal_net = "testing"
internal_net = "INGSOC-net-V4"

# Name of the internal subnet to be created
# IMPORTANT: If this variable is not set or empty,
# then it will be generated with schema
# internal_subnet = "${var.internal_net}-subnet"
# EXAMPLE:
# internal_subnet = "testing-subnet"
#internal_subnet = "INGSOC-subnet"
internal_subnet = "INGSOC-subnet"

# Name of the internal router to be created
# IMPORTANT: If this variable is not set or empty,
# then it will be generated with schema
# internal_router = "${var.internal_net}-router"
# EXAMPLE:
# internal_router = "testing-router"
#internal_router = "INGSOC-router"
internal_router = "INGSOC-router-v4"

# Name of the external network to be used, the one used to allocate floating IPs
# EXAMPLE:
# external_net = "floating"
external_net = "floating"

# Identifier to make all your resources unique and avoid clashes with other users of this terraform project
stack_name = "INGSOC"

# CIDR of the subnet for the internal network
# EXAMPLE:
# subnet_cidr = "172.28.0.0/24"
subnet_cidr = "172.28.0.0/24"

# Number of master nodes
masters = {{.MastCount}}

# Number of worker nodes
workers = {{.WorkCount}}

# Size of the master nodes
# EXAMPLE:
# master_size = "m1.medium"
master_size = "m1.medium"

# Size of the worker nodes
# EXAMPLE:
# worker_size = "m1.medium"
worker_size = "m1.medium"

# Attach persistent volumes to workers
workers_vol_enabled = 0

# Size of the worker volumes in GB
workers_vol_size = 5

# Name of DNS domain
# dnsdomain = "my.domain.com"
dnsdomain = ""

# Set DNS Entry (0 is false, 1 is true)
dnsentry = 0

# define the repositories to use
# EXAMPLE:
# repositories = {
#   repository1 = "http://example.my.repo.com/repository1/"
#   repository2 = "http://example.my.repo.com/repository2/"
# }
repositories = {
  suse_ca            = "http://download.suse.de/ibs/SUSE:/CA/SLE_15_SP1/",
  sle_server_pool    = "http://download.suse.de/ibs/SUSE/Products/SLE-Product-SLES/15-SP1/x86_64/product/",
  basesystem_pool    = "http://download.suse.de/ibs/SUSE/Products/SLE-Module-Basesystem/15-SP1/x86_64/product/",
  containers_pool    = "http://download.suse.de/ibs/SUSE/Products/SLE-Module-Containers/15-SP1/x86_64/product/",
  serverapps_pool    = "http://download.suse.de/ibs/SUSE/Products/SLE-Module-Server-Applications/15-SP1/x86_64/product/",
  sle_server_updates = "http://download.suse.de/ibs/SUSE/Updates/SLE-Product-SLES/15-SP1/x86_64/update/",
  basesystem_updates = "http://download.suse.de/ibs/SUSE/Updates/SLE-Module-Basesystem/15-SP1/x86_64/update/",
  containers_updates = "http://download.suse.de/ibs/SUSE/Updates/SLE-Module-Containers/15-SP1/x86_64/update/",
  serverapps_updates = "http://download.suse.de/ibs/SUSE/Updates/SLE-Module-Server-Applications/15-SP1/x86_64/update/",
  caasp_sprint9 = "http://download.suse.de/ibs/SUSE:/Maintenance:/12065/SUSE_SLE-15-SP1_Update_Products_CASP40_Update/",
  caasp_release = "http://download.suse.de/ibs/SUSE:/SLE-15-SP1:/Update:/Products:/CASP40/standard/",
  caasp_update = "http://download.suse.de/ibs/SUSE:/SLE-15-SP1:/Update:/Products:/CASP40:/Update/standard/",
}

# Minimum required packages. Do not remove them.
# Feel free to add more packages
packages = [
  "ca-certificates-suse",
  "kernel-default",
  "-kernel-default-base",
 // "patterns-caasp-Node"
]

# ssh keys to inject into all the nodes
# EXAMPLE:
# authorized_keys = [
#  "ssh-rsa <key-content>"
# ]
authorized_keys = ["ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC2G7k0zGAjd+0LzhbPcGLkdJrJ/LbLrFxtXe+LPAkrphizfRxdZpSC7Dvr5Vewrkd/kfYObiDc6v23DHxzcilVC2HGLQUNeUer/YE1mL4lnXC1M3cb4eU+vJ/Gyr9XVOOReDRDBCwouaL7IzgYNCsm0O5v2z/w9ugnRLryUY180/oIGeE/aOI1HRh6YOsIn7R3Rv55y8CYSqsbmlHWiDC6iZICZtvYLYmUmCgPX2Fg2eT+aRbAStUcUERm8h246fs1KxywdHHI/6o3E1NNIPIQ0LdzIn5aWvTCd6D511L4rf/k5zbdw/Gql0AygHBR/wnngB5gSDERLKfigzeIlCKf Unsafe Shared Key"]

# IMPORTANT: Replace these ntp servers with ones from your infrastructure
ntp_servers = ["0.novell.pool.ntp.org", "1.novell.pool.ntp.org", "2.novell.pool.ntp.org", "3.novell.pool.ntp.org"]`
)
