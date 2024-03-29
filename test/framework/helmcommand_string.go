// Code generated by "stringer -type=HelmCommand all_type_helpers.go"; DO NOT EDIT.

package framework

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Install-0]
	_ = x[Uninstall-1]
	_ = x[Repo-2]
	_ = x[Template-3]
	_ = x[Add-4]
	_ = x[Update-5]
	_ = x[Remove-6]
}

const _HelmCommand_name = "InstallUninstallRepoTemplateAddUpdateRemove"

var _HelmCommand_index = [...]uint8{0, 7, 16, 20, 28, 31, 37, 43}

func (i HelmCommand) String() string {
	if i < 0 || i >= HelmCommand(len(_HelmCommand_index)-1) {
		return "HelmCommand(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _HelmCommand_name[_HelmCommand_index[i]:_HelmCommand_index[i+1]]
}
