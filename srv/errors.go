package srv

import "github.com/ability-sh/abi-lib/errors"

func IsErrno(err error, errno int32) bool {
	e, ok := err.(*errors.Error)
	if ok {
		return e.Errno == errno
	}
	return false
}
