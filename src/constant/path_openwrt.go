//go:build openwrt

package constant

const (
	AppConfigDir = "/etc/routex"
	AppShareDir  = "/usr/share/routex"
	AppStateDir  = "/etc/routex/state"
	PIDPath      = "/var/run/routexd.pid"
	SockPath     = "/var/run/routexd.sock"
	PasswdFile   = "/etc/passwd"
	ShadowFile   = "/etc/shadow"
)
