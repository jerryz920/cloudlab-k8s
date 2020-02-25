package kvstore

const (
	KvStoreAddress         string = "0.0.0.0:19850"
	MetadataProxyAddress   string = "0.0.0.0:19851"
	BuildingServiceAddress string = "http://0.0.0.0:19852"
	CmdProxyAddress        string = "http://0.0.0.0:19853"
	PrincipalNameLimit     int    = 256
)

var (
	writeTimeout int = 30
)
