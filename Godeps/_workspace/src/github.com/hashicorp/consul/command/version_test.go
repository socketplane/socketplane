package command

import (
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/mitchellh/cli"
	"testing"
)

func TestVersionCommand_implements(t *testing.T) {
	var _ cli.Command = &VersionCommand{}
}
