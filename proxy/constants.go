package main

const (
	RIAK_BUCKET_TYPE      = "trustnet"
	RIAK_INDEX_NAME       = "netmap"
	CIDR_BUCKET           = "cidr"
	QUERY_BUCKET          = "_yz_rb"
	QUERY_KEY             = "_yz_rk"
	QUERY_BUCKET_TYPE     = "_yz_rt"
	DEFAULT_VM_TRUST_HUB  = "noauth_vm"
	DEFAULT_CTN_TRUST_HUB = "noauth_ctn"
	VM_INSTANCE_TYPE      = "vm"
	NORMAL_INSTANCE_TYPE  = "normal"

	/// NO need to authenticate an ID prepened with "noauth"
	NOAUTH_PREFIX = "noauth_"
)
