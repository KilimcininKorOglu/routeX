//go:build !entware && !openwrt

package constant

const (
	AppConfigDir = "/etc/routex"
	AppShareDir  = "/usr/share/routex"
	AppStateDir  = "/var/lib/routex"
	PIDPath      = "/var/run/routexd.pid"
	SockPath     = "/var/run/routexd.sock"
	PasswdFile   = "/etc/passwd"
	ShadowFile   = "/etc/shadow"
)
