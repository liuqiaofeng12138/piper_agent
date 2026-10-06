package meta

// IFExceptions mirrors MiscRoute.IF_Exceptions.
var IFExceptions = map[string]string{
	"TemplateException":              "one.rewind.nio.distributor.exception.TemplateException",
	"Vars":                           "one.rewind.nio.distributor.exception.TemplateException$Vars",
	"Failed":                         "one.rewind.nio.distributor.exception.AccountException$Failed",
	"Frozen":                         "one.rewind.nio.distributor.exception.AccountException$Frozen",
	"NotLogin":                       "one.rewind.nio.distributor.exception.AccountException$NotLogin",
	"ManualIntervention":             "one.rewind.nio.distributor.exception.AccountException$ManualIntervention",
	"Banned":                         "one.rewind.nio.distributor.exception.ProxyException$Banned",
}
