package schema

func init() {
	NodeRegister = make(map[string]*NodeImplDesc)
	PolicyRegister = make(map[string]*PolicyImplDesc)
	initBaseNodeImplDesc()
	initExpressPointDesc()
	initLeafNodeDesc()
	initPolicies()
	initRdrNodes()
}
