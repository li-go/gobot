// Code generated by "stringer -type=TaskStatus"; DO NOT EDIT.

package configurablecommand

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Pending-0]
	_ = x[Killed-1]
	_ = x[Running-2]
	_ = x[Succeeded-3]
	_ = x[Failed-4]
}

const _TaskStatus_name = "PendingKilledRunningSucceededFailed"

var _TaskStatus_index = [...]uint8{0, 7, 13, 20, 29, 35}

func (i TaskStatus) String() string {
	if i < 0 || i >= TaskStatus(len(_TaskStatus_index)-1) {
		return "TaskStatus(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _TaskStatus_name[_TaskStatus_index[i]:_TaskStatus_index[i+1]]
}
