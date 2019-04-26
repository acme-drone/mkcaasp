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
	Password           []byte
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
