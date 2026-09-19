package accessruntime

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultCloudControllerAddress    = "cloud.anytty.com:443"
	defaultCloudControllerServerName = "cloud.anytty.com"
)

// CloudControllerEndpoint 是 Cloud Controller/Edge 的本地连接参数。
type CloudControllerEndpoint struct {
	Address    string
	ServerName string
	CAPEM      []byte
}

// CloudControllerEndpointFromEnvironment 解析 Cloud Controller 连接参数。
// 环境变量缺省时回退官方地址；address 与 serverName 必须成对配置。
func CloudControllerEndpointFromEnvironment() (CloudControllerEndpoint, error) {
	address := strings.TrimSpace(os.Getenv("ANYTTY_CLOUD_CONTROLLER_ADDRESS"))
	serverName := strings.TrimSpace(os.Getenv("ANYTTY_CLOUD_CONTROLLER_SERVER_NAME"))
	caFile := strings.TrimSpace(os.Getenv("ANYTTY_CLOUD_CONTROLLER_CA"))
	if address == "" && serverName == "" && caFile == "" {
		address = defaultCloudControllerAddress
		serverName = defaultCloudControllerServerName
	}
	if address == "" || serverName == "" {
		return CloudControllerEndpoint{}, fmt.Errorf("ANYTTY_CLOUD_CONTROLLER_ADDRESS and ANYTTY_CLOUD_CONTROLLER_SERVER_NAME must be configured together")
	}
	var caPEM []byte
	if caFile != "" {
		var err error
		caPEM, err = os.ReadFile(caFile)
		if err != nil {
			return CloudControllerEndpoint{}, fmt.Errorf("read AnyTTY Cloud Controller CA: %w", err)
		}
	}
	return CloudControllerEndpoint{Address: address, ServerName: serverName, CAPEM: caPEM}, nil
}
