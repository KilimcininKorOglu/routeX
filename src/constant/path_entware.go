//go:build entware

package constant

const (
	AppConfigDir = "/opt/etc/routex"
	AppShareDir  = "/opt/usr/share/routex"
	AppStateDir  = "/opt/var/lib/routex"
	PIDPath      = "/opt/var/run/routexd.pid"
	SockPath     = "/opt/var/run/routexd.sock"
	PasswdFile   = "/opt/etc/passwd"
	ShadowFile   = "/opt/etc/shadow"
)
