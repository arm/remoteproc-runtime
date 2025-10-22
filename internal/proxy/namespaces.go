package proxy

import (
	"fmt"
	"log/slog"

	"github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/sys/unix"
)

var specNamespacesToUnixCloneFlags = map[specs.LinuxNamespaceType]uintptr{
	specs.CgroupNamespace:  unix.CLONE_NEWCGROUP,
	specs.IPCNamespace:     unix.CLONE_NEWIPC,
	specs.MountNamespace:   unix.CLONE_NEWNS,
	specs.NetworkNamespace: unix.CLONE_NEWNET,
	specs.PIDNamespace:     unix.CLONE_NEWPID,
	specs.TimeNamespace:    unix.CLONE_NEWTIME,
	specs.UserNamespace:    unix.CLONE_NEWUSER,
	specs.UTSNamespace:     unix.CLONE_NEWUTS,
}

func ParseNamespaceFlags(namespaces []specs.LinuxNamespace) (uintptr, error) {
	if namespaces == nil {
		return 0, nil
	}

	var flags uintptr
	for _, ns := range namespaces {
		if ns.Path != "" {
			continue
		}
		flag, ok := specNamespacesToUnixCloneFlags[ns.Type]
		if !ok {
			err := fmt.Errorf("unknown namespace type %q", ns.Type)
			return 0, err
		}
		flags |= flag
	}
	return flags, nil
}

func LinuxCloneFlags(logger *slog.Logger, isRoot bool, namespaces []specs.LinuxNamespace) (uintptr, error) {
	flags, err := ParseNamespaceFlags(namespaces)
	if err != nil {
		return 0, err
	}

	if !isRoot && flags != 0 {
		logger.Warn("running non-root; namespace isolation disabled")
		return 0, nil
	}

	return flags, nil
}
