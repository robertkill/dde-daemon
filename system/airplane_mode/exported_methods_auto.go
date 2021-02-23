// Code generated by "dbusutil-gen em -type Manager"; DO NOT EDIT.

package airplane_mode

import (
	"pkg.deepin.io/lib/dbusutil"
)

func (v *Manager) GetExportedMethods() dbusutil.ExportedMethods {
	return dbusutil.ExportedMethods{
		{
			Name: "DumpState",
			Fn:   v.DumpState,
		},
		{
			Name:   "Enable",
			Fn:     v.Enable,
			InArgs: []string{"enabled"},
		},
		{
			Name:   "EnableBluetooth",
			Fn:     v.EnableBluetooth,
			InArgs: []string{"enabled"},
		},
		{
			Name:   "EnableWifi",
			Fn:     v.EnableWifi,
			InArgs: []string{"enabled"},
		},
	}
}