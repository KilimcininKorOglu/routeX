package constant

import (
	"time"

	"routex/models"
)

var DefaultAppConfig = models.AppConfig{
	DNSProxy: models.AppConfigDNSProxy{
		Host:            models.AppConfigDNSProxyServer{Address: "[::]", Port: 3553},
		Upstream:        models.AppConfigDNSProxyServer{Address: "127.0.0.1", Port: 53},
		DisableRemap53:  false,
		DisableFakePTR:  false,
		DisableDropAAAA: false,
		MaxIdleConns:    10,
		MaxConcurrent:   100,
		Timeout:         5000 * time.Millisecond,
		Protocol:        "plain",
	},
	HTTPWeb: models.AppConfigHTTPWeb{
		Enabled: true,
		Auth: models.AppConfigAuth{
			Enabled: true,
		},
		Host: models.AppConfigHTTPWebServer{
			Address: "[::]",
			Port:    7080,
		},
		Skin:     "default",
		Language: "en",
	},
	Netfilter: models.AppConfigNetfilter{
		IPTables: models.AppConfigIPTables{
			ChainPrefix: "MT_",
		},
		IPSet: models.AppConfigIPSet{
			TablePrefix:   "mt_",
			AdditionalTTL: 3600,
		},
		DisableIPv4:         false,
		DisableIPv6:         false,
		StartMarkTableIndex: 0x4D616769, // Magi
	},
	Link:              []string{"br0"},
	ShowAllInterfaces: false,
	LogLevel:          "info",
}

var (
	Version = "unattached"
)
