package gogroup

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type vopts struct {
	invalid bool
	verrstr string
	err     bool
}

func testValidate(t *testing.T, g Grouper, opts vopts, imports string) {
	proc := NewProcessor(g)
	text := "package main\n" + imports
	errValid, err := proc.Validate("", strings.NewReader(text))

	if opts.err {
		assert.NotNil(t, err)
	} else {
		assert.Nil(t, err)
	}

	if opts.invalid || opts.verrstr != "" {
		assert.NotNil(t, errValid)
		if opts.verrstr != "" {
			assert.Contains(t, errValid.Error(), opts.verrstr)
		}
	} else {
		assert.Nil(t, errValid)
	}
}

func TestGroupersValidate(t *testing.T) {
	t.Parallel()

	// No imports statement.
	testValidate(t, grouperCombined{}, vopts{}, "")
	testValidate(t, grouperGoimports{}, vopts{}, "")
	testValidate(t, grouperLocalMiddle{}, vopts{}, "")
	testValidate(t, grouperWeird{}, vopts{}, "")

	// Just one import.
	imports := `import "os"`
	testValidate(t, grouperCombined{}, vopts{}, imports)
	testValidate(t, grouperGoimports{}, vopts{}, imports)
	testValidate(t, grouperLocalMiddle{}, vopts{}, imports)
	testValidate(t, grouperWeird{}, vopts{}, imports)

	// Multiple imports in same group, ordered ok.
	imports = `import (
		"os"
		"strings"
		"testing"
	)`
	testValidate(t, grouperCombined{}, vopts{}, imports)
	testValidate(t, grouperGoimports{}, vopts{}, imports)
	testValidate(t, grouperLocalMiddle{}, vopts{}, imports)
	testValidate(t, grouperWeird{}, vopts{}, imports)

	// Multiple imports in same group, ordered poorly.
	imports = `import (
		"strings"
		"os"
	)`
	testValidate(t, grouperCombined{}, vopts{verrstr: errstrStatementOrder}, imports)
	testValidate(t, grouperGoimports{}, vopts{verrstr: errstrStatementOrder}, imports)
	testValidate(t, grouperLocalMiddle{}, vopts{verrstr: errstrStatementOrder}, imports)
	testValidate(t, grouperWeird{}, vopts{verrstr: errstrStatementOrder}, imports)

	// Imports grouped together.
	imports = `import (
		"github.com/Sirupsen/logrus"
		"os"
	)`
	testValidate(t, grouperCombined{}, vopts{}, imports)
	testValidate(t, grouperGoimports{}, vopts{invalid: true}, imports)
	testValidate(t, grouperLocalMiddle{}, vopts{invalid: true}, imports)
	testValidate(t, grouperWeird{}, vopts{invalid: true}, imports)

	// Std/other separated.
	imports = `import (
		"os"

		"github.com/Sirupsen/logrus"
	)`
	testValidate(t, grouperCombined{}, vopts{invalid: true}, imports)
	testValidate(t, grouperGoimports{}, vopts{}, imports)
	testValidate(t, grouperLocalMiddle{}, vopts{}, imports)
	testValidate(t, grouperWeird{}, vopts{}, imports)

	// Std/other separated but backwards.
	imports = `import (
		"github.com/Sirupsen/logrus"

		"os"
	)`
	testValidate(t, grouperCombined{}, vopts{invalid: true}, imports)
	testValidate(t, grouperGoimports{}, vopts{invalid: true}, imports)
	testValidate(t, grouperLocalMiddle{}, vopts{invalid: true}, imports)
	testValidate(t, grouperWeird{}, vopts{invalid: true}, imports)

	// Std/other/local.
	imports = `import (
		"os"

		"github.com/Sirupsen/logrus"

		"local/foo"
	)`
	testValidate(t, grouperCombined{}, vopts{invalid: true}, imports)
	testValidate(t, grouperGoimports{}, vopts{}, imports)
	testValidate(t, grouperLocalMiddle{}, vopts{invalid: true}, imports)
	testValidate(t, grouperWeird{}, vopts{invalid: true}, imports)

	// Std/other/appengine/local.
	imports = `import (
		"os"
		"testing"

		"github.com/Sirupsen/logrus"

		"appengine"

		"local/foo"
	)`
	testValidate(t, grouperCombined{}, vopts{invalid: true}, imports)
	testValidate(t, grouperGoimports{}, vopts{}, imports)
	testValidate(t, grouperLocalMiddle{}, vopts{invalid: true}, imports)
	testValidate(t, grouperWeird{}, vopts{invalid: true}, imports)

	// Local in the middle.
	imports = `import (
		"os"
		"strings"

		"local/bar"
		"local/foo"

		"github.com/Sirupsen/logrus"
		"gopkg.in/redis.v3"
	)`
	testValidate(t, grouperCombined{}, vopts{invalid: true}, imports)
	testValidate(t, grouperGoimports{}, vopts{invalid: true}, imports)
	testValidate(t, grouperLocalMiddle{}, vopts{}, imports)
	testValidate(t, grouperWeird{}, vopts{invalid: true}, imports)

	// Weird ordering, just to prove we can.
	imports = `import (
		"strings"

		"go/parser"
		"gopkg.in/redis.v3"
		"local/pkg"

		"github.com/Sirupsen/logrus"
		"local/foo/bar"
	)`
	testValidate(t, grouperCombined{}, vopts{invalid: true}, imports)
	testValidate(t, grouperGoimports{}, vopts{invalid: true}, imports)
	testValidate(t, grouperLocalMiddle{}, vopts{invalid: true}, imports)
	testValidate(t, grouperWeird{}, vopts{}, imports)
}
