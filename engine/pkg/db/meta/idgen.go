package meta

import (
	"piper_go/pkg/util"
)

func AssignID(table string, doc map[string]any) {
	if id, _ := doc["id"].(string); id != "" {
		return
	}
	name := strField(doc, "name")
	version := strField(doc, "version")
	if version == "" {
		version = "1"
	}
	switch table {
	case TableIndices:
		doc["id"] = util.MD5Hex(name + "::" + version)
	case TableFunctions:
		doc["id"] = util.MD5Hex(name + "::" + strField(doc, "url"))
	case TableTemplates:
		doc["id"] = util.MD5Hex("Template::" + name + "::" + version)
	case TableTasks:
		doc["id"] = util.MD5Hex(strField(doc, "tpl_id") + "::" + name)
	case TableVarsLists:
		doc["id"] = util.MD5Hex("VarsList::" + name + "::" + version)
	case TableAccounts:
		doc["id"] = util.MD5Hex(strField(doc, "domain") + "::" + strField(doc, "username"))
	case TableProxies:
		doc["id"] = util.MD5Hex(strField(doc, "group_") + "::" + strField(doc, "ssh_host") + "::" + strField(doc, "port"))
	case TableNodes:
		if inst := strField(doc, "inst_id"); inst != "" {
			doc["id"] = inst
		}
	}
}
