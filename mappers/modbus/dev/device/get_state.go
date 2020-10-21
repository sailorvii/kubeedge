package device

import (
	mappercommon "github.com/kubeedge/kubeedge/mappers/common"
	"github.com/kubeedge/kubeedge/mappers/modbus/dev"
	"github.com/kubeedge/kubeedge/mappers/modbus/globals"
	"strconv"
)

type GetState struct {
	Client globals.ModbusClient
	State  dev.DevStatus
	topic  string
}

func (gs *GetState) Run() {
	gs.State = dev.GetStatus(gs.Client)

	var payload []byte
	payload = mappercommon.CreateMessageState(strconv.Itoa(int(gs.State)))
	globals.MqttClient.Publish(gs.topic, payload)
}
