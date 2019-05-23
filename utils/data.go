package utils

// OSAPI holds openstack API variables
type OSAPI struct {
	AuthURL            string
	RegionName         string
	ProjectName        string
	UserDomainName     string
	IdentityAPIVersion string
	Interface          string
	Username           string
	Password           string //[]byte
	ProjectID          string
}

// EnvOS holds as slice with openstack API variables
type EnvOS []string

// CAASPOut is holding caasp terraform output json variables
type CAASPOut struct {
	IPAdminExt   *Admin   `json:"ip_admin_external"`
	IPAdminInt   *Admin   `json:"ip_admin_internal"`
	IPMastersExt Machines `json:"ip_masters"`
	IPWorkersExt Machines `json:"ip_workers"`
}

// SESOut is holding ses terraform output json variables
type SESOut struct {
	K8SSC      Machines `json:"k8s_StorageClass_internal_ip"`
	K8SCS      Machines `json:"ceph_secret"`
	IPAdminExt *Admin   `json:"external_ip_admin"`
	IPAdminInt Machines `json:"internal_ip_admin"`
	IPMonsExt  Machines `json:"external_ip_mons"`
	IPMonsInt  Machines `json:"internal_ip_mons"`
	IPOsdsInt  Machines `json:"internal_ip_osds"`
}

type Admin struct {
	Value string
}

type Machines struct {
	Value []string
}

type CaaSPCluster struct {
	ImageName string
	IntNet    string
	ExtNet    string
	AdmSize   string
	MastSize  string
	MastCount int
	WorkSize  string
	WorkCount int
	DnsDomain string
	DnsEntry  int
	StackName string
	Diff      int //----it is to indicate how many more new nodes you add when appending new nodes to the cluster
}

var CulsterTempl = `image_name = "SUSE-CaaS-Platform-3.0-for-OpenStack-Cloud.x86_64-3.0.0-GM.qcow2"
internal_net = "INGSOC-net"
external_net = "floating"
admin_size = "m1.large"
master_size = "m1.medium"
masters = {{.MastCount}}
worker_size = "m1.medium"
workers = {{.WorkCount}}
workers_vol_enabled = 0
workers_vol_size = 5
dnsdomain = "testing.qa.caasp.suse.net"
dnsentry = 0
stack_name = "INGSOC"`
