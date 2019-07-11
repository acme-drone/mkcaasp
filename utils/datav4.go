package utils

type MkcaaspCfg struct {
	Platform  string  `json: "platform"`
	Deploy    string  `json: "deploy"`
	Vmware    *VMWare `json: "vmware"`
	Skubaroot string  `json: skubaroot`
}

type VMWare struct {
	GOVC_URL      string
	GOVC_USERNAME string
	GOVC_PASSWORD string
	GOVC_INSECURE int
	//-------------
	VSPHERE_SERVER               string
	VSPHERE_USER                 string
	VSPHERE_PASSWORD             string
	VSPHERE_ALLOW_UNVERIFIED_SSL bool
}

type TFOutput struct {
	IP_Load_Balancer *TFTag `json: ip_load_balancer`
	IP_Masters       *TFTag `json: ip_masters`
	IP_Workers       *TFTag `json: ip_workers`
}

type TFTag struct {
	Sensitive bool     `json: sensitive`
	Type      string   `json: type`
	Value     []string `json: value`
}

type ClusterCheck map[string]CaaSPv4Node

type CaaSPv4Node struct {
	IP         string
	NodeName   string
	Username   string
	Network    bool
	SSH        bool
	ContHealth bool
	PackHealth bool
	RepoHealth bool
}
