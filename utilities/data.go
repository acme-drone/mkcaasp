package utilities

// OpenStackAPI holds openstack API variables
type OpenStackAPI struct {
	OSAuthURL            string
	OSRegionName         string
	OSProjectName        string
	OSUserDomainName     string
	OSIdentityAPIVersion string
	OSInterface          string
	OSUsername           string
	OSPassword           string
	OSProjectID          string
}

// EnvOS holds as slice with openstack API variables
type EnvOS []string

// CAASPTFOutput is holding caasp terraform output json variables
type CAASPTFOutput struct {
	CAASPIPAdminExt *Admin   `json:"ip_admin_external"`
	CAASPIPAdminInt *Admin   `json:"ip_admin_internal"`
	IPMastersExt    Machines `json:"ip_masters"`
	IPWorkersExt    Machines `json:"ip_workers"`
}

// SESTFOutput is holding ses terraform output json variables
type SESTFOutput struct {
	K8SStorageClass Machines `json:"k8s_StorageClass_internal_ip"`
	K8SCephSecret   Machines `json:"ceph_secret"`
	SESIPAdminExt   *Admin   `json:"external_ip_admin"`
	SESIPAdminInt   Machines `json:"internal_ip_admin"`
	SESIPMonsExt    Machines `json:"external_ip_mons"`
	SESIPMonsInt    Machines `json:"internal_ip_mons"`
	SESIPOsdsInt    Machines `json:"internal_ip_osds"`
}

type Admin struct {
	Value string
}

type Machines struct {
	Value []string
}
